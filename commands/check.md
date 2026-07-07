---
description: Run the gobuddy quality gate (gofmt, vet, tests) and fix everything it reports
---

Run the `gocheck` tool from the gobuddy MCP server with `working_dir` set to
the repository root of the current project.

Then, until the report is fully clean (`formatted_ok`, `vet_ok`, and
`tests_ok` all true):

1. Fix every `unformatted_files` entry by running `gofmt -w` on it.
2. Fix each `vet_issues` finding in the source it points at.
3. Diagnose each `failing_tests` entry from its `output_tail` and fix the
   code (or the test, when the test itself is wrong — say so explicitly).
4. Re-run `gocheck` to confirm.

Finish with a short summary of what was fixed. If the report was already
clean, just say so.
