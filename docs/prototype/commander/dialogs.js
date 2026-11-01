/* Focused copy, inspect, cleanup and history dialogs. Native dialog traps focus. */
(() => {
  'use strict';
  const app = window.ImagePort;
  const escapeHtml = app.escape;
  const translate = app.t;
  app.dialogRenderers = {};

  /** Render a shared dialog header. @param {string} title Title. @param {string} description Description. @param {string} icon Icon. @returns {string} Markup. */
  app.dialogHeader = function dialogHeader(title, description = '', icon = 'box') {
    return `<header class="dialog-header"><span class="dialog-symbol">${app.icon(icon)}</span><div><h2 id="dialog-title">${escapeHtml(title)}</h2>${description ? `<p>${escapeHtml(description)}</p>` : ''}</div>${app.iconButton('close-dialog', 'close', translate('close'))}</header>`;
  };

  /** Render a labeled form field. @param {string} label Label. @param {string} input Trusted field HTML. @param {string} hint Help. @returns {string} Markup. */
  app.field = function field(label, input, hint = '') {
    return `<label class="field"><span>${escapeHtml(label)}</span>${input}${hint ? `<small>${escapeHtml(hint)}</small>` : ''}</label>`;
  };

  /** Render account options with explicit permissions. @param {object} target Registry. @param {string} selected ID. @returns {string} Options. */
  app.accountOptions = function accountOptions(target, selected) {
    return (target.accounts || []).map(account => `<option value="${escapeHtml(account.id)}" ${account.id === selected ? 'selected' : ''}>${escapeHtml(app.label(account))} · ${escapeHtml(translate(account.permissions.includes('delete') ? 'canMaintain' : account.permissions.includes('write') ? 'canPublish' : 'readOnly'))}${account.signedIn ? '' : ` · ${escapeHtml(translate('signInRequired'))}`}</option>`).join('');
  };

  /** Show a modal, retaining a return focus target. @param {object} draft Modal state. @returns {void} */
  app.showDialog = function showDialog(draft) {
    app.closeMenu();
    const modal = document.getElementById('modal');
    if (!modal.open) app.dialogReturnFocus = document.activeElement;
    app.state.dialog = draft;
    app.renderDialog();
    if (!modal.open) modal.showModal();
    modal.querySelector('[autofocus]')?.focus();
  };

  /** Close the modal and restore its initiating control. @returns {void} */
  app.closeDialog = function closeDialog() {
    const modal = document.getElementById('modal');
    if (modal.open) modal.close();
    app.state.dialog = null; modal.innerHTML = '';
    const previous = app.dialogReturnFocus;
    if (previous?.isConnected) previous.focus({ preventScroll: true });
    else document.getElementById('target-left')?.focus({ preventScroll: true });
  };

  /** Render current modal state. @returns {void} */
  app.renderDialog = function renderDialog() {
    const draft = app.state.dialog;
    if (!draft) return;
    const modal = document.getElementById('modal');
    modal.className = `dialog dialog-${draft.type}`;
    modal.innerHTML = app.dialogRenderers[draft.type](draft);
    if (draft.type === 'copy') app.updateCopyReview();
  };

  /** Render the copy's source/destination summary. @param {object} draft Review. @returns {string} Markup. */
  function copySummary(draft) {
    const target = app.target(draft.targetId);
    return `<div class="copy-summary"><div><span class="eyebrow">${escapeHtml(translate('source'))}</span><strong>${app.icon(draft.sourceTarget.kind)}${escapeHtml(app.label(draft.sourceTarget))}</strong><span class="mono source-reference" title="${escapeHtml(app.sourceReference(draft))}">${escapeHtml(app.sourceReference(draft))}</span></div><span class="summary-arrow">${app.icon('arrow')}</span><div><span class="eyebrow">${escapeHtml(translate('targetLocation'))}</span><strong>${app.icon(target.kind)}${escapeHtml(app.label(target))}</strong><span>${escapeHtml(translate(app.copyRoute(draft)))}</span></div></div>`;
  }

  /** Render destination naming fields. @param {object} draft Review. @returns {string} Markup. */
  function copyNameFields(draft) {
    const target = app.target(draft.targetId);
    if (target.kind === 'filesystem') return `<div class="form-columns">${app.field(translate('filename'), `<input id="copy-filename" name="filename" value="${escapeHtml(draft.filename)}" maxlength="200" required />`)}${app.field(translate('format'), `<select name="format"><option value="oci" ${draft.format === 'oci' ? 'selected' : ''}>OCI archive (.tar)</option><option value="docker" ${draft.format === 'docker' ? 'selected' : ''}>Docker archive (.tar)</option></select>`)}</div>`;
    const projects = app.collect(target.tree, 'project');
    return `${target.kind === 'registry' ? app.field(translate('project'), `<select id="copy-project" name="projectId" required><option value="">${escapeHtml(translate('chooseProject'))}</option>${projects.map(project => `<option value="${escapeHtml(project.id)}" ${draft.projectId === project.id ? 'selected' : ''}>${escapeHtml(project.name)}</option>`).join('')}</select>`) : ''}<div class="form-columns">${app.field(translate('imageName'), `<input id="copy-name" name="name" value="${escapeHtml(draft.name)}" maxlength="240" required />`)}${app.field(translate('versionName'), `<input id="copy-tag" name="tag" value="${escapeHtml(draft.tag)}" maxlength="128" required />`)}</div>`;
  }

  /** Render content and access selections. @param {object} draft Review. @returns {string} Markup. */
  function copyContentFields(draft) {
    const target = app.target(draft.targetId);
    const source = app.copySource(draft);
    return `${draft.entries.length > 1 ? app.field(translate('archiveEntry'), `<select name="entryId">${draft.entries.map(entry => `<option value="${escapeHtml(entry.id)}" ${entry.id === draft.entryId ? 'selected' : ''}>${escapeHtml(entry.image)}:${escapeHtml(entry.name)}</option>`).join('')}</select>`) : ''}${app.field(translate('content'), `<select name="platform"><option value="all" ${draft.platform === 'all' ? 'selected' : ''}>${escapeHtml(translate('allPlatforms'))} (${source.platforms.length})</option>${source.platforms.map(platform => `<option value="${escapeHtml(platform)}" ${draft.platform === platform ? 'selected' : ''}>${escapeHtml(platform)}</option>`).join('')}</select>`)}${target.kind === 'registry' ? app.field(translate('account'), `<select name="accountId" id="copy-account">${app.accountOptions(target, draft.accountId)}</select>`) : ''}<div id="access-feedback" role="status"></div>`;
  }

  /** Describe existing content with an explicit replacement choice. @param {object} draft Review. @returns {string} Markup. */
  function conflictMarkup(draft) {
    const conflict = app.copyConflict(draft);
    if (!conflict.existing) return '';
    if (conflict.same) return `<div class="notice success">${app.icon('check')}<span>${escapeHtml(translate('sameContent'))}</span></div>`;
    const filesystem = app.target(draft.targetId).kind === 'filesystem';
    return `<div class="notice warning"><div>${app.icon('info')}<span><strong>${escapeHtml(translate(filesystem ? 'fileConflict' : 'conflict'))}</strong><small>${escapeHtml(translate('chooseOther'))}</small></span></div><label class="checkbox-label"><input id="copy-replace" name="replace" type="checkbox" ${draft.replace ? 'checked' : ''} />${escapeHtml(translate(filesystem ? 'overwriteFile' : 'overwrite'))}</label></div>`;
  }

  /** Update reference/conflict/access feedback without replacing typed fields. @returns {void} */
  app.updateCopyReview = function updateCopyReview() {
    const draft = app.state.dialog;
    if (draft?.type !== 'copy') return;
    document.getElementById('copy-reference').textContent = app.copyReference(draft);
    const focused = document.activeElement?.id;
    document.getElementById('copy-conflict').innerHTML = conflictMarkup(draft);
    const target = app.target(draft.targetId);
    const account = app.account(target, draft.accountId);
    const denied = target.kind === 'registry' && account?.signedIn && !account.permissions.includes('write') && !app.copyConflict(draft).same;
    document.getElementById('access-feedback').innerHTML = denied ? `<div class="notice warning">${app.icon('lock')}<span>${escapeHtml(translate('writeDenied'))}</span></div>` : '';
    const button = document.getElementById('copy-submit');
    button.disabled = denied;
    button.textContent = translate(app.copyConflict(draft).same ? 'openDestination' : target.kind === 'registry' && !account?.signedIn ? 'signIn' : target.kind === 'filesystem' ? 'startExport' : 'startCopy');
    const error = document.getElementById('dialog-error');
    error.textContent = draft.error ? translate(draft.error) : ''; error.hidden = !draft.error;
    if (focused === 'copy-replace') document.getElementById(focused)?.focus();
  };

  /** Render a complete single-confirmation copy review. @param {object} draft Review. @returns {string} Markup. */
  app.dialogRenderers.copy = function copyDialog(draft) {
    const filesystem = app.target(draft.targetId).kind === 'filesystem';
    return `${app.dialogHeader(translate(filesystem ? 'exportTitle' : 'copyTitle'), translate('review'), filesystem ? 'archive' : 'swap')}<form id="copy-form" novalidate><div class="dialog-body">${copySummary(draft)}${copyNameFields(draft)}${copyContentFields(draft)}<div class="reference-preview"><span>${escapeHtml(translate('reference'))}</span><code id="copy-reference"></code></div><div id="copy-conflict"></div><p class="form-error" id="dialog-error" role="alert" hidden></p></div><footer class="dialog-footer"><span class="keep-original">${app.icon('check')}${escapeHtml(translate('originalStays'))}</span><button type="button" class="button" data-action="close-dialog">${escapeHtml(translate('cancel'))}</button><button class="button primary" id="copy-submit" type="submit">${escapeHtml(translate('startCopy'))}</button></footer></form>`;
  };

  /** Render scoped cleanup with separate delete authorization. @param {object} draft Selection snapshot. @returns {string} Markup. */
  app.dialogRenderers.delete = function deleteDialog(draft) {
    const target = app.target(draft.targetId);
    const account = app.account(target, draft.accountId);
    const denied = target.kind === 'registry' && (!account?.signedIn || !account.permissions.includes('delete'));
    return `${app.dialogHeader(translate('deleteTitle', { name: draft.location || app.label(target) }), translate('deleteHint'), 'trash')}<form id="delete-form" novalidate><div class="dialog-body"><div class="delete-item">${app.icon(draft.node.kind)}<strong class="mono">${escapeHtml(draft.node.image ? `${draft.node.image}:` : '')}${escapeHtml(draft.node.name)}</strong></div><div class="detail-line"><span>${escapeHtml(translate('removalScope'))}</span><strong>${escapeHtml(translate(draft.node.kind === 'archive' ? 'archiveScope' : 'versionScope'))}</strong></div>${target.kind === 'registry' ? app.field(translate('account'), `<select name="accountId">${app.accountOptions(target, draft.accountId)}</select>`) : ''}<div class="notice ${denied ? 'warning' : 'neutral'}">${app.icon(denied ? 'lock' : 'info')}<span>${escapeHtml(translate(denied ? 'deleteDenied' : 'otherCopiesStay'))}</span></div><p class="form-error" id="dialog-error" role="alert" ${draft.error ? '' : 'hidden'}>${escapeHtml(draft.error ? translate(draft.error) : '')}</p></div><footer class="dialog-footer"><button type="button" class="button" data-action="close-dialog">${escapeHtml(translate('cancel'))}</button><button class="button danger" type="submit" ${denied ? 'disabled' : ''}>${escapeHtml(translate('deleteConfirm'))}</button></footer></form>`;
  };

  /** Render selected image/archive information. @param {object} draft Selection. @returns {string} Markup. */
  app.dialogRenderers.inspect = function inspectDialog(draft) {
    const node = draft.node;
    const versions = node.kind === 'version' ? [node] : app.collect(node.children, 'version');
    return `${app.dialogHeader(translate('inspectTitle'), node.name, node.kind)}<div class="dialog-body"><div class="detail-line"><span>${escapeHtml(translate('path'))}</span><code>${escapeHtml(draft.path)}</code></div><div class="detail-line"><span>${escapeHtml(translate('size'))}</span><strong>${escapeHtml(app.size(node.size))}</strong></div>${versions.map(version => `<div class="inspected-entry">${app.icon('image')}<div><strong class="mono">${escapeHtml(version.image)}:${escapeHtml(version.name)}</strong><div class="platform-chips">${version.platforms.map(platform => `<span>${escapeHtml(platform)}</span>`).join('')}</div></div></div>`).join('')}${node.kind === 'file' ? `<p class="muted">${escapeHtml(translate('readme'))}</p>` : ''}</div><footer class="dialog-footer"><button class="button" data-action="close-dialog">${escapeHtml(translate('close'))}</button></footer>`;
  };

  /** Render task history from captured records. @returns {string} Markup. */
  app.dialogRenderers.history = function historyDialog() {
    return `${app.dialogHeader(translate('transfers'), '', 'clock')}<div class="dialog-body history-body">${app.state.tasks.length ? app.state.tasks.map(app.taskRow).join('') : `<div class="empty-state">${app.icon('swap')}<h3>${escapeHtml(translate('noTransfers'))}</h3></div>`}</div><footer class="dialog-footer"><button class="button" data-action="history">${app.icon('refresh')}${escapeHtml(translate('refresh'))}</button><button class="button" data-action="close-dialog">${escapeHtml(translate('close'))}</button></footer>`;
  };

  /** Render a captured task, including failure information. @param {object} draft Task reference. @returns {string} Markup. */
  app.dialogRenderers.task = function taskDialog(draft) {
    const task = app.state.tasks.find(item => item.id === draft.id);
    if (!task) return app.dialogHeader(translate('taskDetails'), translate('taskMissing'));
    const details = [['status', translate(task.state)], ['source', task.sourceLabel],
      ['sourceAccount', task.sourceAccountLabel], ['targetLocation', task.targetLabel],
      ['targetAccount', task.accountLabel]];
    return `${app.dialogHeader(translate('taskDetails'), task.name, 'swap')}<div class="dialog-body">${details.map(([key, value]) => `<div class="detail-line"><span>${escapeHtml(translate(key))}</span><strong>${escapeHtml(value || '—')}</strong></div>`).join('')}<div class="reference-preview"><span>${escapeHtml(translate('savedDestination'))}</span><code>${escapeHtml(task.reference)}</code></div><div class="platform-chips">${app.copyPlatforms(task.draft).map(platform => `<span>${escapeHtml(platform)}</span>`).join('')}</div>${task.error ? `<div class="notice warning">${app.icon('info')}${escapeHtml(translate(task.error))}</div>` : ''}</div><footer class="dialog-footer"><button class="button" data-action="close-dialog">${escapeHtml(translate('close'))}</button>${task.state === 'completed' ? `<button class="button primary" data-action="open-task" data-id="${escapeHtml(task.id)}">${escapeHtml(translate('openDestination'))}</button>` : ''}</footer>`;
  };

})();
