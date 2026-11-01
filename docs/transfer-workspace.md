# Commander-style image transfer workspace

Status: interaction proposal, 2026-10-27; not implemented by this document.
This applies the [project model](project-model.md) and
[user journeys](user-journeys.md) to a browsable transfer workspace.
See the [Chinese discussion companion](transfer-workspace.zh-CN.md).

## Interaction principle

Let users open two places, inspect their contents, select an image and copy it
across. Keep both places visible and show background work in a transfer queue.
The user should be able to find the result at its destination without reading logs.
Each pane has one target dropdown at the top. It directly lists configured
locations of all supported kinds: a registry, a local image engine, or a filesystem
root/folder. Users do not first select a type and then choose another location.
Both dropdowns use the same managed target catalog, with independent selections.
Contents use one compact expandable tree per pane. Names use the available row width;
platform, digest and size belong in Inspect rather than mostly empty columns.
The table/list alternative was removed following workspace feedback on 2026-10-28.
The following is an interaction sketch, not a pixel-level screen specification.

```text
ImagePort       Manage targets       Transfers       Language   Appearance
-------------------------------------------------------------------------
[This computer / Docker v]             | [Company registry v]
Engine: current context                | Account: Release [change]
Search / Refresh                       | Search / Refresh
--------------------------------------|----------------------------------
v service-api                         | v Edge project
    1.4        arm64                   |   v service-api
> worker                              |       1.3
                                      |   > worker
                                      | > Production project
[Copy to Edge project ->]              | [<- Copy to local images]
-------------------------------------------------------------------------
Transfers: Local -> Company / Edge    service-api:1.4    Verifying
```

All names and contents in this sketch are illustrative, not seed data.

## One target dropdown per pane

The main workspace is the two panes and a bottom transfer queue. A global toolbar
provides Manage targets and transfer history; a separate sidebar is not required
for choosing local versus remote content. This supersedes the earlier two-stage
type/registry selector sketch, while preserving the primary domain objects.

An open dropdown can group entries by type, but each selectable row is a concrete
target. Type headings are not an extra required selection step. For example:

```text
Local images
  This computer / Docker       current configured engine
Registries
  Company registry             registry.example.com
  Test registry                test-registry.example.com
File system
  Downloads                    /Users/example/Downloads
  Image exports                /Users/example/Images
------------------------------------------------------
Add target...
Manage targets...
```

Both dropdowns show the same target catalog. Either pane can choose any listed
target, including the same target as the other pane. Distinguish equal friendly
names by type and address/path. Show connection state without silently hiding
disconnected targets. Selecting a target reveals its contents below:

| Target kind | Selected concrete location | Browsable hierarchy |
| --- | --- | --- |
| Registry | A saved registry endpoint | Projects, named images, versions |
| Local images | An existing Docker-compatible engine/context | Named image groups and versions, with identities/platforms in details |
| File system | A drive/root or saved folder | Folders, image archive files, inspected image entries |

The user's “local disk” destination for pull/load means the local engine's image
store, labeled Local images in the UI. It is separate from a filesystem directory:
pull/load produces an engine image, while export produces an archive file.
The app does not ask users to locate or modify the engine's internal storage folder.

Selecting a registry loads its accessible project tree, with contextual sign-in
when needed. It does not automatically select the first project as a write target.

Choose the destination project by selecting or opening its node. Project favorites
may jump directly to that node and select the appropriate registry in the header.
This preserves project-first upload intent while allowing registry-wide browsing.
Add project reuses the contextual setup journey; adding a saved registry connection
does not create remote projects. A project appears once across credentials.
The pane header identifies the active account and access state separately from
the registry selection. Account changes refresh the listing without changing
queued task identities.

Each pane retains independent search, selection, refresh, tree expansion and
navigation state for its locations. Both may display the same target kind;
their roles are not permanently local-left/remote-right. Switching targets
clears incompatible source/target selections and updates the visible location.
Queued tasks retain their captured destinations. Restoring a prior location must
show its path/account and validate access before another action can be confirmed.

Use one expandable tree per pane, with full-width names and compact spacing while
retaining readable body text. Expand/collapse changes
visibility only; it is distinct from selecting rows or choosing a target project.
Children load on demand with provider pagination preserved. Loading, empty,
permission-denied, unsupported enumeration and network failure are distinct node
states. Known project/reference entry remains available when enumeration is not.

