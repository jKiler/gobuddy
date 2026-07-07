# gobuddy

Go coding buddy for AI agents: a Claude Code plugin (skills, hooks, slash
command) built around a standalone MCP server with module-aware Go tools.

## Install as a Claude Code plugin (recommended)

```
/plugin marketplace add jKiler/gobuddy
/plugin install gobuddy@gobuddy
```

Then build the MCP server binary once inside the installed plugin directory:

```bash
make build
```

The plugin bundles:

- **MCP tools** — `godoc` and `gocheck` (see below)
- **`go-standards` skill** — Go idioms loaded on demand while writing or
  reviewing Go code
- **PostToolUse hook** — after any edit to a `.go` file, checks `gofmt -l`
  and `go vet` on the touched package and feeds violations back immediately
- **`/gobuddy:check` command** — runs the quality gate and fixes what it
  reports

## Install as a standalone MCP server

Any MCP client can use the server directly:

```bash
make build
```

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

Module-aware Go documentation for packages and symbols. Set `working_dir`
to the target module so dependency versions resolve correctly; `mode`
selects `doc` (default), `all` (full package docs), or `src` (symbol
source).

```json
{"package": "fmt", "symbol": "Printf", "working_dir": "/path/to/project", "mode": "doc"}
```

### `gocheck`

One-call quality gate: runs `gofmt -l`, `go vet`, and `go test` for a
module and returns a compact structured report — unformatted files, vet
issues, and failing tests with truncated output tails.

```json
{"working_dir": "/path/to/project", "packages": ["./..."], "include_tests": true}
```

## Skills

### `go-standards`

Go coding standards (formatting, naming, error handling, testing) shipped as
an [Agent Skill](skills/go-standards/SKILL.md), loaded on demand by
skill-aware hosts such as Claude Code.

## Architecture

Each layer owns what it does best — MCP for computation, skills for
knowledge, hooks for guaranteed enforcement:

```
.claude-plugin/        plugin + marketplace manifests
skills/go-standards/   Go idioms as an on-demand Agent Skill
hooks/                 PostToolUse gofmt/vet enforcement
commands/              /gobuddy:check quality-gate loop
cmd/gobuddy/           stdio MCP server (standalone-capable)
tools/                 MCP tools: godoc, gocheck
internal/run/          subprocess execution with timeouts
```

The full design rationale and milestone history live in
[doc/design.md](doc/design.md).

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
make check   # Run the CI gate (fmt check, vet, build, tests)
make clean   # Clean build artifacts
```
