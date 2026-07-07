# Go Development Guidelines

All code contributed to this project must follow the project's Go coding
standards defined in [skills/go-standards/SKILL.md](skills/go-standards/SKILL.md)
(formatting with `gofmt`, naming conventions, explicit error handling with
wrapped context, simplicity over cleverness, doc comments on all exported
identifiers, table-driven tests).

Run `make check` (fmt check, vet, build, tests) before committing; it is the
same gate CI enforces.

# Agent Guidelines
- **Reading URLs:** ALWAYS read URLs provided by the user. They are not optional.
- **Go Documentation:** ALWAYS use the `godoc` tool to retrieve documentation about Go packages or symbols. Prefer `godoc` over `WebFetch` and `WebSearch`. Only use web-based tools when `godoc` doesn't provide a clear answer.
