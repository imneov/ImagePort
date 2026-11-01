{
  const app = window.ImagePort;
  const text = (value) => app.escape(app.t(value));

  /** Render navigation filters; each tuple contains a label, stable value, and count. */
  app.tabs = (items, current, action) => `<div class="tabs">${items.map(([label, value, count]) => `<button class="tab ${current === value ? "active" : ""}" data-action="${action}:${value}" aria-pressed="${current === value}">${text(label)}${count === undefined ? "" : `<span class="tab-count">${count}</span>`}</button>`).join("")}</div>`;

  /** Return the registry-scoped fixture set before filtering or pagination. */
  app.registryImages = () => {
    if (!app.currentRegistry().connected) return [];
    if (app.state.registry === "production") return app.images;
    return app.state.registry === "dockerhub" ? app.images.filter((image) => ["nginx", "redis", "alpine"].includes(image.id)) : [];
  };

  /** Apply search, favorites and platform filters to the whole scoped fixture set. */
  app.filteredImages = () => app.registryImages().filter((image) => {
    const matchesQuery = `${image.name}:${image.tag}`.toLowerCase().includes(app.state.query.toLowerCase());
    const matchesPlatform = app.state.platform === "all" || image.platforms.includes(app.state.platform);
    return matchesQuery && matchesPlatform && (!app.state.favoritesOnly || app.state.favorites.has(image.id));
  });

  /** Render the registry context without making connectivity an image permission claim. */
  app.registryContext = () => {
    const registry = app.currentRegistry();
    return `<section class="registry-context" aria-label="${text("Current registry")}">
      <div class="registry-icon">${app.icon("server")}</div>
      <div><div class="registry-choice"><select id="registry-picker" class="registry-select" aria-label="${text("Current registry")}">${app.registryOptions(registry.id)}</select>${app.icon("down")}</div><p class="registry-address">${app.escape(registry.host)}</p></div>
      <div class="context-right">${app.status(registry.connected ? "Connected" : "Disconnected")}<span class="muted">${text("Last synced just now")}</span>${app.button("Refresh", "refresh", "refresh")}</div>
    </section>`;
  };

  /** Render one image row with direct detail/download actions and a working favorite. */
  app.imageRow = (image) => {
    const favorite = app.state.favorites.has(image.id);
    return `<tr>
      <td><div class="image-name"><span class="repo-icon ${image.color}">${app.icon("box")}</span><div><button class="repo-link" data-action="image:${image.id}">${app.escape(image.name)}</button><div class="repo-sub">${text("Latest")} <span class="mono">${app.escape(image.tag)}</span></div></div></div></td>
      <td><span class="badge">${image.tags} ${text("Tags").toLowerCase()}</span></td>
      <td>${app.platforms(image.platforms)}</td><td>${image.size}</td><td class="muted">${text(image.updated)}</td>
      <td><div class="actions" style="justify-content:flex-end">
        <button class="icon-button ${favorite ? "favorite" : ""}" aria-pressed="${favorite}" aria-label="${text(favorite ? "Remove from favorites" : "Add to favorites")}" title="${text(favorite ? "Remove from favorites" : "Add to favorites")}" data-action="favorite:${image.id}">${app.icon("star")}</button>
        ${app.iconButton("download", "Download image", `download:${image.id}`)}${app.iconButton("right", "View image details", `image:${image.id}`)}
      </div></td></tr>`;
  };

  /** Render correctly paged image rows and controls derived from the filtered total. */
  app.imageResults = () => {
    const images = app.filteredImages();
    const pageSize = 6;
    const pageCount = Math.max(1, Math.ceil(images.length / pageSize));
    app.state.pageNumber = Math.min(app.state.pageNumber, pageCount);
    const start = (app.state.pageNumber - 1) * pageSize;
    if (!app.currentRegistry().connected) return app.empty("Disconnected", "Connect to this registry to browse images.", app.button("Manage connection", `connection:${app.state.registry}`, "link"));
    if (!images.length) return app.empty("No repositories found", "Try a different search or platform filter.", app.button("Clear filters", "clear-filters"));
    return `<div class="table-wrap"><table><thead><tr>${["Repository", "Tags", "Platforms", "Size", "Updated", "Actions"].map((label) => `<th scope="col">${text(label)}</th>`).join("")}</tr></thead><tbody>${images.slice(start, start + pageSize).map(app.imageRow).join("")}</tbody></table></div>
      <div class="pagination"><span>${text("Showing")} ${start + 1}–${Math.min(start + pageSize, images.length)} <span class="muted">${text("of")} ${images.length} ${text("repositories")}</span></span>
      <div class="pagination-actions"><span>${text("Page")} ${app.state.pageNumber} / ${pageCount}</span>${app.iconButton("left", "Previous page", "previous-page", app.state.pageNumber === 1 ? "disabled" : "")}${app.iconButton("right", "Next page", "next-page", app.state.pageNumber === pageCount ? "disabled" : "")}</div></div>`;
  };

  /** Render compact task activity; progress comes from sample state, never network work. */
  app.activityRow = (task) => `<div class="activity-row">
    <div class="activity-icon">${app.icon(task.operation === "Upload" ? "upload" : task.operation === "Copy tag" ? "copy" : "download")}</div>
    <div class="activity-info"><button class="repo-link" data-action="task:${task.id}"><strong>${app.escape(task.image)}</strong></button><p>${text(task.operation)} <span aria-hidden="true">·</span> ${app.escape(task.platform)}</p></div>
    ${task.status === "Running" ? `<div class="activity-progress"><div class="progress-meta"><span>${text(task.stage)}</span><span>${task.progress}%</span></div><div class="progress" role="progressbar" aria-label="${app.escape(task.image)}" aria-valuenow="${task.progress}" aria-valuemin="0" aria-valuemax="100"><span style="width:${task.progress}%"></span></div></div>` : `<div class="activity-status">${app.status(task.status)}<small>${text(task.time)}</small></div>`}
  </div>`;

  /** Render the primary image workspace with current-context controls and task activity. */
  app.imagesPage = () => `${app.heading("Images", "Browse, inspect, and move your container images.", app.button("Import archive", "import", "upload") + app.button("Add registry", "connection:new", "plus", "primary"))}
    ${app.registryContext()}
    ${app.tabs([["All repositories", "all", app.registryImages().length], ["Favorites", "favorites", app.registryImages().filter((image) => app.state.favorites.has(image.id)).length]], app.state.favoritesOnly ? "favorites" : "all", "image-tab")}
    <div class="toolbar"><label class="filter-search">${app.icon("search")}<span class="sr-only">${text("Search repositories…")}</span><input id="repository-search" type="search" placeholder="${text("Search repositories…")}" value="${app.escape(app.state.query)}"></label>
      <div class="filter-controls"><select class="select-control" id="platform-filter" aria-label="${text("Platforms")}"><option value="all">${text("All platforms")}</option>${["linux/amd64", "linux/arm64", "linux/arm/v7"].map((platform) => `<option ${app.state.platform === platform ? "selected" : ""}>${platform}</option>`).join("")}</select><span class="badge">OCI Registry</span></div></div>
    <section id="image-results" aria-label="${text("All repositories")}">${app.imageResults()}</section>
    <section class="section"><div class="section-heading"><h2>${text("Recent transfers")}</h2>${app.button("View all transfers", "navigate:transfers", "right", "text")}</div><div id="recent-transfers" class="activity-list">${app.transfers.slice(0, 2).map(app.activityRow).join("")}</div></section>${app.footer()}`;

  /** Render a saved registry card with separate browsing and editing actions. */
  app.registryCard = (registry) => `<article class="connection-card"><div class="card-top"><span class="registry-icon">${app.icon("server")}</span>${app.status(registry.connected ? "Connected" : "Disconnected")}</div>
    <h2>${app.escape(registry.name)}</h2><p class="mono">${app.escape(registry.host)}</p><div class="connection-meta"><span class="badge">${app.escape(registry.kind)}</span><span>${registry.repositories} ${text("repositories")}</span></div>
    <div class="card-actions">${app.button(registry.connected ? "Browse images" : "Reconnect", registry.connected ? `browse:${registry.id}` : `connection:${registry.id}`, registry.connected ? "box" : "refresh")}${app.iconButton("settings", "Edit connection", `connection:${registry.id}`)}</div></article>`;

  /** Render registry administration without mixing it into the primary image table. */
  app.registriesPage = () => `${app.heading("Registries", "Manage your registry connections in one place.", app.button("Add registry", "connection:new", "plus", "primary"))}
    <div class="card-grid">${app.registries.map(app.registryCard).join("")}</div>
    <div class="callout section">${app.icon("shield")}<div><strong>${text("Connect once. Work across registries.")}</strong><br>${text("Credentials stay in your system keychain or in this session.")}</div></div>${app.footer()}`;

  /** Render local archive rows from explicit prototype fixtures. */
  app.archivesPage = () => `${app.heading("Local archives", "Your downloaded and imported image archives.", app.button("Downloads folder", "folder", "folder") + app.button("Import archive", "import", "plus", "primary"))}
    <div class="table-wrap"><table><thead><tr>${["Archive", "Format", "Platform", "Size", "Status", "Actions"].map((label) => `<th scope="col">${text(label)}</th>`).join("")}</tr></thead><tbody>${app.archives.map((archive, index) => `<tr>
      <td><div class="image-name"><span class="repo-icon gray">${app.icon("folder")}</span><div><button class="repo-link" data-action="archive:${index}">${app.escape(archive.name)}</button><div class="repo-sub">${app.escape(archive.image)}</div></div></div></td><td><span class="badge">${archive.format}</span></td><td class="mono">${archive.platform}</td><td>${archive.size}</td><td>${app.status("Verified")}</td><td>${app.iconButton("upload", "Upload archive", `upload-archive:${index}`)}${app.iconButton("right", "Inspect archive", `archive:${index}`)}</td>
    </tr>`).join("")}</tbody></table></div>${app.footer()}`;

  /** Return transfer fixtures matching the selected status category. */
  app.filteredTasks = () => app.transfers.filter((task) => app.state.transferFilter === "all" || (app.state.transferFilter === "active" ? ["Running", "Queued"].includes(task.status) : task.status === app.state.transferFilter));

  /** Render a transfer row with status-specific actions and inspectable detail. */
  app.transferRow = (task) => `<tr><td><div class="image-name"><span class="repo-icon gray">${app.icon(task.operation === "Upload" ? "upload" : "download")}</span><div><button class="repo-link" data-action="task:${task.id}">${app.escape(task.image)}</button><p class="repo-sub">${app.escape(task.platform)}</p></div></div></td><td>${text(task.operation)}</td><td>${app.status(task.status)}</td>
    <td>${task.status === "Running" ? `<div style="min-width:110px"><div class="progress-meta"><span>${task.size}</span><span>${task.progress}%</span></div><div class="progress"><span style="width:${task.progress}%"></span></div></div>` : `<span class="small muted">${text(task.stage)}</span>`}</td><td class="muted">${text(task.time)}</td><td>${["Running", "Queued"].includes(task.status) ? app.iconButton("close", "Cancel task", `cancel:${task.id}`) : ["Failed", "Cancelled"].includes(task.status) ? app.iconButton("refresh", "Retry task", `retry:${task.id}`) : ""}${app.iconButton("right", "View task", `task:${task.id}`)}</td></tr>`;

  /** Render transfer filters and preserve all tasks when switching views. */
  app.transferResults = () => {
    const tasks = app.filteredTasks();
    if (!tasks.length) return app.empty("No tasks in this view", "Your transfers will appear here.");
    return `<div class="table-wrap"><table><thead><tr>${["Image", "Operation", "Status", "Progress", "Created", "Actions"].map((label) => `<th scope="col">${text(label)}</th>`).join("")}</tr></thead><tbody>${tasks.map(app.transferRow).join("")}</tbody></table></div>`;
  };

  /** Render the task center; counts come from the same task array used by the rows. */
  app.transfersPage = () => `${app.heading("Transfers", "Track transfers and inspect verified results.")}
    <div id="transfer-tabs">${app.transferTabs()}</div>
    <section id="transfer-results" aria-label="${text("All tasks")}">${app.transferResults()}</section>${app.footer()}`;

  /** Derive every transfer filter count from the same underlying fixture collection. */
  app.transferTabs = () => app.tabs([["All tasks", "all", app.transfers.length], ["In progress", "active", app.activeCount()], ["Completed", "Completed", app.transfers.filter((task) => task.status === "Completed").length], ["Failed", "Failed", app.transfers.filter((task) => task.status === "Failed").length]], app.state.transferFilter, "transfer-tab");

  /** Render one settings row with application-owned control markup. */
  app.settingRow = (title, description, control) => `<div class="setting-row"><div><h3>${text(title)}</h3><p>${text(description)}</p></div>${control}</div>`;

  /** Render persisted presentation preferences; no live credential management occurs. */
  app.settingsPage = () => `${app.heading("Settings", "Customize ImagePort to fit your workflow.")}
    <div class="settings-layout"><aside class="settings-menu"><p>${text("General")}</p></aside><section class="settings-panel"><h2>${text("Appearance & language")}</h2><p class="description">${text("Make this workspace feel like yours.")}</p>
    ${app.settingRow("Language", "Changes apply immediately and are remembered.", `<select id="language-setting" class="select-control" aria-label="${text("Language")}"><option value="en" ${app.state.language === "en" ? "selected" : ""}>English</option><option value="zh-Hans" ${app.state.language === "zh-Hans" ? "selected" : ""}>简体中文</option></select>`)}
    ${app.settingRow("Appearance", "Choose a light or dark interface.", `<select id="theme-setting" class="select-control" aria-label="${text("Appearance")}"><option value="light" ${app.state.theme === "light" ? "selected" : ""}>${text("Light")}</option><option value="dark" ${app.state.theme === "dark" ? "selected" : ""}>${text("Dark")}</option></select>`)}
    ${app.settingRow("Default download location", "Choose where image archives are saved.", app.button("Change", "location", "folder"))}<p class="small mono muted">${app.escape(app.state.downloadLocation || "~/Downloads/ImagePort")}</p>
    ${app.settingRow("Concurrent transfers", "Maximum number of transfers running at once.", `<select id="concurrency-setting" class="select-control" aria-label="${text("Concurrent transfers")}">${[1, 2, 3, 4].map((count) => `<option ${count === (app.state.concurrency || 2) ? "selected" : ""}>${count}</option>`).join("")}</select>`)}
    ${app.settingRow("Credentials", "Use your system keychain, or explicitly choose session-only storage.", `<span class="badge">${text("System keychain")}</span>`)}
    <div class="callout section">${app.icon("box")}<div><strong>ImagePort</strong><br>${text("Interactive prototype · Sample data")}</div></div></section></div>${app.footer()}`;
}
