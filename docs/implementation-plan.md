# ImagePort implementation plan

Source: `方案.docx`, Image Management Client Development Plan, reviewed 2026-10-26.

## Product scope

The latest product proposal combines [project-model.md](project-model.md) with
[user-journeys.md](user-journeys.md). Projects, images and image archive files are
the main objects; registry endpoints and multiple account credentials support
project access. The proposal adds contextual sign-in, copy routes and cleanup.
The [transfer workspace](transfer-workspace.md) develops the user's Commander analogy
into two panes with one target dropdown each, a shared target manager, tree
browsing and a transfer queue. The 2026-10-28 workspace feedback removes the
alternative list/table mode in favor of a compact tree. These proposals take
precedence over earlier UI assumptions; implementation status remains separate.

ImagePort is a standalone desktop image management client written in Go. Fyne
provides the UI; regclient provides OCI registry and image transfer capabilities.
Registry/archive workflows do not need Docker or a daemon. The optional local
image workflow uses an existing Docker-compatible engine through the official
Moby Go SDK. End users do not need Go or administrator privileges.

The approved desktop layout uses a full-width toolbar and a left sidebar for
Images, Registries, Local archives, Transfers, and Settings. Archive import is a
focused inspection/destination dialog; image and task details use a right drawer. Supported operations are browsing repositories and tags, resolving
platforms, exporting one selected platform as OCI Layout tar, importing one
selected docker-save/OCI archive entry, and copying a pinned manifest to a tag.
Images uses a copy model: choose a source location and image, choose a destination
and name/tag, then copy. Sources include local Docker and saved registries. Targets
include independently authenticated registries and local OCI archives for remote
sources. Local tagging remains optional. Harbor browsing uses authorized projects
and their repositories/tags; other OCI registries retain catalog/direct access. The application does not run
containers, build images or manage the engine lifecycle. Image deletion is
proposed in the user journeys and is not currently implemented.

## Implementation sequence

Toolchain policy: use Go 1.26.10, the security-patched previous supported release
series as of 2026-10-26. Pin the same version in `.go-version`, `go.mod`, and CI.
Reassess support before each release; do not freeze an unsupported compiler in
the name of stability. End users install packaged binaries and do not need Go.

1. Pin dependencies; prove regclient import/export and digest readback against an
   in-process OCI registry without Docker.
2. Implement domain models, secure streaming archive inspection, SQLite metadata,
   system-keyring/session secrets, and registry adapter.
3. Implement persistent queue, attempts, cancellation, retry, target serialization,
   restart interruption, explicit queue resume, and result reconciliation.
4. Connect Fyne screens through a presenter. Require explicit platform selection,
   archive-entry selection, and overwrite confirmation. Keep network/file work off
   the UI goroutine and marshal updates through `fyne.Do`.
5. Add behavioral tests, platform packaging CI, contributor/security documentation,
   and an honest acceptance matrix.

## Invariants

- HTTPS and certificate validation by default; optional per-profile CA, never a
  skip-TLS toggle. Environment proxy support, no embedded proxy credentials.
- Credentials live in the OS keyring or explicitly selected session memory.
  Metadata, task records and logs contain no passwords or tokens.
- Source manifests are immutable after confirmation. Retries create attempts and
  do not follow changed tags. Import retries verify the original archive identity.
- Archive scanning is streaming and bounded; reject traversal, links, duplicates,
  corrupt blobs and incomplete graphs. Do not extract into user directories.
- Exports use a same-directory partial file, verify before publication, and never
  overwrite an existing final file.
- Remote overwrite confirmation records the previously observed digest and is
  checked again before execution. This is not a server-side compare-and-swap.
- Success requires digest/file verification. Ambiguous remote outcomes remain
  interrupted rather than being blindly retried.
- Persist active tasks as interrupted on restart; queue resumption is explicit.

## Release gate

Automated tests cannot establish real Windows/macOS/Linux user acceptance. Track
the document's 20 cases in `docs/acceptance.md`; keep unverified cases pending.
Unsigned CI artifacts are development packages, not signed production releases.
