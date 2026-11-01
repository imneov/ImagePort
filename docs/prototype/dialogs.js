{
  const app = window.ImagePort;
  const text = (value) => app.escape(app.t(value));

  /** Open a native modal, retaining browser-managed focus trapping and Escape behavior. */
  app.showModal = (content, drawer = false) => {
    const modal = document.querySelector("#modal");
    if (modal.open) modal.close();
    modal.className = drawer ? "drawer" : "";
    modal.innerHTML = content;
    modal.showModal();
  };

  /** Build a consistently labeled dialog with optional form submission. */
  app.dialogLayout = (title, description, body, footer, form = "") => {
    const content = `<header class="dialog-header"><div><h2 id="dialog-title">${text(title)}</h2>${description ? `<p>${text(description)}</p>` : ""}</div>${app.iconButton("close", "Close", "close")}</header><div class="dialog-body">${body}</div><footer class="dialog-footer">${footer}</footer>`;
    return form ? `<form id="${form}">${content}</form>` : content;
  };

  /** Render a source summary with escaped image reference and trusted icon. */
  app.imageSummary = (image) => `<div class="dialog-image"><span class="repo-icon ${image.color || "gray"}">${app.icon("box")}</span><div><strong>${app.escape(image.name)}<span class="muted">:${app.escape(image.tag)}</span></strong><small>${app.escape(app.currentRegistry().host)} <span aria-hidden="true">·</span> ${image.size}</small></div></div>`;

  /** Render explicit platform radio choices; no architecture is silently selected. */
  app.platformChoices = (image) => `<fieldset style="border:0;padding:0;margin:0"><legend class="field-label">${text("Select a platform")}</legend>${image.platforms.map((platform) => `<label class="radio-option"><input type="radio" name="platform" value="${app.escape(platform)}" required><span class="mono">${app.escape(platform)}</span><span class="badge">${image.size}</span></label>`).join("")}<p class="platform-hint">${text("Choose explicitly. Your computer's architecture is not selected automatically.")}</p></fieldset>`;

  /** Open a detail drawer for a selected repository/tag without leaving the image list. */
  app.openImage = (id, tab = "overview", tag = "") => {
    const original = app.image(id);
    if (!original) return;
    app.state.selectedImage = id;
    app.modalImage = { ...original, tag: tag || original.tag };
    const image = app.modalImage;
    const body = `${app.imageSummary(image)}${app.tabs([["Overview", "overview"], ["Tags", "tags", image.tags]], tab, "detail-tab")}${tab === "tags" ? app.tagDetails(image) : app.overviewDetails(image)}`;
    const footer = `${app.button("Copy tag", `copy-tag:${id}`, "copy")}${app.button("Download", `download:${id}`, "download", "primary")}`;
    app.showModal(app.dialogLayout("Image", image.name, body, footer), true);
  };

  /** Render digest, provenance and available platforms for the illustrative selected image. */
  app.overviewDetails = (image) => `<dl class="detail-grid"><dt>${text("Tag")}</dt><dd><span class="badge">${app.escape(image.tag)}</span></dd><dt>${text("Size")}</dt><dd>${image.size}</dd><dt>${text("Updated")}</dt><dd>${text(image.updated)}</dd><dt>${text("Format")}</dt><dd>OCI Image</dd></dl>
    <section class="detail-section"><h3>${text("Available platforms")}</h3>${image.platforms.map((platform) => `<div class="tag-row"><span class="mono">${platform}</span><span class="muted small">${image.size}</span></div>`).join("")}</section>
    <section class="detail-section"><h3>${text("Image reference")}</h3><div class="code-block"><code>${app.escape(app.reference(image))}</code>${app.iconButton("copy", "Copy reference", "copy-reference")}</div></section>
    <section class="detail-section"><h3>${text("Manifest digest")}</h3><div class="code-block"><code>${app.digest()}</code></div></section>`;

  /** Render selectable sample tags; selection is carried through subsequent transfer forms. */
  app.tagDetails = (image) => `<p class="small muted">${text("Select a tag to inspect or transfer.")}</p>${app.tagNames(image).map((tag) => `<div class="tag-row"><span class="tag-info"><span class="badge">${app.escape(tag)}</span><br><small>${text(image.updated)}</small></span>${app.platforms(image.platforms)}${app.iconButton("right", "View details", `image-tag:${tag}`)}</div>`).join("")}`;

  /** Open a platform-gated download form using the selected source tag. */
  app.openDownload = (image) => {
    app.modalImage = image;
    const body = `${app.imageSummary(image)}${app.platformChoices(image)}${app.field("Download location", "location", app.state.downloadLocation || "~/Downloads/ImagePort", "required")}${app.field("Filename", "filename", `${image.id}-${image.tag}.tar`, 'required pattern="[^\\x2f\\x5c]+[.]tar"')}`;
    const footer = app.button("Cancel", "close") + app.button("Queue download", "", "download", "primary", 'type="submit" id="queue-download" disabled');
    app.showModal(app.dialogLayout("Download image", "Choose a platform and download an OCI image archive.", body, footer, "download-form"));
  };

  /** Open a copy form with explicit platform and conditional overwrite consent. */
  app.openCopy = (image) => {
    app.modalImage = image;
    const destination = `<div class="form-field"><label for="copy-registry">${text("Target registry")}</label><select id="copy-registry" name="registry">${app.registryOptions(app.state.registry)}</select></div>${app.field("Target repository:tag", "target", `${image.name}:copy`, 'required pattern="[a-z0-9][a-z0-9._\\x2f\\x2d]*:[A-Za-z0-9_][A-Za-z0-9_.\\x2d]*"')}`;
    const warning = `<div id="copy-conflict" class="hidden"><div class="callout warning">${app.icon("alert")}<p>${text("This tag already exists. Replacing it requires confirmation.")}</p></div><label class="check-label" style="margin-top:12px"><input id="overwrite-consent" type="checkbox" name="overwrite">${text("I understand this replaces the current target tag.")}</label></div><p id="copy-error" class="form-error" role="alert"></p>`;
    const footer = app.button("Cancel", "close") + app.button("Queue copy", "", "copy", "primary", 'type="submit" id="queue-copy" disabled');
    app.showModal(app.dialogLayout("Copy to a new tag", "The source image stays unchanged.", app.imageSummary(image) + app.platformChoices(image) + destination + warning, footer, "copy-form"));
  };

  /** Open registry editing; entered passwords are discarded without persistence or requests. */
  app.openConnection = (id) => {
    const registry = app.registries.find((item) => item.id === id);
    app.editingRegistry = registry?.id || "";
    const authentication = `<div class="form-field"><label for="connection-auth">${text("Authentication")}</label><select id="connection-auth" name="auth"><option value="account">${text("Username & password")}</option><option value="anonymous">${text("Anonymous")}</option></select></div>`;
    const account = `<div id="connection-account"><div class="field-row">${app.field("Username", "username", "", `autocomplete="off" placeholder="${text("Enter your username")}" required`)}${app.field("Password or token", "password", "", `type="password" autocomplete="off" placeholder="${text("Enter password or token")}" required`)}</div></div>`;
    const storage = `<div class="form-field"><label for="connection-storage">${text("Credential storage")}</label><select id="connection-storage" name="storage"><option>${text("System keychain")}</option><option>${text("Session only")}</option></select><small>${text("This is sample data. No credentials are sent or stored.")}</small></div>`;
    const body = `${app.field("Display name", "name", registry?.name || "", "required")}${app.field("Registry address", "host", registry?.host || "", 'required placeholder="registry.example.com" pattern="[A-Za-z0-9.\\-]+(:[0-9]+)?"', "Use a hostname and optional port. HTTPS is the default.")}${authentication}${account}${storage}`;
    const footer = app.button("Cancel", "close") + app.button("Save connection", "", "check", "primary", 'type="submit"');
    app.showModal(app.dialogLayout(registry ? "Edit connection" : "New registry connection", "Keep your images organized across registries.", body, footer, "connection-form"));
  };

  /** Render a three-step import flow; only names are read from local files. */
  app.openImport = (step = 1) => {
    app.state.importStep = step;
    const labels = ["Choose archive", "Inspect", "Destination registry"];
    const steps = `<div class="steps">${labels.map((label, index) => `${index ? '<span class="step-line"></span>' : ""}<span class="step ${index + 1 === step ? "active" : ""}"><span>${index + 1}</span>${text(label)}</span>`).join("")}</div>`;
    const body = steps + (step === 1 ? app.importChoose() : step === 2 ? app.importInspect() : app.importDestination());
    const action = step === 3 ? app.button("Queue upload", "", "upload", "primary", 'type="submit"') : app.button("Continue", "import-next", "right", "primary", `id="import-continue" ${step === 1 && app.state.archiveName ? "" : "disabled"}`);
    const footer = app.button(step === 1 ? "Cancel" : "Back", step === 1 ? "close" : "import-back") + action;
    app.showModal(app.dialogLayout("Import archive", step === 3 ? "Verify the destination before uploading." : "Choose an OCI Layout or docker-save .tar file.", body, footer, step === 3 ? "upload-form" : ""));
  };

  /** Render file selection with a clearly identified sample archive alternative. */
  app.importChoose = () => `<div class="dropzone" id="archive-dropzone">${app.icon("upload")}<h3>${text("Drop an image archive here")}</h3><p>OCI Layout / docker-save<br>.tar</p>${app.button("Choose file", "choose-file", "folder")}<input id="archive-file" type="file" accept=".tar" hidden></div><button type="button" class="sample-link" data-action="sample-archive">${text("Use sample archive")}</button><div id="chosen-archive">${app.state.archiveName ? app.archiveSelection() : ""}</div>`;

  /** Render only a selected filename, never reading or displaying file contents. */
  app.archiveSelection = () => `<div class="archive-selected">${app.icon("folder")}<div><strong>${app.escape(app.state.archiveName)}</strong><small>${app.t("OCI Layout")} / docker-save</small></div>${app.icon("check")}</div>`;

  /** Render illustrative inspection results, with explicit entry consent required. */
  app.importInspect = () => `<div class="callout">${app.icon("help")}<p>${text("All data and transfers are simulated. Nothing connects to a real registry.")}</p></div><h3 style="margin-top:22px">${text("One image entry found")}</h3><p class="description">${text("Select the entry you want to upload.")}</p><label class="radio-option"><input type="radio" name="archive-entry" value="sample-entry" required><span><strong>platform/api-gateway:v2.8.1</strong><br><small class="muted">linux/amd64 · linux/arm64</small></span></label><p class="platform-hint">${text("All platforms in this entry are preserved.")}</p><div class="code-block"><code>${app.escape(app.state.archiveName)}</code></div>`;

  /** Render destination confirmation for the selected sample archive entry. */
  app.importDestination = () => `<div class="archive-selected">${app.icon("folder")}<div><strong>${app.escape(app.state.archiveName)}</strong><small>platform/api-gateway:v2.8.1 · 2 ${text("Platforms")}</small></div></div><div class="form-field"><label for="upload-registry">${text("Target registry")}</label><select id="upload-registry" name="registry">${app.registryOptions(app.state.registry)}</select></div>${app.field("Target repository:tag", "target", "platform/api-gateway:imported", 'required pattern="[a-z0-9][a-z0-9._\\x2f\\x2d]*:[A-Za-z0-9_][A-Za-z0-9_.\\x2d]*"')}<div id="upload-conflict" class="hidden"><div class="callout warning">${app.icon("alert")}<p>${text("This tag already exists. Replacing it requires confirmation.")}</p></div><label class="check-label" style="margin-top:12px"><input name="overwrite" type="checkbox">${text("I understand this replaces the current target tag.")}</label></div>`;

  /** Open a sample archive's readable metadata with an upload action. */
  app.openArchive = (index) => {
    const archive = app.archives[index];
    if (!archive) return;
    const body = `<div class="archive-selected">${app.icon("folder")}<strong>${app.escape(archive.name)}</strong></div><dl class="detail-grid"><dt>${text("Image")}</dt><dd>${app.escape(archive.image)}</dd><dt>${text("Format")}</dt><dd>${archive.format}</dd><dt>${text("Platform")}</dt><dd class="mono">${archive.platform}</dd><dt>${text("Size")}</dt><dd>${archive.size}</dd><dt>${text("Status")}</dt><dd>${app.status("Verified")}</dd></dl><h3 style="margin-bottom:12px">${text("File SHA256")}</h3><div class="code-block"><code>${app.digest().replace("sha256:", "")}</code></div>`;
    app.showModal(app.dialogLayout("Archive details", "Content checked", body, app.button("Close", "close") + app.button("Upload archive", `upload-archive:${index}`, "upload", "primary")), true);
  };

  /** Open task history for one sample transfer, including failure and retry actions. */
  app.openTask = (id) => {
    const task = app.transfers.find((item) => item.id === Number(id));
    if (!task) return;
    const body = `<div style="margin-bottom:22px">${app.status(task.status)}</div><div class="code-block"><code>${app.escape(task.image)}</code></div><dl class="detail-grid"><dt>${text("Operation")}</dt><dd>${text(task.operation)}</dd><dt>${text("Platform")}</dt><dd class="mono">${task.platform}</dd><dt>${text("Stage")}</dt><dd>${text(task.stage)}</dd><dt>${text("Attempt")}</dt><dd>${task.attempt || 1}</dd><dt>${text("Created")}</dt><dd>${text(task.time)}</dd><dt>${text("Destination")}</dt><dd class="mono">${app.escape(task.destination || (task.operation === "Download" ? "~/Downloads/ImagePort" : "harbor.example.com"))}</dd></dl><h3 style="margin-bottom:12px">${text("Expected digest")}</h3><div class="code-block"><code>${app.digest()}</code></div>${task.status === "Failed" ? `<div class="callout warning section">${app.icon("alert")}<p>${text(task.stage)}</p></div>` : ""}`;
    const action = ["Running", "Queued"].includes(task.status) ? app.button("Request cancellation", `cancel:${task.id}`, "close") : ["Failed", "Cancelled"].includes(task.status) ? app.button("Retry with same source", `retry:${task.id}`, "refresh", "primary") : app.button("Verify result", `verify:${task.id}`, "shield");
    app.showModal(app.dialogLayout("Task details", `#IP-${String(task.id).padStart(4, "0")}`, body, app.button("Close", "close") + action), true);
  };

  /** Open a command palette that searches the implemented prototype destinations. */
  app.openSearch = () => {
    app.showModal(`<h2 id="dialog-title" class="sr-only">${text("Quick navigation")}</h2><label class="command-search">${app.icon("search")}<input id="command-search" placeholder="${text("Search pages and actions…")}" aria-label="${text("Search pages and actions…")}" autofocus>${app.iconButton("close", "Close", "close")}</label><div class="command-list" id="command-results">${app.commandResults("")}</div>`);
  };

  /** Filter known destinations using both their English and displayed names. */
  app.commandResults = (query) => {
    const matches = app.navigation.filter((item) => `${item.label} ${app.t(item.label)}`.toLowerCase().includes(query.toLowerCase()));
    return `<p class="command-title">${text("Quick navigation")}</p>${matches.map((item) => `<button class="command-item" type="button" data-action="navigate:${item.page}">${app.icon(item.icon)}${text(item.label)}<span class="muted">${text("Go to")}</span></button>`).join("") || `<p class="command-title">${text("No matching pages")}</p>`}`;
  };

  /** Open help explaining keyboard access and the simulation boundary. */
  app.openHelp = () => {
    const body = `<h3>${text("Prototype scope")}</h3><p class="description">${text("All data and transfers are simulated. Nothing connects to a real registry.")}</p><p class="description">${text("Pick a registry, open an image, select a platform, then queue a download or copy.")}</p><h3 style="margin-top:24px">${text("Keyboard shortcuts")}</h3><dl class="detail-grid"><dt>${text("Open search")}</dt><dd><kbd>⌘ / Ctrl + K</kbd></dd><dt>${text("Dismiss dialog")}</dt><dd><kbd>Esc</kbd></dd></dl>`;
    app.showModal(app.dialogLayout("Help & shortcuts", "", body, app.button("Close", "close")));
  };
}