There is no flat all-images table: registry enumeration is not universally available,
and opening a location must not recursively fetch every repository and tag.
Nodes support keyboard navigation and visible focus; the path field and Up action
remain available for direct navigation.
Opening an image shows versions or details; it never starts a transfer. Version
rows show platform scope or offer details. Multiple names and untagged local images
remain identifiable without invented names. Image groupings are not writable
filesystem folders, and expanding a version does not select a single platform.

The app remembers the last workspace when practical and presents disconnected
locations honestly. Remembering a project or account preference does not imply
that credentials remain valid. Source selection survives destination sign-in.

## Manage the targets offered by both dropdowns

Manage targets is a dedicated screen/dialog available from the global toolbar and
the bottom of either dropdown. It lists saved targets with name, type, address/path,
connection state and Edit / Remove actions. Add target asks for kind only during
configuration, then opens the corresponding setup form.

- Registry: friendly name, endpoint, provider and transport/trust configuration.
  Add or associate credentials separately, allowing several accounts on one endpoint.
  A non-destructive connection check reports what was actually verified. Save the
  endpoint as one target; accounts are access settings, not duplicate target rows.
- Local images: choose an existing engine/context and friendly name; verify access
  without starting an engine or changing system permissions. Default discovery can
  offer the current context. Missing/unavailable engines have an actionable state.
- File system: choose a real drive/root/folder and friendly name; show the path and
  observed accessibility. An unavailable folder remains distinguishable from empty.

Saving updates both dropdowns. If setup began from a pane, allow opening the saved
target there while preserving the other pane's state. Registry projects and images
remain server content discovered below that target, not copied into configuration.
Project shortcuts may locate a node without redefining the registry target.

Removing a target removes local configuration, not remote images, local engine
images or files. Credential deletion is an explicit separate access operation.
Target edits/removal cannot redirect queued tasks: tasks keep their reviewed
location/identity, and report missing access or configuration if they cannot proceed.
Changing an endpoint for an in-progress workflow requires renewed target review.

## Copying to the other pane

1. Choose a concrete target from each pane's single dropdown. For a registry
   destination, choose its project in the tree. If only the registry root is
   open, prompt for a project before queueing; never pick one implicitly.
2. Select a source image/version. The primary action names the destination:
   Copy to Edge project, Copy to local images, or Export image file.
3. Open one compact review with source, destination, account, target image name,
   version and platform scope. Pre-fill editable names from the source. Selecting
   a target image group can pre-fill its name; the review makes that mapping explicit.
4. Resolve missing sign-in or account choice in that review, preserving the inputs.
   If a name/version exists, explain same content or request an explicit conflict
   decision. Do not add a separate mandatory tagging operation.
5. Confirm once to queue the copy. Capture source content, destination reference,
   source/target account contexts, platform scope and the conflict decision.
6. Verify completion, refresh the matching visible destination, and provide Open
   destination even when that pane has since navigated elsewhere. Do not replace
   the user's current location just because a background task finished.

Both panes can provide directionally explicit buttons. Selection determines the
source; a plain Upload button with an ambiguous target is insufficient. Drag and
drop may later open the same review, but the visible button is the initial primary
interaction. Opening or double-clicking a row must not upload it immediately.

Changing another pane, browsing a different project or changing a default account
does not retarget queued work. Source and target may use different accounts even
on the same registry. Unsupported credential isolation must be explained before
queueing, as described in the project model.

### Routes determined by the selected targets

| Source -> Destination | Operation and verified result |
| --- | --- |
| Registry -> Registry | Copy image content into the selected destination project |
| Registry -> Local images | Pull into the chosen engine; image appears in its image list |
| Local images -> Registry | Upload into the selected project; local image retained |
| Registry or local images -> File system | Export a selected image to an archive in the chosen folder |
| File system image archive -> Local images | Inspect/select entries and load into the chosen engine |
| File system image archive -> Registry | Inspect/select entries and upload into the selected project |

The user initiates a copy; the review can explain Pull, Upload, Export or Import
in context. Determine the route from target kinds and actual selected content,
not from left/right position or filename alone. Invalid source/target selections
explain what is needed before queueing. Generic file management is outside the
image-transfer scope; two filesystem panes do not imply arbitrary file copy support.

## Image files in the same workspace

