# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Nomi CLI is a command-line interface tool for interacting with the Nomi.ai API, allowing users to manage and chat with their Nomis directly from the terminal. The CLI is built in Go using the Cobra framework.

## Core Architecture

- **Main Application Structure**: Uses Cobra for CLI command management
- **API Client**: Centralized `NomiClient` in `client.go` handles all HTTP communication
  - Structured error handling with `APIError` type
  - 30-second timeout configuration
  - Reusable request patterns with authentication headers
  - Methods: `GetNomis()`, `GetNomi()`, `GetRooms()`, `SendMessage()`, `FindNomiByName()`
- **Authentication**: Uses API key in environment variable or passed as a flag
- **Interactive TUI**:
  - Selectable Menu: When run without arguments, displays a styled, navigable menu of Nomis using BubbleTea
  - Chat Interface: Interactive chat with proper line editing (using readline library)
- **User Experience**:
  - Color-coded interface with consistent styling across components
  - Arrow key navigation in both menu selection and chat input
  - Command history in chat sessions
- **Commands**:
  - `list-nomis`: Displays all Nomis (with optional --full flag)
  - `get-nomi`: Gets details for a specific Nomi by ID
  - `chat`: Interactive chat session with a Nomi (by name)
  - `list-rooms`: Lists chat rooms
  - `version`: Shows the CLI version

## Common Development Commands

### Building the CLI

```bash
# Build the application
make build

# Or manually with version tagging
go build -ldflags "-X main.Version=$(git describe --tags --always --dirty)" -o nomi-cli
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run tests with terminal coverage output
make test-coverage-text

# Run a specific test
go test -v -run TestFunctionName
```

### Installation

```bash
# Install to $GOPATH/bin
make install

# Clean build artifacts
make clean
```

### Running Locally

```bash
# Run directly without building
go run main.go [COMMAND]

# Example: List Nomis
go run main.go list-nomis

# Example: Chat with Nomi
go run main.go chat NomiName
```

## API Configuration

The CLI requires configuration for the Nomi.ai API:

- API Key: Set via `NOMI_API_KEY` environment variable or `-k/--api-key` flag
- API URL: Set via `NOMI_API_URL` environment variable (defaults to "https://api.nomi.ai/v1")

Example:
```bash
export NOMI_API_KEY=your_api_key_here
export NOMI_API_URL=https://api.nomi.ai/v1
```

## API Client Architecture

The application uses a centralized API client pattern implemented in `client.go`:

### NomiClient Structure
- **Initialization**: Created once in `main.go` during `PersistentPreRunE` and stored as global `client` variable
- **HTTP Client**: Reuses single HTTP client instance with 30-second timeout
- **Error Handling**: Returns structured `APIError` with status codes and descriptive messages
- **Authentication**: Automatically adds Bearer token headers to all requests

### Key Methods
```go
// Get all Nomis
nomis, err := client.GetNomis()

// Get specific Nomi by ID
nomi, err := client.GetNomi(id)

// Get all rooms
rooms, err := client.GetRooms()

// Send chat message
response, err := client.SendMessage(nomiID, message)

// Find Nomi UUID by name (case-insensitive)
uuid, err := client.FindNomiByName(name)
```

### Testing Notes
- Tests must initialize the global `client` variable with a mock server
- Example test setup:
```go
baseURL = server.URL
apiKey = "test-api-key"
client = NewNomiClient(apiKey, baseURL)
```
- Error message format: `"API error (404 Not Found): Request to /endpoint failed"`

## User Interface Components

### Selectable Menu
- Implemented using BubbleTea library
- Navigate with arrow keys or vim-style j/k keys
- Press Enter to select a Nomi and start chat
- Press q to quit

### Chat Interface
- Uses readline library for input handling
- Features:
  - Full arrow key navigation within text input
  - In-memory command history with up/down arrows (session only, not persisted)
  - Common keyboard shortcuts (Ctrl+A for start of line, Ctrl+E for end of line)
- Colors used consistently across components (colorYellow for titles, colorBlue for instructions, colorGreen for user text)