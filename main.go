package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/andygrunwald/go-jira"
	glean "github.com/gleanwork/api-client-go"
	"github.com/gleanwork/api-client-go/models/components"
	"github.com/joho/godotenv"
)

const (
	jiraTokenKey        = "JIRA_TOKEN"
	jiraBaseURL         = "https://happyreturns.atlassian.net"
	jiraTicketURLPrefix = "https://happyreturns.atlassian.net/browse/"
	jiraUsername        = "zhao.liu@happyreturns.com"
	jiraSprint          = "CF - On Deck"

	gleanTokenKey = "GLEAN_TOKEN"
	gleanInstance = "happyreturns"
	gleanAgentID  = "0ac7bf1977574596a4a3ea410e364d4c"
)

func main() {
	// Load .env file for API tokens
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Load environment variables
	jiraToken := os.Getenv(jiraTokenKey)
	gleanToken := os.Getenv(gleanTokenKey)

	// Check if tokens are loaded
	if jiraToken == "" {
		log.Fatalf("Warning: %s environment variable is not set\n", jiraTokenKey)
	}

	if gleanToken == "" {
		log.Fatalf("Warning: %s environment variable is not set\n", gleanTokenKey)
	}

	jiraClient := getJiraClient(jiraToken)
	gleanClient := getGleanClient(gleanToken)

	ctx := context.Background()

	jql := fmt.Sprintf("status = 'To Do' AND Sprint = '%s' AND Flagged is EMPTY", jiraSprint)
	issues, err := searchJiraIssues(jiraClient, jql)
	if err != nil {
		log.Fatalf("Error searching Jira issues: %v", err)
	}

	for _, issue := range issues {
		issueLink := fmt.Sprintf("%s%s", jiraTicketURLPrefix, issue.Key)
		log.Printf("Processing issue: %s", issueLink)

		// Glean seems to have rate limit over agent usage, so we might need to add a sleep here
		// time.Sleep(30 * time.Second)
		comments, err := getCleanPointingComments(ctx, gleanClient, issueLink)
		if err != nil {
			log.Fatalf("Error getting clean pointing comments: %v", err)
		}

		for _, comment := range comments {
			err := postJiraComment(jiraClient, issue.ID, comment)
			if err != nil {
				log.Printf("Error posting Jira comment: %v", err)
			}
		}
	}
	log.Printf("Done")
}

func getJiraClient(jiraToken string) *jira.Client {
	tp := jira.BasicAuthTransport{
		Username: jiraUsername,
		Password: jiraToken,
	}

	jiraClient, err := jira.NewClient(tp.Client(), jiraBaseURL)
	if err != nil {
		panic(err)
	}
	return jiraClient
}

func getGleanClient(gleanToken string) *glean.Glean {
	client := &http.Client{
		Timeout: 900 * time.Second,
	}

	s := glean.New(
		glean.WithSecurity(gleanToken),
		glean.WithInstance(gleanInstance),
		glean.WithClient(client),
		glean.WithTimeout(900*time.Second),
	)
	return s
}

func searchJiraIssues(jiraClient *jira.Client, jql string) ([]jira.Issue, error) {
	issues, _, err := jiraClient.Issue.Search(jql, nil)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func postJiraComment(jiraClient *jira.Client, issueID string, comment string) error {
	_, _, err := jiraClient.Issue.AddComment(issueID, &jira.Comment{
		Body: comment,
	})
	if err != nil {
		return err
	}
	return nil
}

func getCleanPointingComments(ctx context.Context, gleanClient *glean.Glean, issueLink string) ([]string, error) {
	comments := []string{}

	res, err := gleanClient.Client.Agents.Run(ctx, components.AgentRunCreate{
		AgentID: gleanAgentID,
		Input: map[string]any{
			"Ticket": issueLink,
		},
	})
	if err != nil {
		return nil, err
	}
	for _, message := range res.GetAgentRunWaitResponse().GetMessages() {
		if *message.GetRole() == "GLEAN_AI" {
			content := message.GetContent()
			if len(content) == 0 {
				return nil, fmt.Errorf("content is empty")
			}

			for _, contentItem := range content {
				if contentItem.GetType() == "text" {
					comments = append(comments, contentItem.GetText())
				}
			}
			break
		}
	}

	return comments, nil
}
