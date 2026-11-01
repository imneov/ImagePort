{
  const app = window.ImagePort;
  /** Escape arbitrary fixture or entered text for HTML/attribute interpolation. */
  app.escape = (value) => String(value).replace(/[&<>"']/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[character]);
  /** Return a trusted decorative SVG by icon name; unknown names use the box icon. */
  app.icon = (name) => `<svg viewBox="0 0 24 24" aria-hidden="true">${app.icons[name] || app.icons.box}</svg>`;
  /** Render a labeled action button; action is interpreted by the central event router. */
  app.button = (label, action, icon = "", style = "", extra = "") => `<button ${extra.includes("type=") ? "" : 'type="button"'} class="btn ${style}" data-action="${app.escape(action)}" ${extra}>${icon ? app.icon(icon) : ""}${app.escape(app.t(label))}</button>`;
  /** Render an accessible icon-only action with a localized name and tooltip. */
  app.iconButton = (icon, label, action, extra = "") => `<button type="button" class="icon-button" data-action="${app.escape(action)}" aria-label="${app.escape(app.t(label))}" title="${app.escape(app.t(label))}" ${extra}>${app.icon(icon)}</button>`;
  /** Render a small state label from a stable English status value. */
  app.status = (value) => {
    const tone = ({ Connected: "good", Completed: "good", Verified: "good", Running: "blue", Queued: "blue", Failed: "bad", Disconnected: "bad" })[value] || "";
    return `<span class="badge ${tone}"><span class="dot"></span>${app.escape(app.t(value))}</span>`;
  };
  /** Render the shared page heading with optional trusted action markup. */
  app.heading = (title, description, actions = "") => `<div class="page-heading"><div><h1 tabindex="-1">${app.t(title)}</h1><p class="description">${app.t(description)}</p></div><div class="heading-actions">${actions}</div></div>`;
  /** Render architecture chips from explicit platform values. */
  app.platforms = (platforms) => `<div class="platforms">${platforms.map((platform) => `<span class="badge">${app.escape(platform.replace("linux/", ""))}</span>`).join("")}</div>`;
  /** Render an empty-result panel; action markup is application-owned. */
  app.empty = (title, description, action = "") => `<div class="empty">${app.icon("search")}<h2>${app.t(title)}</h2><p>${app.t(description)}</p>${action}</div>`;
  /** Return the selected sample registry, defaulting to the first fixture. */
  app.currentRegistry = () => app.registries.find((registry) => registry.id === app.state.registry) || app.registries[0];
  /** Resolve an image fixture by ID; unknown IDs return undefined. */
  app.image = (id) => app.images.find((image) => image.id === id);
  /** Return the complete illustrative tag list, keeping counts and conflict checks aligned. */
  app.tagNames = (image) => [...new Set([image.tag, "stable", "latest", ...Array.from({ length: image.tags }, (_, index) => `build-${image.tags - index}`)])].slice(0, image.tags);
  /** Build a readable, non-authenticated sample image reference. */
  app.reference = (image) => `${app.currentRegistry().host}/${image.name}:${image.tag}`;
  /** Return a stable illustrative digest; it is never used for real registry work. */
  app.digest = () => "sha256:50084cfd55a79d93bcf91767a02f5b8a6cb1af1bea1595eac687101d7728f72c";
  /** Render a labeled input, escaping values and allowing trusted HTML input attributes. */
  app.field = (label, name, value = "", attributes = "", help = "") => `<div class="form-field"><label for="field-${name}">${app.t(label)}</label><input id="field-${name}" name="${name}" value="${app.escape(value)}" ${attributes}>${help ? `<small>${app.t(help)}</small>` : ""}</div>`;
  /** Render registry choices with a selected ID and a stable accessible label. */
  app.registryOptions = (selected) => app.registries.map((registry) => `<option value="${app.escape(registry.id)}" ${registry.id === selected ? "selected" : ""}>${app.escape(registry.name)} · ${app.escape(registry.host)}</option>`).join("");
  /** Render the common footer, identifying the dataset as a discussion prototype. */
  app.footer = () => `<footer class="workspace-footer"><span>${app.t("Interactive prototype · Sample data")}</span><span>ImagePort <span aria-hidden="true">/</span> ${app.t("Image management")}</span></footer>`;
}
