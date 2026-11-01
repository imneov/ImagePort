# Acceptance matrix

This matrix separates automated evidence from real-device release acceptance.
The project has not passed a production release gate. Update evidence only after
the corresponding check actually runs; never infer native acceptance from CI.

| # | Scenario | Automated coverage / planned check | Release acceptance |
| --- | --- | --- | --- |
| 01 | Download on Windows x64 without Docker/Go | In-process registry workflow; Windows CI configured | Pending real device |
| 02 | Incorrect credentials vs insufficient permissions | HTTP 401/403 classification tests | Pending real registry |
| 03 | Anonymous `/v2/`, protected image reads | Connection wording separates reachability and authorization | Pending auth integration |
| 04 | Disabled catalog, direct reference access | Disabled catalog and successful direct resolve integration | macOS arm64: catalog 401, known-path tags/resolve/download passed |
| 05 | Paged repositories/tags and current-page filtering | Server limit/cursor and second-page preservation tests | Pending native browsing |
| 06 | amd64/arm64 and variants | Multi-platform OCI resolve integration | macOS arm64: six-platform chooser, explicit arm64/v8 download passed |
| 07 | Single-platform OCI tar export and consumer load | Export/inspect digest roundtrip | Pending agreed consumer |
| 08 | Docker-save and OCI uploads | Both formats exercised against in-process OCI registry | Pending real registry |
| 09 | Multiple entries, ordinary tar and rootfs tar | Explicit entry UI; malformed-format rejection | Pending multi-entry compatibility |
| 10 | Copy tag and consent before overwrite | Conflict checks and source retention integration | Pending native confirmation flow |
| 11 | Truncation, corruption, traversal and links | Generated adversarial archive tests | Pending resource stress testing |
| 12 | Disk full, lost permissions, publication races | Atomic no-replace publication test | Pending disk fault injection |
| 13 | Cancellation near remote commit | Task cancellation tests; result readback path | Pending commit-window injection |
| 14 | Process killed and restarted | Durable restart state/queue tests | Pending forced native process exit |
| 15 | Lost successful response and state-write failure | Lost remote response integration | Pending post-publication DB failure |
| 16 | Same target and external concurrent modification | Scheduler serialization; fresh digest conflict check | Pending external-writer race |
| 17 | Changed source tag / changed archive on retry | Digest pinning; changed-archive rejection | Pending combined retry workflow |
| 18 | Custom CA and unavailable keyring | Per-profile CA; explicit session policy | Pending OS keyring/TLS acceptance |
| 19 | Secret-free metadata/logs and session cleanup | Metadata model has no password field | Pending release artifact audit |
| 20 | Unknown progress, large archives and UI switching | Headless render/race checks; bounded scanner | Pending native performance baseline |

## Platform sign-off

| Platform | Target architecture | Native launch | Keyring | Packaging/signing | No-Docker workflow |
| --- | --- | --- | --- | --- | --- |
| Windows | amd64 | Pending | Pending | Pending | Pending |
| macOS | arm64; amd64 pending | arm64 passed | Session-only tested; keyring pending | Local unsigned .app launch passed; distribution/signing pending | arm64 download/inspect passed; consumer load pending |
| Linux | amd64, arm64 | Pending | Pending | Pending | Pending |

For each signed-off row record OS version, CPU, display scale, registry product,
authentication mode, proxy/CA setup, archive consumer/version, digest evidence and
test date. Test paths containing spaces and non-ASCII characters, close/restart,
high DPI, readonly targets and keyring access prompts.

## Native interaction evidence — 2026-10-26

The macOS 14.8.4 arm64 native Fyne application was exercised through its desktop controls
against an HTTPS Harbor registry with account authentication and explicitly selected
session-only credentials. No credentials or private registry addresses are fixtures.

- Added a connection and checked HTTPS reachability.
- Confirmed that an authenticated account could read images while global catalog
  listing returned HTTP 401. Entering a known repository path successfully listed
  tags and resolved a six-platform image.
- Explicitly selected `linux/arm64/v8`, downloaded an OCI tar, and observed verified
  success in task history. Reopening the archive in Import found the same manifest
  digest and platform. No Docker/runtime was used.
- Switched English → Simplified Chinese → English through Settings. Navigation,
  task status and archive details changed language; path, selection and completed
  task remained intact. Confirmed the normal exit dialog.

The unnamed single OCI root exposed a blank selection label; the UI now uses its
digest when no reference name exists. Catalog failure guidance now explains the
known-path alternative. Upload/copy writes remain covered by the local test registry,
not this remote native session. Windows/Linux native operation, keyring prompts,
downstream archive consumers and release distribution still require acceptance.

## Redesigned workspace evidence — 2026-10-27

The approved HTML layout is now implemented in the native Go/Fyne application:
top toolbar, collapsible sidebar, Images, Registries, Local archives, Transfers and
Settings. The macOS arm64 application was packaged locally using the executable
produced by `go run`; the unsigned application opened successfully.

- Checked native startup rendering, toolbar language switching and light/dark
  appearance. The initial palette now follows the saved preference before widgets
  are constructed.
- Added an authenticated HTTPS connection using session-only credentials in the
  default application data directory. Known-path browsing returned a real tag;
  image inspection returned six platform choices.
- Explicitly chose `linux/arm64/v8`, queued an OCI download, and observed verified
  success in the redesigned Transfers drawer. The downloaded file's SHA-256 is
  `b28fba249f9beecc59874d3de85cef60783c81ed14ab35571c4f8be714d0c00b`;
  the selected manifest digest is
  `sha256:50084cfd55a79d93bcf91767a02f5b8a6cb1af1bea1595eac687101d7728f72c`.
