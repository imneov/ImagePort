/* Unified target configuration and explicit account selection/sign-in flows. */
(() => {
  'use strict';
  const app = window.ImagePort;
  const escapeHtml = app.escape;
  const translate = app.t;

  /** Build editable target metadata without account secrets. @param {object|undefined} target Existing target. @returns {object} Form state. */
  function targetForm(target) {
    return { label: target ? app.label(target) : '', kind: target?.kind || 'registry',
      address: target?.address || '', provider: target?.provider || 'Harbor' };
  }

  /** Open the one shared target manager. @param {string|null} id Target. @param {string|null} side Origin pane. @param {boolean} create New mode. @returns {void} */
  app.openManager = function openManager(id = null, side = null, create = false) {
    const target = create ? null : app.target(id) || app.paneTarget(side || 'right') || app.state.targets[0];
    app.showDialog({ type: 'manager', targetId: target?.id || null, originSide: side,
      form: { ...targetForm(target), kind: create ? '' : target?.kind || 'registry' }, error: '' });
  };

  /** Render saved targets without mixing them with credential rows. @param {object} draft Manager state. @returns {string} Markup. */
  function managerList(draft) {
    return `<aside class="manager-list"><div class="manager-list-title">${escapeHtml(translate('allTargets'))}<span>${app.state.targets.length}</span></div><div class="manager-saved-targets">${app.state.targets.map(target => `<button type="button" class="manager-target ${target.id === draft.targetId ? 'selected' : ''}" data-action="edit-target" data-id="${escapeHtml(target.id)}"><span class="option-icon ${target.kind}">${app.icon(target.kind)}</span><span><strong>${escapeHtml(app.label(target))}</strong><small class="mono">${escapeHtml(target.address)}</small></span>${app.icon('chevron')}</button>`).join('')}</div><button type="button" class="new-target-row" data-action="manager-new">${app.icon('plus')}${escapeHtml(translate('newTarget'))}</button></aside>`;
  }

  /** Render separate access identities for a registry. @param {object|undefined} target Registry. @returns {string} Markup. */
  function registryAccounts(target) {
    if (!target) return '';
    return `<section class="manager-accounts"><div class="section-caption"><h3>${escapeHtml(translate('accounts'))}</h3><button type="button" class="text-button" data-action="add-account" data-id="${escapeHtml(target.id)}">${app.icon('plus')}${escapeHtml(translate('addAccount'))}</button></div>${target.accounts.map(account => `<div class="saved-account">${app.icon('lock')}<span><strong>${escapeHtml(app.label(account))}</strong><small>${escapeHtml(account.principal)}</small></span><span class="access-pill">${escapeHtml(translate(account.permissions.includes('delete') ? 'canMaintain' : account.permissions.includes('write') ? 'canPublish' : 'readOnly'))}</span></div>`).join('')}</section>`;
  }

  /** Render target form fields. @param {object} draft Manager state. @returns {string} Markup. */
  function managerFields(draft) {
    const target = app.target(draft.targetId);
    const kind = draft.form.kind;
    const labelKey = { registry: 'address', engine: 'engineContext', filesystem: 'folderPath' }[kind];
    const placeholder = { registry: 'https://registry.example.com', engine: 'desktop-linux', filesystem: '/Users/you/Images' }[kind];
    return `<div class="section-caption"><h3>${escapeHtml(translate(target ? 'edit' : 'newTarget'))}</h3><span class="subtle-pill">${escapeHtml(translate(`${kind}One`))}</span></div>${app.field(translate('targetName'), `<input id="target-label" name="label" value="${escapeHtml(draft.form.label)}" maxlength="80" autocomplete="off" required />`)}${app.field(translate('targetType'), `<select name="kind" id="target-kind" ${target ? 'disabled' : ''}>${['registry', 'engine', 'filesystem'].map(value => `<option value="${value}" ${kind === value ? 'selected' : ''}>${escapeHtml(translate(`${value}One`))}</option>`).join('')}</select>`)}${app.field(translate(labelKey), `<input name="address" id="target-address" class="mono" value="${escapeHtml(draft.form.address)}" placeholder="${escapeHtml(placeholder)}" maxlength="500" autocomplete="off" required />`)}${kind === 'registry' ? app.field(translate('provider'), `<select name="provider"><option ${draft.form.provider === 'Harbor' ? 'selected' : ''}>Harbor</option><option ${draft.form.provider === 'OCI' ? 'selected' : ''}>OCI</option></select>`) : ''}${kind === 'registry' ? registryAccounts(target) : ''}<p class="form-error" id="manager-error" role="alert" ${draft.error ? '' : 'hidden'}>${escapeHtml(draft.error ? translate(draft.error) : '')}</p>`;
  }

  /** Show supported target types before opening a new connection form. @returns {string} Markup. */
  function targetKinds() {
    return `<div class="target-kinds"><h3>${escapeHtml(translate('newTarget'))}</h3><div>${['registry', 'engine', 'filesystem'].map(kind => `<button type="button" data-action="manager-kind" data-kind="${kind}">${app.icon(kind)}<strong>${escapeHtml(translate(`${kind}One`))}</strong></button>`).join('')}</div></div>`;
  }

  /** Render the target manager. @param {object} draft Manager. @returns {string} Markup. */
  app.dialogRenderers.manager = function managerDialog(draft) {
    return `${app.dialogHeader(translate('manage'), '', 'settings')}<div class="manager-layout">${managerList(draft)}${draft.form.kind ? `<form id="target-form" class="manager-form" novalidate><div class="manager-fields">${managerFields(draft)}</div><footer class="manager-form-footer">${draft.targetId ? `<button class="text-button danger-text" type="button" data-action="remove-target" data-id="${escapeHtml(draft.targetId)}">${app.icon('trash')}${escapeHtml(translate('remove'))}</button>` : '<span></span>'}<button class="button primary" type="submit">${escapeHtml(translate('save'))}</button></footer></form>` : targetKinds()}</div><footer class="dialog-footer"><button class="button" data-action="close-dialog">${escapeHtml(translate('close'))}</button>${draft.targetId ? `<button class="button" data-action="open-managed-target" data-id="${escapeHtml(draft.targetId)}">${escapeHtml(translate(draft.originSide === 'left' ? 'openHere' : 'openRight'))}${app.icon('arrow')}</button>` : ''}</footer>`;
  };

  /** Capture manager fields before an update. @returns {void} */
  app.syncTargetForm = function syncTargetForm() {
    const form = document.getElementById('target-form');
    const draft = app.state.dialog;
    if (!form || draft?.type !== 'manager') return;
    draft.form = { label: form.elements.label.value, kind: form.elements.kind.value,
      address: form.elements.address.value, provider: form.elements.provider?.value || 'OCI' };
  };

  /** Save a validated target and update both menus. @returns {void} */
  app.submitTarget = function submitTarget() {
    app.syncTargetForm();
    const draft = app.state.dialog;
    const result = app.validateTarget(draft.form, draft.targetId);
    if (result.error) {
      draft.error = result.error;
      const error = document.getElementById('manager-error');
      error.hidden = false; error.textContent = translate(result.error); return;
    }
    const target = app.saveTarget(result.value, draft.targetId);
    draft.targetId = target.id; draft.form = targetForm(target); draft.error = '';
    app.render(); app.renderDialog(); app.toast(translate('targetSaved'));
  };

  /** Open access selection for one pane or a manager record. @param {string} targetId Registry. @param {string|null} side Pane. @param {object|null} returnTo Previous modal. @returns {void} */
  app.openAccounts = function openAccounts(targetId, side = null, returnTo = null) {
    const target = app.target(targetId);
    app.showDialog({ type: 'accounts', targetId, side, returnTo,
      accountId: (side ? app.paneAccount(side)?.id : target.accountId) || target.accounts[0]?.id });
  };

  /** Render account choice independently from registry identity. @param {object} draft Access state. @returns {string} Markup. */
  app.dialogRenderers.accounts = function accountsDialog(draft) {
    const target = app.target(draft.targetId);
    return `${app.dialogHeader(translate('selectAccount'), app.label(target), 'lock')}<form id="account-form"><div class="dialog-body"><div class="account-options">${target.accounts.map(account => `<label class="account-option"><input type="radio" name="accountId" value="${escapeHtml(account.id)}" ${account.id === draft.accountId ? 'checked' : ''} /><span class="account-avatar">${app.icon('lock')}</span><span><strong>${escapeHtml(app.label(account))}</strong><small class="mono">${escapeHtml(account.principal)}</small><small>${escapeHtml(translate(account.permissions.includes('delete') ? 'canMaintain' : account.permissions.includes('write') ? 'canPublish' : 'readOnly'))}</small></span><span class="access-pill">${escapeHtml(translate(account.signedIn ? 'connected' : 'signInRequired'))}</span></label>`).join('')}</div></div><footer class="dialog-footer"><button type="button" class="text-button" data-action="add-account" data-id="${escapeHtml(target.id)}">${app.icon('plus')}${escapeHtml(translate('addAccount'))}</button><button type="button" class="button" data-action="close-dialog">${escapeHtml(translate('cancel'))}</button><button type="submit" class="button primary">${escapeHtml(translate('useAccount'))}</button></footer></form>`;
  };

  /** Apply an explicitly selected access context only to its intended pane. @returns {void} */
  app.submitAccount = function submitAccount() {
    const draft = app.state.dialog;
    const selected = document.querySelector('#account-form input:checked')?.value;
    const target = app.target(draft.targetId);
    if (!app.account(target, selected)) return;
    if (draft.side) { app.state.panes[draft.side].accountId = selected; app.state.panes[draft.side].selected = null; }
    else target.accountId = selected;
    if (!app.account(target, selected).signedIn) {
      app.openLogin(target.id, selected, draft.side, draft.returnTo); return;
    }
    app.render();
    if (draft.returnTo) app.showDialog(draft.returnTo); else app.closeDialog();
  };

  /** Open a demo sign-in without storing a secret. @param {string} targetId Registry. @param {string} accountId Identity. @param {string|null} side Pane. @param {object|null} returnTo Preserved flow. @returns {void} */
  app.openLogin = function openLogin(targetId, accountId, side = null, returnTo = null) {
    app.showDialog({ type: 'login', targetId, accountId, side, returnTo });
  };

  /** Render sign-in for a saved identity without accepting a secret. @param {object} draft Login. @returns {string} Markup. */
  app.dialogRenderers.login = function loginDialog(draft) {
    const target = app.target(draft.targetId);
    const account = app.account(target, draft.accountId);
    return `${app.dialogHeader(translate('signIn'), app.label(target), 'lock')}<form id="login-form"><div class="dialog-body">${app.field(translate('username'), `<input value="${escapeHtml(account?.principal || '')}" readonly />`)}</div><footer class="dialog-footer"><button class="button" type="button" data-action="login-back">${escapeHtml(translate('back'))}</button><button class="button primary" type="submit">${escapeHtml(translate('signIn'))}</button></footer></form>`;
  };

  /** Complete sample sign-in and resume the initiating action. @returns {void} */
  app.submitLogin = function submitLogin() {
    const draft = app.state.dialog;
    const target = app.target(draft.targetId);
    const account = app.account(target, draft.accountId);
    if (!account) { app.toast(translate('noAccounts')); return; }
    account.signedIn = true;
    if (draft.side) app.state.panes[draft.side].accountId = account.id;
    app.render();
    if (draft.returnTo) app.showDialog(draft.returnTo); else app.closeDialog();
  };

  /** Render metadata-only account creation. @param {object} draft Account form. @returns {string} Markup. */
  app.dialogRenderers.addAccount = function addAccountDialog(draft) {
    return `${app.dialogHeader(translate('addAccount'), app.label(app.target(draft.targetId)), 'lock')}<form id="add-account-form" novalidate><div class="dialog-body">${app.field(translate('accountLabel'), '<input name="label" maxlength="80" required autofocus />')}${app.field(translate('username'), '<input name="principal" maxlength="80" required />')}<p class="form-error" id="account-error" role="alert" hidden></p></div><footer class="dialog-footer"><button type="button" class="button" data-action="account-back">${escapeHtml(translate('back'))}</button><button type="submit" class="button primary">${escapeHtml(translate('addAccount'))}</button></footer></form>`;
  };

  /** Add sample identity metadata without accepting passwords. @returns {void} */
  app.submitNewAccount = function submitNewAccount() {
    const draft = app.state.dialog;
    const form = document.getElementById('add-account-form');
    const label = form.elements.label.value.trim();
    const principal = form.elements.principal.value.trim();
    if (!label || !principal || label.length > 80 || principal.length > 80 || /[\x00-\x1f]/.test(label + principal)) {
      const error = document.getElementById('account-error'); error.hidden = false; error.textContent = translate('invalidTarget'); return;
    }
    const target = app.target(draft.targetId);
    target.accounts.push({ id: app.uid('account'), label, principal, permissions: ['read', 'write'], signedIn: false });
    app.toast(translate('accountAdded'));
    if (draft.returnTo) app.showDialog(draft.returnTo); else app.openAccounts(target.id);
  };

  /** Render removal of configuration, not content. @param {object} draft Removal. @returns {string} Markup. */
  app.dialogRenderers.removeTarget = function removeTargetDialog(draft) {
    return `${app.dialogHeader(translate('removeTargetTitle', { name: app.label(app.target(draft.targetId)) }), translate('removeHint'), 'settings')}<div class="dialog-body"><div class="notice neutral">${app.icon('info')}<span>${escapeHtml(translate('otherCopiesStay'))}</span></div></div><footer class="dialog-footer"><button class="button" data-action="target-remove-back">${escapeHtml(translate('cancel'))}</button><button class="button danger" data-action="confirm-remove-target" data-id="${escapeHtml(draft.targetId)}">${escapeHtml(translate('remove'))}</button></footer>`;
  };

  /** Remove a saved target without cascading into its content. @param {string} id Target. @returns {void} */
  app.removeTarget = function removeTarget(id) {
    const originSide = app.state.dialog.returnTo?.originSide || null;
    app.state.targets = app.state.targets.filter(target => target.id !== id);
    Object.values(app.state.panes).filter(pane => pane.targetId === id).forEach(pane => {
      pane.targetId = null; pane.selected = null; pane.browseId = null; pane.accountId = null;
    });
    app.render(); app.openManager(null, originSide); app.toast(translate('targetRemoved'));
  };
})();
