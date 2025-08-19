package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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

	completedTicketsFile = "completed_tickets.txt"
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

	// Load the list of completed tickets so we don't process them again
	completedTickets, err := getCompletedTickets()
	if err != nil {
		log.Fatalf("Error getting completed tickets: %v", err)
	}
	log.Printf("Completed tickets: %v", completedTickets)

	ctx := context.Background()

	jql := fmt.Sprintf("status = 'To Do' AND Sprint = '%s' AND Flagged is EMPTY", jiraSprint)
	issues, err := searchJiraIssues(jiraClient, jql)
	if err != nil {
		log.Fatalf("Error searching Jira issues: %v", err)
	}

	for _, issue := range issues {
		if _, ok := completedTickets[issue.Key]; ok {
			log.Printf("Skipping completed ticket: %s", issue.Key)
			continue
		}

		issueLink := fmt.Sprintf("%s%s", jiraTicketURLPrefix, issue.Key)
		log.Printf("Processing issue: %s", issueLink)

		// Glean seems to have rate limit over agent usage, so we might need to add a sleep here
		time.Sleep(10 * time.Second)
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

		err = updateAIEstimatedField(jiraClient, issue.Key)
		if err != nil {
			log.Printf("Error updating AI Estimated field: %v", err)
		}

		err = saveCompletedTickets(issue.Key)
		if err != nil {
			log.Printf("Error saving completed ticket %s: %v", issue.Key, err)
		}
	}
	log.Printf("Done")
}

func getCompletedTickets() (map[string]struct{}, error) {
	completedTickets := make(map[string]struct{})

	file, err := os.Open(completedTicketsFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			completedTickets[line] = struct{}{}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return completedTickets, nil
}

func saveCompletedTickets(key string) error {
	file, err := os.OpenFile(completedTicketsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(key + "\n")
	if err != nil {
		return err
	}
	return nil
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

// updateAIEstimatedField updates the AI Estimated customfield (customfield_11728) for a given issue
func updateAIEstimatedField(jiraClient *jira.Client, issueID string) error {
	// Create the issue update structure
	issue := &jira.Issue{
		Key: issueID,
		Fields: &jira.IssueFields{
			Unknowns: map[string]interface{}{
				"customfield_11728": map[string]interface{}{
					"value": "AI Estimated ",
				},
			},
		},
	}

	// Update the issue
	_, _, err := jiraClient.Issue.Update(issue)
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
