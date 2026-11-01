# Commander interaction prototype

Open [index.html](index.html) directly in a browser. All assets are local; no Go
service, package installation or registry credentials are needed. The default
language is English; the header switches immediately to Simplified Chinese.

This implements an interactive discussion artifact for
[transfer-workspace.md](../../transfer-workspace.md), not the native application.

## Try it

1. Start with the selected local `service-api:2.4.0`, with `edge` selected in the
   right-hand registry. Copy, review the target reference and platforms, and confirm.
   The queue simulates copying/verification; the destination tree gains the version.
2. Open either target dropdown. Pick an actual registry, local Docker image engine,
   Downloads or Image exports directly; there is no intermediate type selector.
3. Select Image exports on the other side to export a sample archive. Select that
   archive to import into local images or upload to a registry project.
4. In Downloads, expand Releases and select `release-bundle.tar`. Copy opens an
   entry picker for its two sample images. Ordinary files cannot be imported.
5. Manage targets lists saved locations on the left and configuration on the right.
   The add button first offers Registry, Local image engine and Folder / disk. It creates/edits/removes shared configuration. A new registry opens
   a sample sign-in; sign-in returns to the initiating copy without losing inputs.
   Its illustrative `library` project is fixture content, not a remote project creation.
6. Select a read-only identity to see upload denial, or a maintenance identity for
   scoped deletion. Accounts do not create duplicate registries or merge permissions.
7. Inspect a conflict, change the version, cancel/retry a task, or browse somewhere
   else while copying. Captured targets/identities do not follow later pane changes.

| Source -> Destination | Simulated outcome |
| --- | --- |
| Registry -> Registry | Image version appears in the selected project |
| Registry -> Local images | Image version appears in the engine tree (pull) |
| Local images -> Registry | Image version appears in the selected project (upload) |
| Registry/local images -> File system | An inspectable archive appears in the chosen folder |
| Filesystem archive -> Local images | Selected entry appears in the engine tree (load) |
| Filesystem archive -> Registry | Selected entry appears in the selected project |

## Interface reference

The user-provided Commander One screenshots (`1.png`, `2.png`, `3.png`) guide the
compact desktop layout: one toolbar, two contiguous panes, a location selector and
path per pane, dense content rows, per-pane status and a shared bottom command bar.
The active pane owns Inspect (F3), Copy (F5) and Delete (F8); the opposite pane is
the copy destination. Copy preserves the source image name unless the user changes
it explicitly in review; an unrelated selection in the other pane does not rename it. Only supported image tasks are shown. Transfer rows appear
when tasks exist; the toolbar opens history, including its empty state.

Every visible element must help browse, choose, copy or clean up content, or make
a necessary decision about access, conflicts or destructive scope. Design rationale
and implementation details belong in documentation.

## Boundaries

Names, addresses, platforms, sizes, identities, permissions, archive contents and
progress are explicit fixtures. No real registry connection, authentication, Docker
operation, archive inspection, file publication or image deletion occurs. Sign-in reconnects a saved sample identity without requesting a password; do not enter real credentials. New account forms accept
labels/identity names only, with explicitly simulated publisher access.

Target/content/task state exists only in memory and resets on reload. Only language
and appearance preferences persist in browser local storage, with a session fallback
if storage is unavailable. No external fonts, CDN, analytics or API requests are used.
Prototype boundaries are documented here; the workspace uses ordinary task labels.
It contains no design narrative, slogans or implementation notes.

Saved discussion screenshots: [workspace](preview.png) and
[target manager](targets-preview.png) and [new target](new-target-preview.png), shown in Chinese for review. The live page
defaults to English until the user switches language.

The browser's local filesystem target is an illustrative folder tree; it does not
grant the HTML page access to actual user files. Existing archive entries are read-only;
export creates a new sample file or explicitly replaces one. Changing an endpoint
clears its old sample access/content; it does not reuse another host's credentials.

## Verification

Verified in Chromium through `agent-browser` with direct `file://` access:
target selection, tree/list navigation, retained search focus, local upload, archive
export, archive-to-engine import, archive-to-registry entry selection, registry-to-engine
pull, and project-to-project copying with separate accounts on the same registry.
Verified sign-in with retained names, required project selection, overwrite consent
revocation after edits, upload/delete permission feedback, independent source retention,
cancel/retry, queue destination retention while navigating, and target creation.

The Commander One revision also verifies F3/F5/F8, active-pane command direction,
source naming with an unrelated destination selection, target-type selection and
the empty queue staying out of the workspace.

Desktop, Chinese/dark appearance and compact viewport checks are documented as UI
verification only. They are not evidence of real registry operations or native
Windows/macOS/Linux acceptance. JavaScript syntax is checked with `node --check`.

## Files

| File | Responsibility |
| --- | --- |
| `messages.js` | English source messages and Chinese translations |
| `model.js` | Explicit fixtures, domain lookup and per-pane access state |
| `operations.js` | Copy routes, input validation, conflicts and simulated tasks |
| `render.js` | Workspace, tree/list, target dropdown and queue rendering |
| `dialogs.js` | Copy review, inspection, cleanup and history dialogs |
| `targets.js` | Target manager and explicit account/sign-in flows |
| `app.js` | Event routing, keyboard navigation and browser lifecycle |
| `styles.css`, `dialogs.css` | Local styles and responsive/dark appearance |

## Go/Fyne implementation previews

The approved layout is now the native application's main workspace. Files
`native-workspace.png`, `native-targets.png`, `native-copy.png` and
`native-workspace-zh.png` render actual Fyne widgets with deterministic sample
content. They are produced by the UI preview test, not by the HTML simulation or
a private registry. Generate them from the repository root with:

```sh
IMAGEPORT_UI_SCREENSHOT_DIR="$PWD/docs/prototype/commander" go test -tags ci ./internal/ui -run TestCommanderPreview
```

See [the workspace guide](../../workspace.md) for implemented operations and
[acceptance](../../acceptance.md) for verification boundaries.
