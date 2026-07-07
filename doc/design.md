# gobuddy — Modernization Design & Implementation Plan

**Status:** Proposed
**Last updated:** 2026-07-07
**Vision:** gobuddy stays a *Go coding buddy* — but it grows from "an MCP server
with two tools" into a **Claude Code plugin for Go development**: skills that
teach, hooks that enforce, and a slim MCP core that computes what an LLM
shouldn't guess.

---

## 1. Where we are

The repo today is a stdio MCP server (`cmd/gobuddy`) exposing two tools:

| Tool | What it does | Verdict |
|------|--------------|---------|
| `godoc` | Wraps `go doc <pkg> [symbol]` | **Keep & deepen.** Genuinely useful, but currently adds little over running `go doc` in a shell. |
| `standards` | Aggregates coding-standards markdown from local files and git repos, with priority ordering and a 2-week clone cache | **Retire.** This problem is now solved natively by the host. |

### 1.1 What the ecosystem changed underneath us

When gobuddy was written, "how do I feed my team's coding standards to the
agent?" was an open problem worth an MCP tool. It no longer is:

- **CLAUDE.md** is loaded automatically per-project and per-user; this repo
  already has one that duplicates `CODING_STANDARDS.md`.
- **Agent Skills** (`SKILL.md`) are the idiomatic way to ship procedural
  knowledge — loaded on demand at ~30–50 tokens of overhead each, vs. an MCP
  tool schema that costs context on every session.
- **Plugins** bundle skills, hooks, slash commands, and MCP servers into one
  installable, versionable unit with marketplace distribution.
- **Code review is built in** (`/code-review`, `/security-review`), so
  "fetch the standards so the agent can review against them" is no longer a
  workflow gobuddy needs to power.

The division of labor in 2026 is: **MCP for computation and external state;
skills for knowledge; hooks for guaranteed enforcement**. gobuddy currently
uses MCP for all three, which is why half of it went stale.

### 1.2 Concrete defects found in review

1. **SDK five minor versions behind** — `go-sdk v1.1.0`; current stable is
   `v1.6.1`. Notable changes since: input-validation errors returned as
   `CallToolResult` instead of protocol errors (v1.5), origin-verification
   default flip (v1.6), `ToolAnnotations` best practices.
2. **Error-handling anti-pattern** — both tools return a non-nil Go `error`
   *and* a populated `CallToolResult` (`tools/godoc.go:38-42`,
   `tools/standards.go:234-241`). Modern SDK convention: expected tool
   failures ("package not found") are results with `IsError: true`;
   Go errors are reserved for protocol-level failures.
3. **`godoc` is not module-aware** — it runs `go doc` in the *server's*
   working directory, so documentation for the user's project dependencies
   resolves against the wrong module (or fails). There is also no timeout.
4. **`standards` is a security smell** — it will `git clone` any URL passed
   in tool input (SSH included) onto the host, silently ignores `git pull`
   failures (`_ = cmd.Run()`), and its default sources include a placeholder
   `git@github.com:your-org/go-standards.git`.
5. **No CI** — nothing verifies `gofmt`, `go vet`, or tests on push.
6. **No MCP-surface test** — nothing asserts which tools the server exposes;
   unit tests call the Go functions directly.
7. **Hardcoded version** — `v1.0.0` in `main.go`, unrelated to git tags.
8. **Non-hermetic tests** — the `godoc` "nonexistent package" case shells out
   to the network-touching `go doc` and the standards tests depend on repo
   files at fixed relative paths.

---

## 2. Target architecture

```
gobuddy/  (installable Claude Code plugin + standalone MCP server)
├── .claude-plugin/
│   ├── plugin.json            # plugin manifest (name, version, mcp, hooks)
│   └── marketplace.json       # single-entry marketplace for `/plugin` install
├── skills/
│   └── go-standards/SKILL.md  # the *knowledge* that `standards` used to fetch
├── hooks/
│   └── gofmt-check.sh         # PostToolUse: gofmt+vet on edited *.go files
├── commands/
│   └── check.md               # /gobuddy:check — one-shot quality gate
├── cmd/gobuddy/main.go        # slim MCP server (stdio)
├── tools/                     # MCP tools: computation only
│   ├── godoc.go               # module-aware doc lookup (kept, deepened)
│   └── gocheck.go             # NEW: fmt+vet+test in one structured call
├── internal/run/              # shared safe exec helper (timeouts, dir)
└── doc/design.md              # this file
```

**What each layer owns:**

- **MCP tools** (need runtime computation, benefit from structured output):
  - `godoc` — module-aware documentation, symbol listing, doc for the
    *project's* dependency versions.
  - `gocheck` — run `gofmt -l`, `go vet`, `go test -json` in a target module
    and return a compact structured summary (pass/fail, failing tests,
    unformatted files). One call instead of three shell round-trips; output
    is filtered so it doesn't flood context.
