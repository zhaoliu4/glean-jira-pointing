# Glean Jira Pointing Tool

This Go application automates the process of adding story point estimates to Jira tickets using Glean AI agents. It searches for tickets in a specific sprint and uses a Glean agent to generate pointing comments.

## Features

- Automatically finds unflagged Jira tickets in "To Do" status within a specified sprint
- Uses Glean AI agent to generate story point estimates
- Posts pointing comments directly to Jira tickets
- Tracks completed tickets to avoid reprocessing

## Prerequisites

- Go 1.24.1 or later
- Jira API access token
- Glean API access token
- Access to the specified Jira instance and Glean workspace

## Setup

### 1. Clone the Repository

```bash
git clone <repository-url>
cd glean-jira-pointing
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Environment Configuration

Create a `.env` file in the root directory with your API tokens:

```bash
# Jira API Configuration
JIRA_TOKEN=your_jira_api_token_here

# Glean API Configuration  
GLEAN_TOKEN=your_glean_api_token_here
```

### 4. Configuration Constants

The application uses several constants that you may need to modify in `main.go`:

```go
const (
    // Jira Configuration
    jiraBaseURL         = "https://happyreturns.atlassian.net"
    jiraUsername        = "zhao.liu@happyreturns.com"
    jiraSprint          = "CF - On Deck"
    
    // Glean Configuration
    gleanInstance = "happyreturns"
    gleanAgentID  = "0ac7bf1977574596a4a3ea410e364d4c"
)
```

**Key Configuration Points:**

- **`jiraBaseURL`**: Your Jira instance URL
- **`jiraUsername`**: Your Jira username/email
- **`jiraSprint`**: The sprint name to search for tickets (currently "CF - On Deck")
- **`gleanInstance`**: Your Glean workspace instance name
- **`gleanAgentID`**: The ID of the Glean agent that generates pointing estimates

### 5. Customize for Your Environment

Update the constants in `main.go` to match your setup:

1. **Jira Configuration:**
   - Update `jiraUsername` to your Jira email/username
   - Modify `jiraSprint` to target your specific sprint

2. **Glean Configuration:**
   - Replace `gleanAgentID` with your specific agent ID

## Usage

### Running the Application

```bash
go run main.go
```

The application will:

1. Load environment variables from `.env`
2. Connect to Jira and Glean APIs
3. Search for unflagged tickets in "To Do" status within the specified sprint
4. Skip tickets that have already been processed (tracked in `completed_tickets.txt`)
5. Use the Glean agent to generate pointing estimates
6. Post comments to Jira tickets
7. Track completed tickets in `completed_tickets.txt` to avoid reprocessing

## How It Works

1. **Ticket Discovery**: Uses JQL query to find tickets in "To Do" status within the specified sprint
2. **Glean Integration**: Sends ticket URLs to a Glean agent for story point estimation
3. **Comment Generation**: The Glean agent analyzes the ticket and generates pointing comments
4. **Jira Integration**: Posts the generated comments directly to the Jira tickets
5. **Progress Tracking**: Maintains a list of completed tickets to prevent reprocessing

## Error Handling

- Validates that required environment variables are set
- Logs errors for failed comment postings
- Continues processing other tickets even if one fails
- Tracks completed tickets to maintain progress

## Dependencies

- `github.com/andygrunwald/go-jira`: Jira API client
- `github.com/gleanwork/api-client-go`: Glean API client  
- `github.com/joho/godotenv`: Environment variable loading
