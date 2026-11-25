# Nomi CLI

The `Nomi CLI` is a command-line tool for interacting with the [Nomi.ai API](https://api.nomi.ai/), allowing users to manage and chat with their Nomis directly from the terminal. This tool is designed for developers and enthusiasts who want to seamlessly integrate Nomi.ai's capabilities into their workflows.

![Nomi CLI Screenshot](./docs/images/nomi-cli-005-screenshot.png)

## Features

- **List Nomis**:

  - Display a minimal list of Nomis (Name and Relationship).
  - Use the `--full` flag to view detailed information.

- **Get Nomi Details**:

  - Retrieve detailed information about a specific Nomi by ID.

- **Group Chat Rooms**:

  - List, create, edit, and delete rooms, and chat in an existing room.

- **Chat with Nomis**:
  - Start a live, interactive full-screen chat session with a Nomi.
  - Specify the Nomi by name instead of ID for ease of use.
  - Press **Tab** mid-conversation to open a Nomi switcher and jump to a
    different Nomi without leaving the app. Each conversation is kept in
    memory while the app is open, so you can switch back and forth freely.

## Requirements

- Go 1.19 or later.
- Access to the Nomi.ai API with a valid API key.

## Installation

1. Clone the repository:

```bash
git clone https://github.com/sjourdan/nomi-cli.git
cd nomi-cli
```

1. Build the CLI:

```bash
go build -o nomi-cli
```

3. Add the CLI to your PATH (optional):

```bash
sudo mv nomi-cli /usr/local/bin/nomi
```

## Configuration

Environment Variables

- `NOMI_API_KEY`: Your API key for authenticating requests.
- `NOMI_API_URL` (optional): Base URL of the Nomi.ai API. Defaults to https://api.nomi.ai/v1.

Set these variables in your shell (or use a `.env` file):

```bash
export NOMI_API_KEY=your_api_key_here
export NOMI_API_URL=https://api.nomi.ai/v1
```

## Usage

### Commands

1. List Nomis

Displays all your Nomis.

- Minimal Output:

```bash
nomi list-nomis
```

- Full Output:

```bash
nomi list-nomis --full
```

2. Get Nomi Details

Retrieve detailed information about a specific Nomi by ID.

```bash
nomi get-nomi <nomi-id>
```

Example:

```bash
nomi get-nomi 123e4567-e89b-12d3-a456-426614174000
```

3. Chat with Nomis

Start a live, interactive full-screen chat session with a Nomi.

```bash
./nomi-cli chat [NOMI_NAME]
```

Example:

```bash
./nomi-cli chat John
```

Running `nomi-cli` with no arguments opens the same chat UI with the Nomi
switcher in front, ready to pick whom to talk to.

Key bindings inside the chat:

- **Enter** — send the message you typed
- **Tab** — open the Nomi switcher (type to filter, ↑/↓ to move, Enter to pick)
- **PgUp / PgDn** — scroll the conversation history
- **Esc** — close the switcher
- **Ctrl+C** — quit

4. List and Create Rooms

Use `list-nomis` to see available names and repeat `--nomi` for each participant. Name matching ignores case.

```bash
nomi list-rooms
nomi create-room "Book club" --nomi 'Courtney Love' --nomi 'Alice' --note "Bring a book"
```

A room takes 1 to 10 Nomis. Add `--backchanneling` to enable backchanneling. The command prints the created room's UUID and status.

If two Nomis share a name, use the participant's UUID from `list-nomis --full`.

5. Join an Existing Room

Use the room name. Name matching ignores case. If several rooms share a name, use the UUID from `list-rooms`.

```bash
nomi join-room 'Book club'
```

Press **Tab** to choose who replies first. **Enter** posts your message and starts a turn of up to five Nomi replies total. The CLI randomly chooses each subsequent speaker from the other participants, then waits for your next message. The first participant is selected when the room opens, and the transcript shows messages from the current CLI session.

6. Edit an Existing Room

Use the room name or UUID and pass the settings you want to change.

```bash
nomi edit-room 'Book club' --name 'Weekend readers' --note 'Pick a new book each week'
nomi edit-room 'Weekend readers' --nomi 'Courtney Love' --nomi 'Alice'
```

`--nomi` replaces the full participant list. Use `--note ''` to clear the note, `--backchanneling` to enable backchanneling, or `--backchanneling=false` to disable it. If room names are ambiguous, use the UUID from `list-rooms`.

7. Delete a Room

```bash
nomi delete-room 'Book club'
```

Use a room name or the UUID from `list-rooms`. If several rooms share a name, use the UUID to select one.

### Help

To see a list of available commands and options:

```bash
nomi --help
```

## Example Workflow

1. List your Nomis:

```bash
./nomi-cli list-nomis
```

Output:

```bash
John (Mentor)
Jane (Friend)
```

2. Start a chat with John:

```bash
./nomi-cli chat John
```

Chat session:

```bash
Chat session started with Alice. Type your message and press Enter to send.
Type 'exit' to end the session.
You: Hello, Alice!
Nomi: Hi there! How can I help you today?
```

## Development

### Running Locally

You can run the CLI directly from the source directory without building:

```bash
go run main.go [COMMAND]
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any enhancements, bug fixes, or documentation updates.

## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Acknowledgments

- Nomi.ai for their powerful API.
- The Cobra CLI framework for simplifying CLI development.