A filesystem target browses real folders and shows the actual path. Recognized image
archives can be inspected or expanded to reveal image entries and platform scope,
allowing a selected entry to be copied into a project or local engine. Other files
remain recognizable as ordinary files and cannot initiate image import. Unsupported
or invalid archives show an inspection result rather than an empty image list.
Selecting a multi-image file opens entry/scope selection before import. An archive
is not an ordinary writable image directory.

Copying an image toward a filesystem folder means Export image file. Pre-fill
the selected folder, then ask for filename, format and supported content scope.
Use the same transfer queue. Export creates a new snapshot with an explicit file
overwrite decision when necessary. Browsing an existing archive does not imply
that a copy will append to or rewrite it. Copy to local images and export to a
file remain distinct actions and results.

The file's original project, if known, is provenance rather than a forced import
destination. Deleting an archive file does not delete remote or local images.

## Transfer queue and interruption

The bottom queue shows image/version, exact source and destination, account labels,
state and available actions. A dedicated Transfers view provides history/details.
Only show byte counts, speed, percentage and remaining time where measured. Unknown
totals use an activity state; copied bytes are not proof of verified completion.

Use Queued, Copying, Verifying, Completed, Needs sign-in, Failed, Cancelling,
Cancelled and Needs checking as appropriate. Offer Cancel for active work, Retry
for an eligible failure and Open destination after verified completion. A cancelled
or interrupted transfer reports what is known to have been written. Byte-level
resume and Pause are not promised unless the transport actually supports them.

Sign-in returns to the affected task under its selected identity. Permission denial
never silently changes accounts. An uncertain result is reconciled before retry;
batch results stay per item so completed items need not be blindly repeated.

## Cleanup inside a location

Select an image/version and use Delete from [project/local engine]. Keep deletion
separate from the copy arrows, review the actual affected names/content and check
delete permission independently. Selecting a project shortcut or saved connection
offers Remove shortcut / Remove account, not an ambiguous image deletion.

Version labels are references to content. Rename, drag, or a missing row must not
silently become content deletion. Creating another name uses the copy review;
removing the old name, if supported, uses a separate explicit deletion review.
Automatic move, folder synchronization and deleting unmatched images are outside
this initial proposal. See the user journeys for cleanup scope and verification.

## Walkthrough acceptance

- First use: read a real local image, add a destination project in the other pane,
  sign in, copy with a clear target name, find the remote result and retain the source.
- Pane selection: choose concrete registry, engine and folder targets directly
  from either single dropdown, expand their trees and change targets
  without retaining an incompatible target or retargeting queued transfers.
- Target management: add/edit a registry with several account identities, an engine
  target and a folder target; both dropdowns update. Removing configuration never
  deletes target content or silently changes a queued task's identity/destination.
- Tree: expansion does not transfer or select content implicitly; loading,
  paging, empty and unreadable states are distinguishable without requiring full-catalog enumeration.
- Reverse direction: open a project image, copy into local images, and find it in
  the engine list rather than receiving only an archive file.
- Project to project: use independent access contexts, including different accounts
  on the same registry; explain an unsupported route before queueing.
- File route: export, inspect the resulting archive, select an entry and import it
  into a different project; original project and file are retained.
- Interruption: navigate while copying, resolve an expired login or version conflict
  without losing inputs, and inspect mixed batch results without retargeting tasks.
- Cleanup: delete from one specified place with accurate alias/platform scope;
  unrelated project copies, local copies and archive files remain independent.

These are acceptance scenarios for later implementation, not completed live tests.
The existing From/To screen is a starting point; its current feature set and Profile
coupling do not establish that this interaction proposal has been implemented.

## Interactive prototype

The [Commander HTML prototype](prototype/commander/index.html) demonstrates these
interactions using sample data, including one concrete-target dropdown per pane,
shared target management, transfer review and the queue. Open it directly in a
browser. See its [notes](prototype/commander/README.md) for simulated boundaries and
browser verification. This does not implement the native Go/Fyne migration.

## Reference inspected

The official [Double Commander Drives list documentation](https://doublecmd.github.io/doc/en/help.html#iface_drive_btn)
includes a screenshot of a dropdown containing concrete drive locations and
describes independent left/right drive-list commands. This is the navigation
reference inspected here; the unified registry/engine/folder target catalog and
management workflow above are ImagePort's proposed adaptation, not features
claimed for Double Commander. Inspection used documentation/screenshots, not a
live desktop session with that application.
