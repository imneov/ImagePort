# Changelog

Notable project changes will be recorded here. Releases will use semantic
versioning once public versions are published.

## Unreleased

### Added

- Copy images between visible From and To locations, with independent destination
  names/tags and no prerequisite local tagging step.
- Cross-registry copies using independent saved connections and verified digests.
- Independent archive import destinations and unambiguous saved connection labels.
- Local Docker listing, inspection and optional tagging through official SDKs.
  Uploads capture immutable image IDs and engine identities.
- Harbor project/repository/tag browsing with server totals and pagination.

- Initial Go/Fyne desktop image-management implementation and regclient adapter.
- Repository contribution policies, security guidance and English-first standards.
- Persistent transfer tasks and secure archive/credential boundaries.
- Automated checks and an explicit native-platform acceptance matrix.
- Persistent English/Simplified Chinese interface selection without resetting inputs or tasks.
- Actionable guidance when repository catalog listing is unavailable or not permitted.
- Digest labels for unnamed OCI roots in archive selection.
- Approved five-screen application layout with a toolbar, sidebar and detail drawers.
- Real archive history, connection cards, scoped favorites, cursor navigation,
  transfer status filters, command search and remembered light/dark appearance.
- Guards against stale registry/inspection responses after inputs or navigation change.

No production release has been published.
