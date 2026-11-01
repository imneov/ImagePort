# Contributing to ImagePort

Thank you for contributing. ImagePort is a Go desktop application focused on
container image management. Please keep changes aligned with that scope.

## Before changing code

For a bug, describe observed behavior, reproduction steps, expected behavior and
relevant sanitized evidence. Trace data from the registry response through the
adapter, presenter and UI before choosing a fix. In pagination changes, identify
whether the server or client owns pagination; do not slice an already paged result.

For larger features, open a proposal describing the user workflow and failure
semantics. Avoid unrelated refactors. Never include real credentials or private
image layers in issues, fixtures, logs or screenshots.

## Local workflow

1. Install the version in `.go-version` and Fyne's native prerequisites.
2. Create a topic branch from `main`.
3. Run `go run ./cmd/client` for a quick local check.
4. Add meaningful behavioral tests for changed safety or transfer semantics.
5. Run `gofmt`, `go test -race -tags ci ./...` and `go vet -tags ci ./...`.
6. Check changed screens with the native application and include relevant evidence.
7. Open a pull request explaining the problem, resulting behavior and validation.

Headless smoke rendering is available with:

```sh
mkdir -p tmp
go run -tags ci ./cmd/client --smoke --data-dir ./tmp/smoke --screenshot ./tmp/desktop.png
```

## Coding standards

- Use English for identifiers, comments, errors, logs, default UI and documentation.
- Keep files below 1,000 lines and functions at or below 50 lines.
- Document functions, their meaningful inputs, return values and non-obvious logic.
- Use descriptive names. Keep SDK and OS capabilities in lower-level adapters;
  application services coordinate features; UI renders and submits intent.
- UI must not call the SDK or start transfer goroutines. Marshal background Fyne
  updates through `fyne.Do`.
- Handle errors explicitly. Do not swallow failures, expose raw auth responses,
  use empty handlers, or use temporary workarounds to conceal root causes.
- Log key task boundaries using opaque task IDs; never log credentials or tokens.
- Validate user input and archive contents before using them.
- Do not disable TLS verification, add credentials to metadata, invoke Docker, or
  mutate external resources as a diagnostic workaround.
- Change generators rather than hand-editing generated files.
- Pin dependency versions and update checksums with Go tooling.

## Tests and acceptance

Interface text uses `internal/localization` and go-i18n. Keep English message keys
and source comments; add Simplified Chinese text to `zh-Hans.json`. Preserve printf
arguments and use the UI's bound helpers for stable screen captions, or `tr` for
transient dialogs. Render task codes for display without translating persisted
state. Changing language must retain user inputs, selections and running tasks.
Run catalog format/coverage checks and UI language-switch tests with the suite.

Tests generate tiny OCI and Docker archives and use an in-process OCI registry.
The test suite must run without Docker or access to a private registry. Fault
tests should verify real outcomes: file contents, digests, durable state, target
conflicts, cancellation and interrupted recovery.

Update `docs/acceptance.md` when testing a real platform. State exactly which OS,
architecture, registry and archive consumer were tested. A green CI compilation
does not establish real desktop acceptance or signed distribution.

## Pull requests and licensing

Use descriptive commit subjects and focused pull requests. Follow the pull request
template. Contributions are provided under the project's Apache-2.0 license.
Do not introduce code or assets with incompatible licensing. Security issues
follow the private reporting process in `SECURITY.md`.
