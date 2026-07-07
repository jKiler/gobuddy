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

## Skills

### `go-standards`

Go coding standards (formatting, naming, error handling, testing) shipped as
an [Agent Skill](skills/go-standards/SKILL.md), loaded on demand by
skill-aware hosts such as Claude Code.

## Migrating from v1

The `standards` MCP tool was removed. Coding-standards distribution is now
handled natively by agent hosts: project instructions (`CLAUDE.md`), the
`go-standards` skill in this repo, and built-in review commands. If you
aggregated standards from a git repository, copy them into a skill or your
project's `CLAUDE.md` instead.

## Development

```bash
make build   # Build binaries
make test    # Run tests
make fmt     # Format code
make clean   # Clean build artifacts
```
