# ImagePort

A cross-platform desktop application for managing container images, written in Go.

ImagePort manages local images, registry images and image archive files in a
Commander-style workspace. Choose a location in each pane, select an image, and
copy it to the other side. Registry addresses and access accounts are configured
separately. Registry/archive workflows do not require Docker; local image access
uses an existing Docker-compatible engine. ImagePort does not run containers.
End users do not need Go or elevated privileges.

**Development status:** initial implementation, not a production release. Native
Windows, macOS and Linux acceptance is tracked separately from automated tests.
See the [acceptance matrix](docs/acceptance.md) before making compatibility claims.

## Scope

- Two independent target selectors, tree/list browsing, paths, filtering and
  shared Inspect / Copy / Delete commands.
- A target manager for registries, existing Docker contexts and filesystem folders,
  with multiple independent access accounts per registry.
- Harbor project → image → version browsing with server pagination and totals;
  OCI catalogs and direct known-reference access.
- Registry ↔ registry, registry ↔ local engine, local engine ↔ local engine,
  image → archive and archive → registry/engine transfers.
- Explicit project/name/tag, image-root and platform review; immutable source
  identities, replacement consent and result verification.
- Precise local/Harbor tag deletion and selected archive-file deletion. Generic
  registries without safe single-tag deletion report unsupported.
- Persistent task attempts, cancellation, manual retry and interrupted-result checks.
- OS keyring or explicitly selected session credentials; no plaintext fallback.
- English/Simplified Chinese, remembered independent pane choices and light/dark themes.

Image builds, container execution, repository/project deletion and bulk pruning are
outside this implementation. See the [workspace guide](docs/workspace.md) for
supported routes, workflow details and limits.

## Technology

| Area | Choice |
| --- | --- |
| Language and toolchain | Go 1.26.10 |
| Desktop interface | [Fyne](https://fyne.io/), entirely in Go |
| Registry operations | [regclient](https://regclient.org/) Go SDK |
| Optional local images | Official Moby Go SDK and Docker CLI context store |
| Metadata | SQLite with the pure-Go modernc driver |
| Credentials | macOS Keychain, Windows Credential Manager, Linux Secret Service |
| License | Apache License 2.0 |

The compiler is pinned in [.go-version](.go-version). We use the patched previous
supported Go release series rather than chasing the newest series. Review the
[Go support policy](https://go.dev/doc/devel/release#policy) before each release.
Dependencies are pinned in `go.mod` and verified by `go.sum`.

## Product workflow proposal

The [project model](docs/project-model.md) defines projects, images and image
archive files as the primary objects, with separate registry endpoints, credentials
and project access contexts. The [user journeys](docs/user-journeys.md) describe
project setup, contextual sign-in, copying, and explicit deletion/cleanup. They
include proposed behavior beyond the current implementation. A
[Chinese workflow discussion](docs/user-journeys.zh-CN.md) and
[project-model discussion](docs/project-model.zh-CN.md) explain the proposal
for users without technical training.
The [transfer workspace proposal](docs/transfer-workspace.md), with a
[Chinese discussion](docs/transfer-workspace.zh-CN.md), applies the user's Commander
analogy: each pane has one dropdown for concrete registry, local image engine and
filesystem targets. A shared target manager, tree/list browsing and transfer queue
support copying between those locations.

## Development

The Go/Fyne application implements the approved Commander layout. The
[interactive HTML prototype](docs/prototype/commander/index.html) remains a
simulated design reference.
Open it directly in a browser; its sample data is independent of the Go application's database. See the
[prototype notes](docs/prototype/README.md) for the design artifact, and the
[workspace guide](docs/workspace.md) for real application behavior.

Install Go 1.26.10 and the native development prerequisites listed in
[Fyne's setup guide](https://docs.fyne.io/started/). Linux development needs the
OpenGL/X11 development libraries; Windows needs a compatible C compiler; macOS
needs Xcode command-line tools. These are developer requirements, not tools users
must install to manage images.

```sh
go mod download
go run ./cmd/client
```

Useful checks:

```sh
go test -tags ci ./...
go test -race -tags ci ./...
go vet -tags ci ./...
go run ./tools/check
go run -tags ci ./cmd/client --smoke --data-dir ./tmp/smoke --screenshot ./tmp/desktop.png
```

The `ci` tag selects Fyne's official headless test driver; packaged desktop
applications use the native driver. Integration tests start a real in-process OCI
registry and generate small archives; they do not require Docker or external
registry credentials. Use `go run` for local quick checks.

The manually triggered **Development packages** GitHub Actions workflow prepares
unsigned packages on Windows, macOS and Linux runners. The macOS application is
archived before upload to preserve executable permissions. This workflow has not
yet been executed for this initial implementation; signing, notarization and
native installation acceptance remain release work.

## Repository layout

```text
cmd/client/                  Application entry point and dependency assembly
internal/domain/             Image, registry profile, archive and task models
internal/registry/           Application-owned registry port and safe errors
internal/adapter/regclient/  Pinned SDK integration
internal/local/              Application-owned optional Docker image port
internal/adapter/docker/     Moby SDK and existing Docker context integration
internal/archive/            Streaming inspection and safe file publication
internal/transfer/           Preflight, transfer and result verification
internal/task/               Durable queue, attempts, cancellation and recovery
internal/store/              SQLite metadata and schema migration
internal/secret/             OS keyring and explicit session credentials
internal/presenter/          Validated UI commands and background read work
internal/ui/                 Fyne desktop screens
internal/localization/       English message keys and Simplified Chinese catalog
internal/testutil/           Generated deterministic test fixtures
docs/                       Architecture, development plan and acceptance
.github/                    CI and contribution templates
```

## Security and operational notes

HTTPS and certificate verification are enabled by default. Import a CA for a
specific connection when needed. Plain HTTP is restricted to loopback development
registries. Environment proxies use `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY`;
do not put credentials in proxy URLs.

Passwords and tokens are never intentionally persisted in SQLite, preferences,
task snapshots or application logs. If the OS keyring is unavailable, explicitly
choose session-only storage; the application does not fall back to plaintext.

Imports use a private copy of the approved archive. Docker archives additionally
need temporary OCI normalization space. Downloads need space for a same-directory
partial file. Do not assume a successful archive inspection proves compatibility
with every Docker or other downstream loader.

Local pushes capture the daemon endpoint and immutable image ID, export that ID
to a private temporary archive, and use the same conflict checks, normalization,
task history and remote digest verification as archive uploads. This can require
substantial temporary disk space. Adding a tag retains the image's existing tags;
replacing a local alias pointing to a different image requires confirmation.

Cancellation does not roll back remote writes. Interrupted outcomes must be
verified before retry. Target checks serialize this application's writes but do
not provide a registry-wide compare-and-swap guarantee.

## Contributing

English is the primary language for source code, comments, logs, default UI and
project documentation. Select **Settings → Language → English / 简体中文** to change
the application interface. Selection is remembered across restarts; entered text,
selected images and active tasks are retained. Application screens, dialogs and
task states use the selected language. Fyne/platform-owned file-browser and
context-menu internals follow the framework/system locale; unrecognized low-level
diagnostics retain their original text.

Read [CONTRIBUTING.md](CONTRIBUTING.md), [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md),
and [SECURITY.md](SECURITY.md). For architecture decisions, see
[docs/architecture.md](docs/architecture.md).

## License

Licensed under the [Apache License, Version 2.0](LICENSE). Third-party components
retain their respective licenses; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
