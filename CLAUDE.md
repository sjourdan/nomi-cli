# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Nomi CLI is a command-line interface tool for interacting with the Nomi.ai API, allowing users to manage and chat with their Nomis directly from the terminal. The CLI is built in Go using the Cobra framework.

## Core Architecture

- **Main Application Structure**: Uses Cobra for CLI command management
- **API Client**: Centralized `NomiClient` in `client.go` handles all HTTP communication
  - Structured error handling with `APIError` type
  - Per-request timeouts via `context`: 30s for standard calls, 120s for chat
    (the `/chat` endpoint triggers LLM generation and is slow). Overridable
    with `NOMI_API_TIMEOUT` / `NOMI_CHAT_TIMEOUT` (seconds or Go durations).
  - Chat requests retry transient failures (timeouts, 5xx) with exponential backoff
  - Reusable request patterns with authentication headers
  - Methods: `GetNomis()`, `GetNomi()`, `GetRooms()`, `SendMessage()`, `FindNomiByName()`
- **Authentication**: Uses API key in environment variable or passed as a flag
- **Interactive TUI** (`tui.go`):
  - A single full-screen BubbleTea program for chatting and switching Nomis
  - Built from `bubbles` components: `viewport` (transcript), `textarea`
    (composer), `textinput` (switcher filter), `spinner` (typing indicator)
  - Nomi switcher: press Tab to open a centered modal list with substring
    filtering; selecting a Nomi swaps the conversation without leaving the app
  - Per-Nomi conversations are kept in memory, so switching back and forth
    preserves each transcript (not persisted to disk)
  - Sends are async (`tea.Cmd`): the UI never blocks, and replies are routed
    by Nomi ID so they land correctly even after switching
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
- Timeouts (optional): `NOMI_API_TIMEOUT` (default 30s) and `NOMI_CHAT_TIMEOUT` (default 120s); accepts plain seconds (`180`) or Go durations (`3m`)

Example:
```bash
export NOMI_API_KEY=your_api_key_here
export NOMI_API_URL=https://api.nomi.ai/v1
```

## API Client Architecture

The application uses a centralized API client pattern implemented in `client.go`:

### NomiClient Structure
- **Initialization**: Created once in `main.go` during `PersistentPreRunE` and stored as global `client` variable
- **HTTP Client**: Reuses a single HTTP client instance; each request is bounded by its own `context` timeout (30s default, 120s for chat)
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

The interactive TUI lives entirely in `tui.go` as the `chatTUI` BubbleTea model.
Running `nomi-cli` with no arguments opens it with the switcher in front;
`nomi-cli chat <name>` opens it focused on a specific Nomi.

### Key bindings
- **Tab**: open/close the Nomi switcher modal
- **Enter**: send the composed message (chat) or pick the highlighted Nomi (switcher)
- **↑/↓**: move the switcher selection; **PgUp/PgDn**: scroll the transcript
- **Esc**: cancel the switcher
- **Ctrl+C**: quit

### Layout
- Header bar (active Nomi + relationship), scrollable `viewport` transcript,
  bordered `textarea` composer, and a footer hint line
- Messages render as aligned, color-coded bubbles via `lipgloss` (user right,
  Nomi left, errors highlighted)

### Testing the TUI
- `chatTUI` has a value receiver; drive it in tests by feeding `tea.Msg`
  values to `Update` and asserting on the returned model
- Send a `tea.WindowSizeMsg` first (see the `sized` test helper) so layout is
  initialized before exercising key handling or `View()`