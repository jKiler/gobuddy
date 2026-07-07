package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jKiler/gobuddy/internal/run"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// outputTailLimit bounds per-test output in the gocheck summary so a huge
// failure cannot flood the caller's context window.
const outputTailLimit = 2000

// GocheckInput is the input for the gocheck tool.
type GocheckInput struct {
	WorkingDir   string   `json:"working_dir" jsonschema:"directory of the Go module to check"`
	Packages     []string `json:"packages,omitempty" jsonschema:"package patterns to vet and test (defaults to ./...)"`
	IncludeTests *bool    `json:"include_tests,omitempty" jsonschema:"run the test suite in addition to gofmt and go vet (defaults to true)"`
}

// FailingTest describes one failing test with the tail of its output.
type FailingTest struct {
	Name       string `json:"name" jsonschema:"the failing test name (empty for a package-level failure such as a build error)"`
	Package    string `json:"package" jsonschema:"the package the failure occurred in"`
	OutputTail string `json:"output_tail" jsonschema:"the last part of the test output, truncated"`
}

// GocheckOutput is the structured quality-gate report for gocheck.
type GocheckOutput struct {
	FormattedOK      bool          `json:"formatted_ok" jsonschema:"true when gofmt reports no unformatted files"`
	VetOK            bool          `json:"vet_ok" jsonschema:"true when go vet reports no issues"`
	TestsOK          bool          `json:"tests_ok" jsonschema:"true when all tests pass (or tests were not run)"`
	UnformattedFiles []string      `json:"unformatted_files,omitempty" jsonschema:"files that need gofmt"`
	VetIssues        []string      `json:"vet_issues,omitempty" jsonschema:"issues reported by go vet"`
	FailingTests     []FailingTest `json:"failing_tests,omitempty" jsonschema:"tests that failed, with truncated output"`
}

// Gocheck runs gofmt, go vet, and the test suite in one call and returns a
// compact structured report. Check failures are data in the report, not
// tool errors; IsError is reserved for not being able to run the checks.
func Gocheck(ctx context.Context, req *mcp.CallToolRequest, input GocheckInput) (
	*mcp.CallToolResult,
	GocheckOutput,
	error,
) {
	info, err := os.Stat(input.WorkingDir)
	if err != nil || !info.IsDir() {
		return gocheckError(fmt.Sprintf("working_dir %q is not an existing directory", input.WorkingDir))
	}

	packages := input.Packages
	if len(packages) == 0 {
		packages = []string{"./..."}
	}

	out := GocheckOutput{FormattedOK: true, VetOK: true, TestsOK: true}

	// gofmt -l lists unformatted files and exits 0 either way; a non-zero
	// exit means gofmt itself could not run.
	fmtOutput, err := run.Command(ctx, input.WorkingDir, "gofmt", "-l", ".")
	if err != nil {
		return gocheckError(fmt.Sprintf("gofmt failed: %v\nOutput: %s", err, fmtOutput))
	}
	out.UnformattedFiles = splitLines(string(fmtOutput))
	out.FormattedOK = len(out.UnformattedFiles) == 0

	vetOutput, err := run.Command(ctx, input.WorkingDir, "go", append([]string{"vet"}, packages...)...)
	if err != nil {
		out.VetOK = false
		out.VetIssues = splitLines(string(vetOutput))
	}

	if input.IncludeTests == nil || *input.IncludeTests {
		testOutput, err := run.Command(ctx, input.WorkingDir, "go", append([]string{"test", "-json"}, packages...)...)
		if err != nil {
			out.TestsOK = false
			out.FailingTests = parseFailingTests(string(testOutput))
		}
	}

	summary, err := json.Marshal(out)
	if err != nil {
		return gocheckError(fmt.Sprintf("marshaling report: %v", err))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(summary)},
		},
	}, out, nil
}

// testEvent is one entry in the go test -json stream.
type testEvent struct {
	Action  string `json:"Action"`
	Package string `json:"Package"`
	Test    string `json:"Test"`
	Output  string `json:"Output"`
}

// parseFailingTests reduces a go test -json stream to the failing tests
// with the tail of their output. Package-level failures (e.g. build
// errors) are reported only for packages with no individual test failure,
// so they don't duplicate their tests' entries.
func parseFailingTests(stream string) []FailingTest {
	type key struct{ pkg, test string }
	buffers := make(map[key]*strings.Builder)
	var failed []key
	failedTestsByPkg := make(map[string]bool)
	var raw strings.Builder

	for _, line := range strings.Split(stream, "\n") {
		if line == "" {
			continue
		}
		var ev testEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			// Non-JSON lines (e.g. toolchain errors) go to the raw tail.
			raw.WriteString(line)
			raw.WriteString("\n")
			continue
		}
		k := key{ev.Package, ev.Test}
		switch ev.Action {
		case "output":
			b, ok := buffers[k]
			if !ok {
				b = &strings.Builder{}
				buffers[k] = b
			}
			b.WriteString(ev.Output)
		case "fail":
			failed = append(failed, k)
			if ev.Test != "" {
				failedTestsByPkg[ev.Package] = true
			}
		}
	}

	var failing []FailingTest
	for _, k := range failed {
		if k.test == "" && failedTestsByPkg[k.pkg] {
			continue
		}
		var output string
		if b, ok := buffers[k]; ok {
			output = b.String()
		}
		failing = append(failing, FailingTest{
			Name:       k.test,
			Package:    k.pkg,
			OutputTail: tail(output, outputTailLimit),
		})
	}

	if len(failing) == 0 {
		// go test failed without reporting any package: surface the raw
		// output so the caller sees why.
		msg := raw.String()
		if msg == "" {
			msg = stream
		}
		failing = append(failing, FailingTest{OutputTail: tail(msg, outputTailLimit)})
	}
	return failing
}

// splitLines returns the non-empty lines of s.
func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// tail returns at most limit trailing bytes of s, cut on a rune boundary.
func tail(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	s = s[len(s)-limit:]
	for i := 0; i < len(s); i++ {
		if (s[i] & 0xC0) != 0x80 {
			return s[i:]
		}
	}
	return ""
}

// gocheckError creates an IsError tool result, keeping the Go error nil so
// expected failures are reported in-band rather than as protocol errors.
func gocheckError(msg string) (*mcp.CallToolResult, GocheckOutput, error) {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}, GocheckOutput{}, nil
}