- **Skill** (`go-standards`) — the content of `CODING_STANDARDS.md`/CLAUDE.md
  as an on-demand skill: Go idioms, error-handling, table-driven tests.
  Replaces the `standards` MCP tool.
- **Hook** (`gofmt-check.sh`) — deterministic enforcement: after any edit to
  a `.go` file, run `gofmt -l` and `go vet` on the changed package and
  surface violations. Hooks are the only layer with *guaranteed* execution.
- **Command** (`/gobuddy:check`) — thin prompt that invokes the `gocheck`
  tool and fixes what it reports.

The MCP server remains fully usable standalone (any MCP client), so nothing
is lost for non-Claude-Code users; the plugin is packaging on top.

---

## 3. Implementation plan

Milestones are ordered so every step leaves `main`-quality state: each has a
**Verify** block of deterministic commands. A step is done only when **every
command exits 0**. This is the loop contract (see §4).

> Run all verification commands from the repo root.

### M0 — Baseline gate (CI + hygiene) `[x]`

Create the gate we'll hold every later step to.

1. Add `.github/workflows/ci.yml` running on push/PR: `gofmt` check,
   `go vet ./...`, `go build ./...`, `go test ./...` on the pinned Go version.
2. Add a `make check` target that runs the same four things locally.
3. Fix `.gitignore` for `bin/` if not present.

**Verify:**
```bash
test -f .github/workflows/ci.yml
make check                                  # runs fmt-check, vet, build, test
test -z "$(gofmt -l .)"
go vet ./...
```

### M1 — Dependency & SDK modernization `[x]`

1. `go get github.com/modelcontextprotocol/go-sdk@v1.6.1 && go mod tidy`.
2. Fix any breakage from the v1.1→v1.6 jump.
3. Adopt current conventions:
   - Expected tool failures return `CallToolResult{IsError: true}` with a
     nil Go error; reserve Go errors for protocol failures.
   - Add `ToolAnnotations` to every tool (`ReadOnlyHint: true` for `godoc`;
     `Title` for display names).
   - Derive server version from `debug.ReadBuildInfo()` instead of the
     hardcoded `v1.0.0`.

**Verify:**
```bash
go list -m github.com/modelcontextprotocol/go-sdk | grep -q 'v1\.6\.1'
! grep -rn 'Version: *"v1\.0\.0"' cmd/
grep -rn 'ReadOnlyHint' cmd/ tools/ | grep -q .
make check
```

### M2 — MCP-surface integration test `[x]`

Add `cmd/gobuddy` (or `tools`) integration test using the SDK's in-memory
transport: connect a client, `ListTools`, assert the exact expected tool set,
and call `godoc` end-to-end. This pins the public surface so later removals
and additions are verified, not assumed.

**Verify:**
```bash
go test ./... -run 'TestServer' -v | grep -q 'PASS'
make check
```

### M3 — Retire `standards`, land the `go-standards` skill `[x]`

1. Delete `tools/standards.go` + `tools/standards_test.go`; unregister from
   `main.go`; update the M2 surface test's expected tool list.
2. Create `skills/go-standards/SKILL.md` carrying the standards content
   (merge `CODING_STANDARDS.md` + the guidelines half of `CLAUDE.md`);
   frontmatter `name` + `description` describing when to load it.
3. `CODING_STANDARDS.md` becomes a pointer to the skill (or is folded in);
   `CLAUDE.md` keeps only repo-agent instructions, not duplicated standards.
4. Note the removal in README under a "Migrating from v1" heading.

**Verify:**
```bash
test ! -f tools/standards.go
test -f skills/go-standards/SKILL.md
head -1 skills/go-standards/SKILL.md | grep -q '^---$'   # has frontmatter
! grep -rn 'standards' cmd/gobuddy/main.go
grep -q 'Migrating from v1' README.md
make check                                  # includes updated surface test
```

### M4 — `godoc` v2: module-aware and structured `[x]`

1. Add `working_dir` input (optional; default: server cwd) — validated to be
   an existing directory containing a resolvable module — and run
   `go doc` there so the *project's* dependency versions answer.
2. Add `mode` input: `"doc"` (default), `"all"` (`go doc -all`), `"src"`
   (`go doc -src`).
3. Enforce a context timeout (30 s) on the subprocess via shared
   `internal/run` helper.
4. Package-not-found and symbol-not-found become `IsError` results carrying
   the `go doc` stderr — never a protocol error.
5. Table-driven tests for all modes plus the not-found path asserting
   `IsError` (not Go error), using a temp module fixture so tests are
   hermetic (no network).

**Verify:**
```bash
grep -q 'working_dir' tools/godoc.go
grep -q 'CommandContext' internal/run/*.go
go test ./tools/ -run 'TestGodoc' -v | grep -q 'PASS'
GOFLAGS=-mod=mod go test ./... -count=1
make check
```

