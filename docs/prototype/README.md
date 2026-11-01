# ImagePort HTML prototypes

Open [index.html](index.html) or [commander/index.html](commander/index.html) directly
in a browser. The default entry opens the latest Commander design. No installation,
server, build step or network access is required. The earlier design is retained
at [classic.html](classic.html), with its [original notes](classic-README.md).

## Current design

The Commander prototype implements the [transfer workspace proposal](../transfer-workspace.md):

- Two independently browsable panes, each with **one dropdown of concrete targets**.
  Registry endpoints, local image engines and filesystem folders share that list.
- Expandable project/image/version trees, a list alternative, breadcrumbs and search.
- One target manager for both dropdowns: add, edit, remove and open configurations.
  Multiple accounts remain separate from registry/project identity.
- One copy review with destination project, image name/tag, platform scope and access
  identity. Different source/destination accounts are supported on the same host.
- Contextual sign-in retaining inputs, explicit overwrite consent, read-only access
  feedback, scoped deletion, and transfer cancellation/retry/history.
- English/Chinese switching, light/dark appearance and keyboard-accessible controls.

These are simulated interactions, separate from the Go/Fyne application's actual
feature set. See [commander/README.md](commander/README.md) for the routes, fixtures,
verification and implementation files.
