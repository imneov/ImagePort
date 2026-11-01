# Desktop workspace

The Go/Fyne application now implements the approved Commander workspace. Each
pane has one location dropdown, independent access selection, a path, a filter,
and tree/list views. The blue pane marker identifies the source of the shared
Inspect (F3), Copy (F5) and Delete (F8) actions. The opposite pane is the copy
destination. A modal blocks these workspace shortcuts while it is open.

Run `go run ./cmd/client` from the repository, or open a current packaged build.
Close and restart an older running process to load source changes. The default
database is in the OS configuration directory's `ImagePort` folder. The
`--data-dir /absolute/path` option opens independent metadata and task history.
Local locations are initialized once per database; deleting their configurations
does not silently recreate them.

## Configure locations and access

Choose **Manage targets**. Saved locations appear on the left; their settings
appear on the right. **New target** offers Image registry, Local images, or
Folder / disk. Local image locations use an existing Docker context; an empty
context uses Docker's current configuration. File locations use a selected folder
as their browsing boundary. Configuration does not start a daemon or create files.

A registry location holds its address. Add one or more access accounts separately.
Accounts hold a display name, username/token, optional CA and credential storage
policy. Passwords stay in the OS keyring or explicitly selected session memory.
Each pane can choose a different account for the same registry. Identical display
names are disambiguated. The two panes remember their own location/account IDs.
**Sign in** edits the selected account, allowing expired session credentials to
be replaced. Anonymous access remains a separate choice.

Removing a location or account removes configuration and its own credentials;
it does not delete images, archive files, or task history.

## Browse and copy

Harbor locations expose project → image → version. Other OCI registries use
repository → version. Enter a known repository path to list versions when global
listing is restricted, or enter a complete image reference for direct access.
Errors remain in the affected pane. **Load more** retains server cursors and totals;
the filter searches received rows and does not apply a second pagination step.
Local images are grouped by name with their actual tags and immutable daemon IDs.
Folders show files, with image archives eligible for import.

Select an image version or archive, then Copy. The review preserves the source
name, shows the exact destination reference/file, and includes only the inputs
for that route. A destination registry requires a project, image name, version
name and account. Known projects appear in an editable selector. An unrelated
selection in the destination pane never renames the source.

| Source | Destination | Result |
| --- | --- | --- |
| Registry | Registry | Copy selected manifest or complete index |
| Local images | Registry | Export pinned daemon image and upload |
| Registry | Local images | Pull selected platform into the selected engine |
| Local images | Local images | Add a copy under the reviewed tag/engine |
| Registry / Local images | Folder | Publish a verified `.tar` without overwriting |
| Image archive | Registry | Upload one explicitly selected image root, retaining its platforms |
| Image archive | Local images | Load one selected image and platform under the reviewed tag |

Multi-image archives require an explicit image choice. Engine destinations require
one platform; registry/file copies can retain the complete remote index. Existing
target versions require replacement consent. Changing the name, project, account,
platform or archive entry revokes that consent. Execution rechecks the observed
target before writing. Other clients can still race this check; it is not an
atomic registry-wide compare-and-swap.

**Swap panes** exchanges location, account, browsing context, selection and filter.
Language and appearance switches retain user inputs. The interface is English by
default; EN/中 or Settings changes language immediately.

## Delete and track work

Delete confirms one exact version name or archive file. Local tags are removed
without force or parent pruning. Harbor uses its precise tag endpoint. Generic
OCI registries without a precise tag API report unsupported; ImagePort never
substitutes manifest-wide deletion, which could remove other tags. Project,
repository and bulk pruning are outside this implementation.

Transfers appear below the panes with real state and stage. History exposes
captured inputs, attempts, cancellation, retry and interrupted-result verification.
Queued work captures content identity, endpoint, engine, accounts and destination;
navigating the panes cannot retarget it. Editing an account's username prevents a
new task from silently using another identity. Retry retains the original source
and rechecks destination replacement consent. Restored queued work requires an
explicit Resume. Cancellation does not roll back completed writes.

A completed task offers **Open destination**. Completion does not move a pane or
discard its selection automatically. Refresh the destination to see the result.

## Verification and limits

See [acceptance.md](acceptance.md) for evidence. Headless previews in
[the prototype directory](prototype/commander/README.md) render actual Fyne widgets
with deterministic sample content; they are not screenshots of a live registry.
The HTML prototype retains simulated data independent of the application database.

Native Windows/Linux interaction, OS keyring prompts, signing and installer
acceptance remain pending. Stage feedback does not imply measured byte progress.
Temporary export/normalization space is required; byte-level resume, rollback and
container execution are not provided.
