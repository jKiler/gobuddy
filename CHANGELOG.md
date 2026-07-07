# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2026-07-07

gobuddy v2 reshapes the project from "an MCP server with two tools" into a
Claude Code plugin for Go development — skills for knowledge, hooks for
enforcement, and a slim MCP core for computation — while remaining fully
usable as a standalone MCP server. See `doc/design.md` for the full
assessment and plan.

### Added

- `gocheck` MCP tool: runs `gofmt -l`, `go vet`, and `go test -json` for a
  target module in one call and returns a compact structured report
  (unformatted files, vet issues, failing tests with truncated output
  tails).
- Claude Code plugin packaging (`.claude-plugin/plugin.json` and
  `marketplace.json`): install with `/plugin marketplace add jKiler/gobuddy`.
- `go-standards` skill carrying the project's Go coding standards, loaded
  on demand by skill-aware hosts.
- PostToolUse hook (`hooks/gofmt-check.sh`) that fails with feedback when
  an edited `.go` file is not gofmt-clean or its package does not vet.
- `/gobuddy:check` slash command: run the quality gate, fix what it
  reports, repeat until clean.
- GitHub Actions CI and `make check` (fmt check, vet, build, tests);
  `make release-check` adds plugin manifest and hook validation.
- MCP-surface integration tests over the SDK's in-memory transport,
  pinning the exact exposed tool set.
- `internal/run`: shared subprocess helper bounding every external command
  with a 30-second timeout.

### Changed

- `godoc` is module-aware: new `working_dir` input runs `go doc` in the
  target module so dependency versions resolve correctly, and `mode`
  selects `doc` (default), `all`, or `src`.
- Expected tool failures return `CallToolResult{IsError: true}` instead of
  protocol errors, following current MCP SDK conventions.
- MCP SDK upgraded from v1.1.0 to v1.6.1; all tools carry
  `ToolAnnotations`; the server version derives from build info instead of
  a hardcoded string.
- README leads with the plugin install path; standalone MCP usage remains
  documented.

### Removed

- `standards` MCP tool. Coding-standards distribution is handled natively
  by agent hosts (project instructions, skills, built-in review commands);
  the content now ships as the `go-standards` skill. See "Migrating from
  v1" in the README.

## [1.0.0]

### Added

- Initial MCP server with `godoc` and `standards` tools.
