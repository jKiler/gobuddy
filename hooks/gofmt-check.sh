#!/bin/sh
# PostToolUse hook: after an Edit/Write to a .go file, fail with feedback
# (exit 2) when the file is not gofmt-clean or its package does not vet,
# so violations are corrected the moment they are introduced.
set -u

input=$(cat)
file=$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty')

case "$file" in
*.go) ;;
*) exit 0 ;;
esac

[ -f "$file" ] || exit 0

if [ -n "$(gofmt -l "$file")" ]; then
	printf 'gofmt: %s is not gofmt-formatted. Run: gofmt -w %s\n' "$file" "$file" >&2
	exit 2
fi

dir=$(dirname "$file")

# Only vet inside a module; elsewhere go vet cannot resolve the package.
gomod=$(cd "$dir" && go env GOMOD 2>/dev/null)
if [ -z "$gomod" ] || [ "$gomod" = "/dev/null" ] || [ "$gomod" = "NUL" ]; then
	exit 0
fi

if ! vet_out=$(cd "$dir" && go vet . 2>&1); then
	printf 'go vet reported issues in %s:\n%s\n' "$dir" "$vet_out" >&2
	exit 2
fi

exit 0