### M5 — New tool: `gocheck` (the loop-friendly quality gate) `[x]`

One structured call replacing three shell round-trips:

- Input: `working_dir` (required), `packages` (optional, default `./...`),
  `include_tests` (default true).
- Runs `gofmt -l`, `go vet`, and (optionally) `go test -json`; parses test
  JSON to a compact summary.
- Output struct: `{ formatted_ok, vet_ok, tests_ok, unformatted_files[],
  vet_issues[], failing_tests[{name, package, output_tail}] }` — output
  tails truncated so a huge failure can't flood the context window.
- Annotations: `ReadOnlyHint: false` is wrong — it only reads; mark
  `ReadOnlyHint: true`, `OpenWorldHint: false`.
- Hermetic tests against two temp-module fixtures (one clean, one with a
  formatting error + failing test).

**Verify:**
```bash
test -f tools/gocheck.go
go test ./tools/ -run 'TestGocheck' -v | grep -q 'PASS'
go test ./... -run 'TestServer' -v | grep -q 'PASS'   # surface test updated
make check
```

### M6 — Plugin packaging (the flip) `[x]`

1. `.claude-plugin/plugin.json`: name `gobuddy`, description, version,
   `mcpServers` entry pointing at the built binary (`${CLAUDE_PLUGIN_ROOT}`),
   hooks config.
2. `hooks/gofmt-check.sh`: PostToolUse hook (matcher: Edit|Write) — if the
   edited file is `*.go`, run `gofmt -l` on it and `go vet` on its package;
   non-zero + stderr feedback on violation. POSIX sh, `bash -n`-clean.
3. `commands/check.md`: `/gobuddy:check` slash command prompting a
   run-`gocheck`-then-fix loop.
4. `.claude-plugin/marketplace.json` so the repo itself is installable via
   `/plugin marketplace add jKiler/gobuddy`.
5. README: plugin install path (primary) and standalone MCP path (secondary).

**Verify:**
```bash
jq empty .claude-plugin/plugin.json
jq empty .claude-plugin/marketplace.json
jq -e '.name == "gobuddy"' .claude-plugin/plugin.json
bash -n hooks/gofmt-check.sh
test -x hooks/gofmt-check.sh
test -f commands/check.md
test -f skills/go-standards/SKILL.md
grep -q 'marketplace add' README.md
make check
```

### M7 — Release polish `[ ]`

1. README rewrite around the new shape (plugin first; tools reference;
   architecture sketch).
2. `CHANGELOG.md` (Keep-a-Changelog format) documenting v2: standards→skill
   migration, godoc v2, gocheck, plugin packaging.
3. `make release-check` target = `make check` + plugin JSON validation +
   hook syntax check (the union of all Verify blocks above).
4. Tag readiness: version derived from build info (done in M1); confirm
   `git tag v2.0.0` would describe reality.

**Verify:**
```bash
test -f CHANGELOG.md
grep -q 'gocheck' README.md && grep -q 'godoc' README.md
make release-check
```

---

## 4. Loop protocol

This plan is written to be driven by an agent loop (e.g. `/loop`):

1. Find the first milestone whose heading checkbox is `[ ]`.
2. Implement it on branch `claude/codebase-modernization-plan-uhfx78`. If
   the previous milestone's PR has merged, restart the branch from
   `origin/main` first (`git checkout -B <branch> origin/main`); if it
   hasn't merged yet, wait — each milestone builds on the previous one.
3. Run that milestone's **Verify** block; every command must exit 0. Also
   re-run `make check` (the M0 gate) regardless of milestone.
4. On success: flip the milestone's `[ ]` to `[x]` in this file, commit all
   changes with message `feat(mN): <milestone title>`, push, and open a
   **new PR for that milestone** (one PR per iteration).
5. On failure: fix and re-verify within the same iteration; do not advance
   with a red Verify block. If genuinely blocked, record the blocker under a
   `## Blockers` section appended to this file and stop.
6. When every milestone is `[x]` and `make release-check` passes, the loop's
   goal is met.

Rules: one milestone per iteration; never edit a Verify block to make it
pass; if a Verify command proves wrong (typo, impossible assertion), fix it
in the *same* commit with a note here explaining why.

## 5. Explicitly out of scope (considered, rejected)

- **gopls-backed language intelligence** (references, implementations,
  diagnostics-as-you-type): highest ceiling, but a large dependency and
  lifecycle-management burden; revisit as v3 once v2 ships.
- **Keeping `standards` behind a flag**: dead weight; the skill supersedes
  it and the git-clone-from-tool-input surface is a liability.
- **HTTP transport**: stdio covers the plugin and local-client use cases;
  add only with a concrete remote deployment need.
- **staticcheck/golangci-lint integration in `gocheck`**: needs a
  third-party toolchain on the host; keep v2 to stdlib toolchain only,
  reconsider as an optional input later.
