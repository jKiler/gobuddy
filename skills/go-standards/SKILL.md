---
name: go-standards
description: Go coding standards for writing and reviewing Go code — formatting, naming, error handling, simplicity, documentation, and testing conventions. Load when writing, refactoring, or reviewing Go code so the changes follow the project's idioms.
---

# Go Coding Standards

All Go code you write or review should follow these principles.

## Formatting

- All Go code **must** be formatted with `gofmt` before being submitted.

## Naming Conventions

- **Packages:** Use short, concise, all-lowercase names.
- **Variables, functions, and methods:** Use `camelCase` for unexported
  identifiers and `PascalCase` for exported identifiers.
- **Interfaces:** Name interfaces for what they do (e.g., `io.Reader`),
  not with a prefix like `I`.

## Error Handling

- Errors are values. Do not discard them.
- Handle errors explicitly using the `if err != nil` pattern.
- Provide context to errors using `fmt.Errorf("context: %w", err)`.

## Simplicity and Clarity

- "Clear is better than clever." Write code that is easy to understand.
- Avoid unnecessary complexity and abstractions.
- Prefer returning concrete types, not interfaces.

## Documentation

- All exported identifiers (`PascalCase`) **must** have a doc comment.
- Comments should explain the *why*, not the *what*.

## Testing

- Write table-driven tests.
- Aim for >80% code coverage.
