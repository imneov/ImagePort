# Repository instructions

## Language and scope

- English is primary for source, comments, logs, error messages, default UI and docs.
- ImagePort manages images; it does not run containers. Registry/archive workflows do not require Docker;
  optional local image access uses an existing Docker-compatible engine.
- Every visible UI element must support the current user task. Keep design rationale
  and implementation notes in documentation, outside the application interface.
- Follow the source design in `docs/implementation-plan.md` and repository policies.

## Architecture and quality

- Keep files below 1,000 lines and functions at or below 50 lines.
- Document function purpose, meaningful parameters and returns; explain complex logic.
- Use descriptive variable names and split modules by responsibility.
- UI calls the presenter; SDK calls belong in the adapter. Transfer goroutines belong
  in the task scheduler. Background Fyne updates must use `fyne.Do`.
- Handle errors and log key task boundaries without logging secrets.
- Never hand-edit generated output; modify its generator.
- Do not apply workaround mutations to external resources to hide application bugs.

## Debugging and verification

- Understand the reported symptom and supplied evidence before reading implementation.
- Trace registry/API data through adapter, presenter and UI; validate each boundary.
- Preserve server pagination metadata/cursors; avoid double pagination.
- Prefer `go run` for quick local Go checks, not `go build`.
- Run meaningful tests and static checks appropriate to changes.
- For browser work, prefer the `agent-browser` skill. If network access fails, the
  user-provided development proxy is available; do not embed it in app defaults.
- Do not manually change permissions or Kubernetes custom resources as a workaround.
- Sandbox images, if introduced later, must be built with GitHub and pushed to the
  cloud registry; no local-only delivery substitutes.
