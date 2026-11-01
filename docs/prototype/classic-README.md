# ImagePort interface prototype

This is the earlier interface design. The latest design is the
[Commander prototype](commander/index.html).

Open `classic.html` directly in a browser. No installation, build tool, CDN, network
access, Go service or registry credentials are required. This is a clickable HTML
design artifact, separate from the Go/Fyne application.

The visual reference is [Shadcn Admin](https://shadcn-admin.netlify.app/). The
prototype follows its compact tables, restrained surfaces and navigation structure.
It introduces a full-width top bar and a persistent left menu for a desktop image
management workflow. It uses original HTML/CSS rather than the reference's source.

## Discussion workflows

- Images: switch registry, search, filter platforms, favorite and page repositories.
- Image drawer: inspect overview/tags, select a tag, copy its reference.
- Download: explicitly choose a platform and location, queue a simulated transfer.
- Copy: choose a platform and destination. Targeting an existing sample repository
  with `stable`, `latest`, or its displayed tag requires overwrite consent.
- Import: select a `.tar` filename or use the sample archive, explicitly select
  its sample entry, then confirm the destination. No local archive bytes are read.
- Transfers: inspect tasks, cancel an active transfer, retry a failed transfer.
- Registries: add/edit sample connection metadata. Passwords are discarded.
- Settings: language, theme, download location and concurrency controls.
- Global controls: collapse sidebar, English/Chinese, light/dark, command search
  (`Cmd/Ctrl+K`), and keyboard-accessible modal dialogs.

All images, progress, digest values, inspection results and connection statuses are
illustrative fixtures. New transfers animate as simulations. The initial in-progress
fixture stays still for discussion. No real transfer, connection, keychain write or
file publication occurs. Only language/theme persist in browser local storage.
All other sample state resets on reload; this is independent of the Go database.

## Proposed information architecture

The initial page is **Images**, scoped by a registry selector in the workspace.
Top navigation contains global search, task activity, language and appearance.
The sidebar separates image work from connections, local files, tasks and settings.
Opening image details preserves the current list underneath a right drawer.
Multi-step actions use focused dialogs rather than expanding the main page.

The layout was approved and implemented in the five-screen Go/Fyne application.
This HTML artifact remains a design reference with illustrative data; the native
application uses real registry responses and durable transfer history.

## Verification

Checked in Chromium on 2026-10-26 at 1440×1000 and 390×844, including direct
`file://` access: repository search/pagination, image drawer, explicit-platform
download, overwrite consent and renewed consent after target changes, sample import,
anonymous connection creation, retry, command search, language/theme switching,
and the compact navigation menu. The final browser session reported no uncaught
JavaScript errors. This validates prototype interactions, not registry operations.

## Files

`data.js` contains explicit sample fixtures, `catalog.js` English/Chinese messages,
`components.js` shared markup, `views.js` page layouts, `dialogs.js` focused flows,
`app.js` state/events. Styles are split into base/layout (`styles.css`), components,
dialogs, and responsive behavior. There is no framework runtime.