- Confirmed that Local archives displayed this actual verified output. Reopening
  it through the redesigned import dialog remains pending native re-verification:
  the desktop automation service reported `noWindowsAvailable` and then timed
  out, while the application process remained running. No remote upload or copy
  was attempted during this session.
- Passed `go test -race -tags ci ./...`, `go vet -tags ci ./...`,
  `go run ./tools/check`, and `go mod verify`. Regression coverage includes
  current-page cursor preservation, connection reset, explicit platform consent,
  task filters, verified archive history, navigation state and saved appearance.

These checks do not sign off Windows/Linux native operation, OS keyring prompts,
installer signing, or remote write behavior. The UI shows registry data actually
available from the API; richer prototype metadata remained outside that version. Cross-registry copy
was subsequently implemented in the copy-workspace revision below. See [workspace.md](workspace.md).

## Known release gaps

Disk-space preflight warnings, measured byte/object progress, diagnostic export,
signed/notarized installers and full enterprise-proxy validation are not complete.
The current implementation uses stage/activity feedback. Temporary import copies
and Docker normalization can require substantially more disk space than the source
archive. No byte-level resume or rollback guarantee is offered.

## Copy workspace and optional Docker evidence — 2026-10-27

Images now uses side-by-side From and To locations. Copying a local image names
its destination directly; adding a local tag is optional. Remote selections require
explicit platform consent and can target another saved registry or an OCI tar.
The primary copy button remains fixed below the destination panel's scroll area.

- The production Docker adapter read the active OrbStack context and the native
  macOS window displayed 45 actual local images. Keyboard selection of a real
  `postgres:16-alpine` image populated its immutable image ID, `linux/arm64/v8`
  platform, destination repository/tag and complete target preview. No local tags
  were changed during this check.
- An opt-in test exported the existing local `registry.k8s.io/pause:3.9` image
  through the actual Moby SDK, uploaded to an ephemeral loopback registry, and
  independently verified its resulting manifest digest. No cloud tag was created.
- An actual two-registry test copied one explicitly selected platform while
  retaining the original multi-platform index. A changed saved source hostname
  was rejected without creating the target tag.
- UI regressions cover independent source/destination selectors, duplicate saved
  connection names, archive import destinations, source-change invalidation,
  editable destination naming, language switching, Harbor product API routing,
  server totals and current-page filtering. Docker transport tests reject
  unverified non-loopback TCP connections and insecure TLS environment settings.
- Real Harbor read-only diagnostics returned 17 accessible projects and an
  83-repository project; the supplied account had project administrator access.
  The adapter now uses authorized project APIs instead of requiring global catalog
  access. Native Harbor browsing after reentering session credentials remains to
  be rechecked in this latest copy workspace.
- Passed `go test -race -tags ci ./...`, `go vet -tags ci ./...`,
  `go run ./tools/check`, and `go mod verify`. The final button placement adjustment
  also passed the UI race suite and source checks. Focused Go review found no
  outstanding correctness/security blocker. Installed staticcheck/golangci binaries
  were built with an older Go version and could not validate this toolchain.

The unsigned macOS application was refreshed from the executable produced by
`go run`. Desktop automation's coordinate actions continued to fail with
`noWindowsAvailable`; keyboard input was used to verify real image selection.
This is not acceptance of cloud writes, Windows/Linux native behavior, keyring
prompts or distribution signing. Fyne also reports a missing framework-owned
`(Select one)` translation at widget construction; application selectors use
explicit translated captions.

## Commander implementation evidence — 2026-10-27

The approved Commander baseline replaces the previous five-page entry screen with
two independent tree/list panes, one location selector per pane, a shared operation
bar, target/account management dialogs and a durable transfer queue. Earlier
sections describe earlier versions and do not constitute acceptance of this layout.

- Fyne software-rendered interaction tests check visible received rows (including
  a regression for a hidden tree root), retained server cursors/totals, independent
  pane identity and command direction, source naming, replacement-consent
  revocation, explicit multi-image/platform selection and bilingual state retention.
- Metadata migration tests group legacy connections by endpoint while preserving
  account IDs, secret references and existing task inputs. Initial local locations
  are scoped to the database and are not recreated after removal.
- Same-host, separate-account integration uses a real in-process OCI registry with
  distinct read-source/write-target permissions. Copy preserves the complete index
  while isolating the two credentials. Harbor deletion tests assert the precise tag
  endpoint and verify that another alias sharing the digest remains readable.
- On this macOS arm64 machine, the existing `registry.k8s.io/pause:3.9` image was
  exported through the Moby SDK, uploaded to an ephemeral loopback registry, pulled
  back under a unique test tag, copied under another tag, exported to an inspected
  archive and loaded under a third tag. Each task reached verified success. The
  test removed only its own unique aliases and retained the original image.
- This real workflow exposed SDK request-body ownership closing the caller's file.
  The adapter now retains caller ownership; an SDK transport regression covers it.
- Bundled unmodified OFL-licensed regular/bold CJK fonts remove reliance on
  system font availability. A catalog-wide glyph check and Chinese rendering
  preview cover the missing-glyph regression found during visual review.
- Native-widget preview images use deterministic sample data and Fyne's official
  software renderer. They do not prove native OS input behavior or cloud writes.

The native entry point was also launched with `go run ./cmd/client` using the
default metadata directory. Desktop automation rejected the unbundled executable
as an application target, so this launch does not certify native input interactions.

Windows/Linux native acceptance, current Commander interaction through a real
macOS window, cloud writes, OS keyring prompts and distribution signing remain
pending. Generic OCI single-tag deletion is deliberately unsupported without an
appropriate product API. No container was started in these checks.
