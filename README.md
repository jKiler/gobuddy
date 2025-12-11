# gobuddy

MCP server for Go development assistance.

## Installation

```bash
make build
```

## Usage

```bash
./bin/gobuddy
```

Configure in your MCP client (e.g., Claude Desktop):

```json
{
  "mcpServers": {
    "gobuddy": {
      "command": "/path/to/bin/gobuddy"
    }
  }
}
```

## Tools

### `godoc`

Get Go documentation for packages and symbols.

```json
{"package": "fmt", "symbol": "Printf"}
```

### `standards`

Aggregate coding standards from local files and git repositories.

```json
{
  "sources": [
    {"type": "local", "location": ".", "files": ["CODING_STANDARDS.md"], "priority": 1},
    {"type": "git", "location": "git@github.com:org/standards.git", "files": ["STANDARDS.md"], "priority": 10}
  ]
}
```

Features: priority-based ordering, git caching (2 weeks), SSH auth support.

## Development

```bash
make build   # Build binaries
make test    # Run tests
make fmt     # Format code
make clean   # Clean build artifacts
```
