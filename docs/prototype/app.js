{
  const app = window.ImagePort;
  const text = (value) => app.escape(app.t(value));
  app.navigation = [
    { page: "images", label: "Images", icon: "box" },
    { page: "registries", label: "Registries", icon: "server" },
    { page: "archives", label: "Local archives", icon: "folder" },
    { page: "transfers", label: "Transfers", icon: "arrows" },
    { page: "settings", label: "Settings", icon: "settings" },
  ];

  /** Read validated presentation preferences; unavailable storage keeps in-memory defaults. */
  app.loadPreferences = () => {
    try {
      const language = localStorage.getItem("imageport-prototype-language");
      const theme = localStorage.getItem("imageport-prototype-theme");
      if (["en", "zh-Hans"].includes(language)) app.state.language = language;
      if (["light", "dark"].includes(theme)) app.state.theme = theme;
    } catch (error) {
      console.warn("Prototype preferences are unavailable; using in-memory defaults.");
    }
  };

  /** Persist only language/theme, never entered account or registry details. */
  app.savePreferences = () => {
    try {
      localStorage.setItem("imageport-prototype-language", app.state.language);
      localStorage.setItem("imageport-prototype-theme", app.state.theme);
    } catch (error) {
      app.toast("Preferences cannot be saved; changes apply only in this tab.");
    }
  };

  /** Count the active tasks used by both the global indicator and task center. */
  app.activeCount = () => app.transfers.filter((task) => ["Running", "Queued"].includes(task.status)).length;

  /** Render a primary navigation item with stable current-page semantics. */
  app.navItem = (item) => `<button type="button" class="nav-button" data-action="navigate:${item.page}" ${app.state.page === item.page ? 'aria-current="page"' : ""} title="${text(item.label)}">${app.icon(item.icon)}<span class="nav-text">${text(item.label)}</span>${item.page === "transfers" ? `<span id="sidebar-active-count" class="count" ${app.activeCount() ? "" : "hidden"}>${app.activeCount()}</span>` : ""}</button>`;

  /** Render the persistent application frame: top bar, sidebar, and route content. */
  app.shell = () => {
    const page = app.navigation.find((item) => item.page === app.state.page);
    const views = { images: app.imagesPage, registries: app.registriesPage, archives: app.archivesPage, transfers: app.transfersPage, settings: app.settingsPage };
    const active = app.activeCount();
    const mobile = matchMedia("(max-width: 600px)").matches;
    return `<div class="app ${app.state.collapsed ? "collapsed" : ""} ${mobile && app.state.mobileMenu ? "mobile-menu" : ""}">
      <header class="topbar"><a class="brand" href="#images" data-action="navigate:images" aria-label="ImagePort"><span class="brand-mark">${app.icon("layers")}</span><span class="brand-name">ImagePort <small>DESKTOP</small></span></a>
        <button type="button" class="top-action" data-action="sidebar" aria-label="${text("Toggle sidebar")}" aria-expanded="${mobile ? !!app.state.mobileMenu : !app.state.collapsed}">${app.icon("sidebar")}</button><span class="top-separator"></span><div class="breadcrumb"><span class="muted">${text("Workspace")}</span>${app.icon("right")}<span>${text(page.label)}</span></div>
        <div class="top-actions"><button type="button" class="search-trigger" data-action="search" aria-label="${text("Search anything…")}">${app.icon("search")}<span>${text("Search anything…")}</span><kbd>⌘ K</kbd></button>
        <button type="button" class="top-action task-top" data-action="navigate:transfers" aria-label="${active} ${text("active transfers")}">${active ? '<span class="activity-dot"></span>' : app.icon("arrows")}<span id="active-count">${active} ${text("In progress").toLowerCase()}</span></button>
        <button type="button" class="top-action" data-action="language" aria-label="${text("Switch language")}" title="${text("Switch language")}">${app.icon("globe")}<span>${app.state.language === "en" ? "中" : "EN"}</span></button>
        <button type="button" class="top-action" data-action="theme" aria-label="${text("Toggle theme")}" title="${text("Toggle theme")}" aria-pressed="${app.state.theme === "dark"}">${app.icon(app.state.theme === "dark" ? "sun" : "moon")}</button></div>
      </header>
      <button type="button" class="mobile-veil" data-action="sidebar" aria-label="${text("Close")}"></button>
      <aside class="sidebar"><p class="nav-label">${text("Management")}</p><nav aria-label="${text("Management")}">${app.navigation.filter((item) => item.page !== "settings").map(app.navItem).join("")}</nav>
        <div class="sidebar-bottom">${app.navItem(app.navigation[4])}<button type="button" class="nav-button" data-action="help" title="${text("Help & shortcuts")}">${app.icon("help")}<span class="nav-text">${text("Help & shortcuts")}</span></button>
          <div class="local-status">${app.icon("shield")}<span>${text("Local workspace")}</span><span class="version">v0.1</span></div><p class="prototype-label">${text("Interactive prototype · Sample data")}</p></div>
      </aside><main id="main" class="workspace" tabindex="-1"><div class="content">${views[app.state.page]()}</div></main>
    </div>`;
  };

  /** Refresh page markup while keeping keyboard focus on a surviving named control. */
  app.render = () => {
    const focused = document.activeElement;
    const focusId = focused?.id;
    const focusAction = focused?.dataset?.action;
    document.documentElement.lang = app.state.language;
    document.body.classList.toggle("dark", app.state.theme === "dark");
    document.querySelector("#application").innerHTML = app.shell();
    const target = focusId ? document.getElementById(focusId) : focusAction ? document.querySelector(`[data-action="${CSS.escape(focusAction)}"]`) : null;
    target?.focus({ preventScroll: true });
  };

  /** Navigate only to implemented pages; preserve filters and task state across pages. */
  app.navigate = (page) => {
    if (!app.navigation.some((item) => item.page === page)) return;
    document.querySelector("#modal").close();
    app.state.page = page;
    app.state.mobileMenu = false;
    history.replaceState(null, "", `#${page}`);
    app.render();
    document.querySelector("main h1")?.focus({ preventScroll: true });
  };

  /** Show a brief accessible notification without modifying the application data. */
  app.toast = (message) => {
    const toast = document.querySelector("#toast");
    clearTimeout(app.toastTimer);
    toast.textContent = app.t(message);
    toast.classList.add("visible");
    app.toastTimer = setTimeout(() => toast.classList.remove("visible"), 3800);
  };

  /** Resolve selected tag context only while its drawer is visible. */
  app.actionImage = (id) => {
    const modal = document.querySelector("#modal");
    return modal.open && modal.classList.contains("drawer") && app.modalImage?.id === id ? app.modalImage : app.image(id);
  };

  /** Update the repository results without disrupting the focused search field. */
  app.refreshImages = () => {
    const results = document.querySelector("#image-results");
    if (results) results.innerHTML = app.imageResults();
  };

  /** Change presentation preferences and preserve the current data model. */
  app.preference = (key, value) => {
    if (key === "language" && !["en", "zh-Hans"].includes(value)) return;
    if (key === "theme" && !["light", "dark"].includes(value)) return;
    app.state[key] = value;
    app.savePreferences();
    app.render();
  };

  /** Dispatch parameterized prototype actions from buttons and links. */
  app.dispatch = (action) => {
    const [name, ...parts] = action.split(":");
    const value = parts.join(":");
    const routes = {
      navigate: () => app.navigate(value), image: () => app.openImage(value), download: () => app.openDownload(app.actionImage(value)),
      "copy-tag": () => app.openCopy(app.actionImage(value)), connection: () => app.openConnection(value),
      favorite: () => { app.state.favorites.has(value) ? app.state.favorites.delete(value) : app.state.favorites.add(value); app.render(); },
      browse: () => { app.state.registry = value; app.state.pageNumber = 1; app.navigate("images"); },
      "image-tab": () => { app.state.favoritesOnly = value === "favorites"; app.state.pageNumber = 1; app.render(); },
      "transfer-tab": () => { app.state.transferFilter = value; app.render(); },
      "detail-tab": () => app.openImage(app.state.selectedImage, value, app.modalImage.tag),
      "image-tag": () => app.openImage(app.state.selectedImage, "overview", value),
      archive: () => app.openArchive(Number(value)), "upload-archive": () => { app.state.archiveName = app.archives[Number(value)].name; app.openImport(2); },
      task: () => app.openTask(value), cancel: () => app.confirmTask(value, "cancel"), retry: () => app.confirmTask(value, "retry"),
      verify: () => app.toast("Simulated result verified"),
    };
    if (routes[name]) routes[name](); else app.simpleAction(name);
  };

  /** Handle unparameterized global actions with real page/modal state changes. */
  app.simpleAction = (action) => {
    const routes = {
      close: () => document.querySelector("#modal").close(), search: app.openSearch, help: app.openHelp,
      sidebar: () => { if (matchMedia("(max-width: 600px)").matches) app.state.mobileMenu = !app.state.mobileMenu; else app.state.collapsed = !app.state.collapsed; app.render(); },
      language: () => app.preference("language", app.state.language === "en" ? "zh-Hans" : "en"),
      theme: () => app.preference("theme", app.state.theme === "light" ? "dark" : "light"),
      refresh: () => app.toast("Connected · sample environment"),
      "clear-filters": () => { Object.assign(app.state, { query: "", platform: "all", favoritesOnly: false, pageNumber: 1 }); app.render(); },
      "next-page": () => { app.state.pageNumber++; app.refreshImages(); }, "previous-page": () => { app.state.pageNumber = Math.max(1, app.state.pageNumber - 1); app.refreshImages(); },
      import: () => { app.state.archiveName = ""; app.openImport(); }, "choose-file": () => document.querySelector("#archive-file").click(),
      "sample-archive": () => app.chooseArchive("api-gateway-v2.8.1.tar"), "import-next": () => app.openImport(app.state.importStep + 1), "import-back": () => app.openImport(app.state.importStep - 1),
      "copy-reference": app.copyReference, location: app.openLocation,
      folder: () => app.showModal(app.dialogLayout("Archive location", "Sample file location", `<div class="code-block"><code>${app.escape(app.state.downloadLocation || "~/Downloads/ImagePort")}</code></div>`, app.button("Close", "close"))),
    };
    routes[action]?.();
  };

  /** Retain only the chosen tar filename; local bytes are never inspected by this prototype. */
  app.chooseArchive = (name) => {
    if (!name.toLowerCase().endsWith(".tar")) return app.toast("Only .tar image archives are supported in this prototype.");
    app.state.archiveName = name;
    document.querySelector("#chosen-archive").innerHTML = app.archiveSelection();
    document.querySelector("#import-continue").disabled = false;
  };

  /** Copy the displayed reference, reporting denied clipboard access explicitly. */
  app.copyReference = async () => {
    try {
      await navigator.clipboard.writeText(app.reference(app.modalImage));
      app.toast("Reference copied");
    } catch (error) {
      app.toast("Copy failed. Select and copy the reference manually.");
    }
  };

  /** Open the editable sample download location without invoking OS file permissions. */
  app.openLocation = () => {
    app.showModal(app.dialogLayout("Choose location", "Default download location", app.field("Download location", "location", app.state.downloadLocation || "~/Downloads/ImagePort", "required"), app.button("Cancel", "close") + app.button("Save", "", "check", "primary", 'type="submit"'), "location-form"));
  };

  /** Detect an existing sample target using repository, tag and registry context. */
  app.targetExists = (target, registry) => {
    if (registry !== "production") return false;
    return app.images.some((image) => app.tagNames(image).some((tag) => `${image.name}:${tag}` === target));
  };

  /** Enforce platform selection, source/target difference, and explicit overwrite consent. */
  app.validateCopy = () => {
    const form = document.querySelector("#copy-form");
    if (!form) return;
    const target = form.elements.target.value.trim();
    const sameSource = target === `${app.modalImage.name}:${app.modalImage.tag}` && form.elements.registry.value === app.state.registry;
    const exists = app.targetExists(target, form.elements.registry.value) && !sameSource;
    document.querySelector("#copy-conflict").classList.toggle("hidden", !exists);
    form.elements.overwrite.required = exists;
    form.elements.target.setCustomValidity(sameSource ? app.t("The target tag must differ from the source tag.") : "");
    document.querySelector("#copy-error").textContent = sameSource ? app.t("The target tag must differ from the source tag.") : "";
    document.querySelector("#queue-copy").disabled = !form.elements.platform.value || !target || sameSource || (exists && !form.elements.overwrite.checked);
  };

  /** Require overwrite consent only when the sample upload destination already exists. */
  app.validateUpload = () => {
    const form = document.querySelector("#upload-form");
    if (!form) return;
    const exists = app.targetExists(form.elements.target.value.trim(), form.elements.registry.value);
    document.querySelector("#upload-conflict").classList.toggle("hidden", !exists);
    form.elements.overwrite.required = exists;
    if (!exists) form.elements.overwrite.checked = false;
  };

  /** Save non-secret connection metadata for this prototype tab only. */
  app.saveConnection = (form) => {
    const registry = { id: app.editingRegistry || `sample-${Date.now()}`, name: form.elements.name.value.trim(), host: form.elements.host.value.trim(), kind: "OCI Registry", repositories: 0, connected: true };
    const existing = app.registries.findIndex((item) => item.id === app.editingRegistry);
    if (existing >= 0) app.registries[existing] = { ...app.registries[existing], ...registry }; else app.registries.push(registry);
    form.reset();
    document.querySelector("#modal").close();
    app.render();
    app.toast("Connection saved in this prototype");
  };

  /** Create a simulated task and run a bounded visual preview; never perform I/O. */
  app.queueTask = (operation, image, platform, destination) => {
    const task = { id: Math.max(...app.transfers.map((item) => item.id)) + 1, operation, image, platform, destination, status: "Queued", progress: 0, size: "—", time: "Just now", stage: "Queued", attempt: 1 };
    app.transfers.unshift(task);
    document.querySelector("#modal").close();
    app.state.transferFilter = "all";
    app.navigate("transfers");
    app.toast("Transfer queued");
    app.animateTask(task);
  };

  /** Advance only a newly queued task; cancellation and retries stop its prior timer. */
  app.animateTask = (task) => {
    const timer = setInterval(() => {
      if (!["Queued", "Running"].includes(task.status)) { clearInterval(timer); return; }
      task.status = "Running";
      task.stage = "Transferring layers";
      task.progress = Math.min(100, task.progress + 16);
      task.size = `${task.progress}%`;
      if (task.progress === 100) {
        task.status = "Completed";
        task.stage = "Digest verified";
        clearInterval(timer);
        app.toast("Simulated transfer completed");
      }
      const results = document.querySelector("#transfer-results");
      if (results) results.innerHTML = app.transferResults();
      const recent = document.querySelector("#recent-transfers");
      if (recent) recent.innerHTML = app.transfers.slice(0, 2).map(app.activityRow).join("");
      app.refreshTaskIndicators();
    }, 1400);
  };

  /** Keep shared activity indicators and status filters consistent with current tasks. */
  app.refreshTaskIndicators = () => {
    const active = app.activeCount();
    const count = document.querySelector("#active-count");
    if (count) { count.textContent = `${active} ${app.t("In progress").toLowerCase()}`; count.parentElement.setAttribute("aria-label", `${active} ${app.t("active transfers")}`); }
    const sidebar = document.querySelector("#sidebar-active-count");
    if (sidebar) { sidebar.textContent = active; sidebar.hidden = !active; }
    const tabs = document.querySelector("#transfer-tabs");
    if (tabs) tabs.innerHTML = app.transferTabs();
  };

  /** Confirm task cancellation/retry before changing the sample attempt state. */
  app.confirmTask = (id, action) => {
    app.pendingTask = { id: Number(id), action };
    const title = action === "cancel" ? "Cancel this transfer?" : "Retry transfer?";
    const description = action === "cancel" ? "Cancellation is a request. Published remote data is not rolled back." : "The original source digest is retained. A new attempt will be created.";
    app.showModal(app.dialogLayout(title, description, "", app.button("Cancel", "close") + app.button(action === "cancel" ? "Confirm cancellation" : "Queue retry", "", "", "primary", 'type="submit"'), "task-action-form"));
  };

  /** Apply a confirmed operation to the chosen task and retain its identity on retry. */
  app.applyTaskAction = () => {
    const task = app.transfers.find((item) => item.id === app.pendingTask.id);
    if (!task) return;
    if (app.pendingTask.action === "cancel") { task.status = "Cancelled"; task.stage = "Cancelled"; }
    else { task.status = "Queued"; task.stage = "Queued"; task.progress = 0; task.attempt = (task.attempt || 1) + 1; app.animateTask(task); }
    document.querySelector("#modal").close();
    app.render();
  };

  /** Route submitted forms through native HTML validation before touching fixture state. */
  app.submit = (form) => {
    if (form.id === "copy-form") app.validateCopy();
    if (form.id === "upload-form") app.validateUpload();
    if (!form.reportValidity()) return;
    if (form.id === "connection-form") return app.saveConnection(form);
    if (form.id === "task-action-form") return app.applyTaskAction();
    if (form.id === "location-form") {
      app.state.downloadLocation = form.elements.location.value.trim();
      document.querySelector("#modal").close(); app.render(); app.toast("Download location updated"); return;
    }
    if (form.id === "download-form") return app.queueTask("Download", `${app.modalImage.name}:${app.modalImage.tag}`, form.elements.platform.value, `${form.elements.location.value}/${form.elements.filename.value}`);
    if (form.id === "copy-form") return app.queueTask("Copy tag", `${app.modalImage.name}:${app.modalImage.tag} → ${form.elements.target.value}`, form.elements.platform.value, form.elements.target.value);
    if (form.id === "upload-form") return app.queueTask("Upload", form.elements.target.value, "linux/amd64, linux/arm64", form.elements.target.value);
  };

  /** Handle live filtering and dialog-specific conditional form behavior. */
  app.input = (target) => {
    if (target.id === "repository-search") { app.state.query = target.value; app.state.pageNumber = 1; app.refreshImages(); }
    if (target.id === "command-search") document.querySelector("#command-results").innerHTML = app.commandResults(target.value);
    if (target.closest("#copy-form")) {
      if (["target", "registry"].includes(target.name)) target.form.elements.overwrite.checked = false;
      app.validateCopy();
    }
    if (target.closest("#upload-form")) {
      if (["target", "registry"].includes(target.name)) target.form.elements.overwrite.checked = false;
      app.validateUpload();
    }
    if (target.closest("#download-form") && target.name === "platform") {
      document.querySelector("#queue-download").disabled = false;
      document.querySelector("#field-filename").value = `${app.modalImage.id}-${app.modalImage.tag}-${target.value.replaceAll("/", "-")}.tar`;
    }
    if (target.name === "archive-entry") document.querySelector("#import-continue").disabled = !target.checked;
  };

  /** Handle selections without losing typed search terms or silently resetting task history. */
  app.change = (target) => {
    if (target.id === "registry-picker") { app.state.registry = target.value; app.state.pageNumber = 1; app.render(); }
    if (target.id === "platform-filter") { app.state.platform = target.value; app.state.pageNumber = 1; app.refreshImages(); }
    if (target.id === "language-setting") app.preference("language", target.value);
    if (target.id === "theme-setting") app.preference("theme", target.value);
    if (target.id === "concurrency-setting") app.state.concurrency = Number(target.value);
    if (target.id === "archive-file" && target.files[0]) app.chooseArchive(target.files[0].name);
    if (target.id === "connection-auth") {
      const account = target.value === "account";
      document.querySelector("#connection-account").hidden = !account;
      document.querySelectorAll("#connection-account input").forEach((input) => { input.required = account; input.disabled = !account; });
    }
  };

  /** Register delegated UI events once, including keyboard access and archive drag/drop. */
  app.bind = () => {
    document.addEventListener("click", (event) => {
      const button = event.target.closest("[data-action]");
      if (!button || button.disabled) return;
      event.preventDefault();
      if (button.type === "submit" && button.form) { button.form.requestSubmit(); return; }
      app.dispatch(button.dataset.action);
    });
    document.addEventListener("submit", (event) => { event.preventDefault(); app.submit(event.target); });
    document.addEventListener("input", (event) => app.input(event.target));
    document.addEventListener("change", (event) => app.change(event.target));
    document.addEventListener("keydown", (event) => { if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") { event.preventDefault(); app.openSearch(); } });
    matchMedia("(max-width: 600px)").addEventListener("change", () => { app.state.mobileMenu = false; app.render(); });
    document.addEventListener("dragover", (event) => { const zone = event.target.closest("#archive-dropzone"); if (zone) { event.preventDefault(); zone.classList.add("dragover"); } });
    document.addEventListener("dragleave", (event) => event.target.closest("#archive-dropzone")?.classList.remove("dragover"));
    document.addEventListener("drop", (event) => { const zone = event.target.closest("#archive-dropzone"); if (!zone) return; event.preventDefault(); zone.classList.remove("dragover"); if (event.dataTransfer.files[0]) app.chooseArchive(event.dataTransfer.files[0].name); });
    document.querySelector("#modal").addEventListener("close", () => { document.querySelectorAll('#modal input[type="password"]').forEach((input) => { input.value = ""; }); });
  };

  app.loadPreferences();
  const initialPage = location.hash.slice(1);
  if (app.navigation.some((item) => item.page === initialPage)) app.state.page = initialPage;
  app.bind();
  app.render();
}
