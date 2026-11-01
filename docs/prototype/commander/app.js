/* Event routing and browser lifecycle. No external requests or persisted secrets. */
(() => {
  'use strict';
  const app = window.ImagePort;
  let toastTimer;

  /** Show a transient, safely rendered status. @param {string} message Text. @returns {void} */
  app.toast = function toast(message) {
    const toast = document.getElementById('toast');
    toast.textContent = message; toast.hidden = false;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => { toast.hidden = true; }, 3800);
  };

  /** Select a concrete target for one pane only. @param {string} side Pane. @param {string} id Target. @returns {void} */
  app.chooseTarget = function chooseTarget(side, id) {
    if (!app.target(id)) return;
    const pane = app.state.panes[side];
    if (pane.targetId !== id) {
      Object.assign(pane, { targetId: id, selected: null, query: '', browseId: null, accountId: null });
      pane.expanded = new Set();
    }
    app.state.activePane = side; app.closeMenu(); app.render();
    document.getElementById(`target-${side}`)?.focus();
  };

  /** Select a visible row independently of expansion. @param {string} side Pane. @param {string} id Row. @returns {void} */
  function selectRow(side, id) {
    app.state.activePane = side; app.state.panes[side].selected = id; app.render();
    document.getElementById(`row-${side}-${id}`)?.focus({ preventScroll: true });
  }

  /** Expand a group or navigate a list. @param {string} side Pane. @param {string} id Node. @returns {void} */
  function expandNode(side, id) {
    const pane = app.state.panes[side];
    const node = app.find(app.paneTarget(side)?.tree, id)?.node;
    if (!node?.children) return;
    if (pane.view === 'list') { pane.browseId = id; pane.selected = id; }
    else if (pane.expanded.has(id)) pane.expanded.delete(id);
    else pane.expanded.add(id);
    app.render(); document.getElementById(`row-${side}-${id}`)?.focus({ preventScroll: true });
  }

  /** Show read-only details for the current selection. @param {string} side Pane. @returns {void} */
  function inspectSelection(side) {
    const selection = app.selection(side);
    if (!selection) return;
    const target = app.paneTarget(side);
    const path = [target.address, ...selection.ancestors.map(node => node.name), selection.node.name].join('/');
    app.showDialog({ type: 'inspect', node: app.clone(selection.node), path });
  }

  /** Open a cleanup review with exact target and node snapshots. @param {string} side Pane. @returns {void} */
  function deleteSelection(side) {
    const target = app.paneTarget(side);
    const selection = app.selection(side);
    if (!selection || !['version', 'archive'].includes(selection.node.kind) ||
      (target.kind === 'filesystem' && selection.node.kind !== 'archive')) { app.toast(app.t('unsupported')); return; }
    app.showDialog({ type: 'delete', side, targetId: target.id, address: target.address,
      location: [app.label(target), ...selection.ancestors.map(node => node.name)].join(' / '),
      node: app.clone(selection.node), accountId: app.paneAccount(side)?.id || '', error: '' });
  }

  /** Sync copy fields and revoke old overwrite consent after edits. @param {string} changed Edited field. @returns {void} */
  function syncCopyForm(changed = '') {
    const draft = app.state.dialog;
    const form = document.getElementById('copy-form');
    if (draft?.type !== 'copy' || !form) return;
    ['name', 'tag', 'filename', 'format', 'projectId', 'accountId', 'platform', 'entryId'].forEach(key => {
      if (form.elements[key]) draft[key] = form.elements[key].value.trim();
    });
    draft.replace = Boolean(form.elements.replace?.checked);
    if (changed && changed !== 'replace') draft.replace = false;
    draft.error = '';
    if (changed === 'entryId') {
      const source = app.copySource(draft);
      draft.name = source.image; draft.tag = source.name; draft.platform = 'all';
      draft.filename = `${source.image.replaceAll('/', '-')}-${source.name}.tar`;
      app.renderDialog();
    } else app.updateCopyReview();
  }

  /** Submit a reviewed copy or resume it through sign-in. @returns {void} */
  function submitCopy() {
    syncCopyForm();
    const draft = app.state.dialog;
    const target = app.target(draft.targetId);
    if (target?.kind === 'registry' && !app.account(target, draft.accountId)?.signedIn) {
      app.openLogin(target.id, draft.accountId, null, draft); return;
    }
    const error = app.validateCopy(draft);
    if (error) { draft.error = error; app.updateCopyReview(); return; }
    const conflict = app.copyConflict(draft);
    if (conflict.same) {
      app.openDestination({ draft, targetSnapshot: { address: target.address }, resultId: conflict.existing.id }); return;
    }
    app.queueCopy(draft); app.closeDialog(); app.render();
  }

  /** Submit a scoped sample deletion, rechecking current identity/content. @returns {void} */
  function submitDelete() {
    const draft = app.state.dialog;
    const target = app.target(draft.targetId);
    const node = app.find(target?.tree, draft.node.id)?.node;
    const account = app.account(target, draft.accountId);
    const allowed = target?.kind !== 'registry' || (account?.signedIn && account.permissions.includes('delete'));
    if (!target || target.address !== draft.address || !node || node.name !== draft.node.name || node.content !== draft.node.content) {
      draft.error = 'configChanged'; app.renderDialog(); return;
    }
    if (!allowed) { draft.error = 'deleteDenied'; app.renderDialog(); return; }
    app.removeNode(target, node.id); app.closeDialog(); app.render(); app.toast(app.t('deleted'));
  }

  /** Reopen a captured operation for explicit retry review. @param {string} id Task. @returns {void} */
  function retryTask(id) {
    const task = app.state.tasks.find(item => item.id === id);
    const target = app.target(task?.draft.targetId);
    if (!target) { app.toast(app.t('taskMissing')); return; }
    const draft = app.clone(task.draft);
    draft.targetAddress = target.address; draft.error = ''; draft.replace = false;
    if (target.kind === 'registry') {
      if (!app.find(target.tree, draft.projectId)) draft.projectId = '';
      if (!app.account(target, draft.accountId)) draft.accountId = target.accountId;
    }
    app.showDialog(draft);
  }

  /* Each action receives its clicked element; fields are read only by their owning flow. */
  const paneActions = {
    targets: element => {
      const side = element.dataset.side;
      if (app.state.menuSide === side) app.closeMenu(true); else { app.closeMenu(); app.renderMenu(side); }
    },
    'choose-target': element => app.chooseTarget(element.dataset.side, element.dataset.id),
    expand: element => expandNode(element.dataset.side, element.dataset.id),
    copy: element => { const draft = app.prepareCopy(element.dataset.side); if (draft) app.showDialog(draft); },
    inspect: element => inspectSelection(element.dataset.side),
    delete: element => deleteSelection(element.dataset.side),
    refresh: () => app.toast(app.t('refreshed')),
    'clear-search': element => { app.state.panes[element.dataset.side].query = ''; app.render(); },
    root: element => {
      const pane = app.state.panes[element.dataset.side]; pane.browseId = null; pane.selected = null; app.render();
    },
    navigate: element => {
      const pane = app.state.panes[element.dataset.side]; pane.browseId = element.dataset.id;
      pane.selected = element.dataset.id; pane.expanded.add(element.dataset.id); app.render();
    },
    up: element => {
      const side = element.dataset.side;
      const pane = app.state.panes[side];
      const match = app.find(app.paneTarget(side)?.tree, pane.browseId);
      pane.selected = pane.browseId; pane.browseId = match?.ancestors.at(-1)?.id || null; app.render();
    },
    view: element => {
      const pane = app.state.panes[element.dataset.side];
      const selection = app.selection(element.dataset.side);
      if (element.dataset.view === 'list' && pane.view === 'tree' && selection) {
        pane.browseId = selection.node.children ? selection.node.id : selection.ancestors.at(-1)?.id || null;
      }
      pane.view = element.dataset.view;
      if (pane.view === 'tree') app.context(element.dataset.side).forEach(node => pane.expanded.add(node.id));
      app.render();
    },
    swap: () => {
      const previous = app.state.panes.left;
      app.state.panes.left = app.state.panes.right; app.state.panes.right = previous; app.render();
    },
  };

  const targetActions = {
    manage: element => app.openManager(null, element.dataset.side || null),
    'new-target': element => app.openManager(null, element.dataset.side || null, true),
    'manager-new': () => {
      const origin = app.state.dialog.originSide; app.openManager(null, origin, true);
    },
    'manager-kind': element => {
      const kind = element.dataset.kind;
      if (!['registry', 'engine', 'filesystem'].includes(kind)) return;
      app.state.dialog.form.kind = kind; app.renderDialog();
    },
    'edit-target': element => app.openManager(element.dataset.id, app.state.dialog.originSide),
    'open-managed-target': element => {
      const side = app.state.dialog.originSide || 'right'; app.closeDialog(); app.chooseTarget(side, element.dataset.id);
    },
    'remove-target': element => app.showDialog({ type: 'removeTarget', targetId: element.dataset.id, returnTo: app.state.dialog }),
    'target-remove-back': () => app.showDialog(app.state.dialog.returnTo),
    'confirm-remove-target': element => app.removeTarget(element.dataset.id),
    accounts: element => app.openAccounts(app.paneTarget(element.dataset.side).id, element.dataset.side),
    'add-account': element => app.showDialog({ type: 'addAccount', targetId: element.dataset.id, returnTo: app.state.dialog }),
    'account-back': () => app.showDialog(app.state.dialog.returnTo),
    login: element => {
      const side = element.dataset.side; const target = app.paneTarget(side);
      app.openLogin(target.id, app.paneAccount(side)?.id, side);
    },
    'login-back': () => { const previous = app.state.dialog.returnTo; if (previous) app.showDialog(previous); else app.closeDialog(); },
  };

  const globalActions = {
    language: () => {
      app.closeMenu(); app.state.language = app.state.language === 'en' ? 'zh' : 'en';
      app.savePreference('language', app.state.language); app.render(); if (app.state.dialog) app.renderDialog();
    },
    theme: () => {
      app.state.theme = app.state.theme === 'light' ? 'dark' : 'light';
      app.savePreference('theme', app.state.theme); app.render();
    },
    workspace: () => { app.closeDialog(); document.getElementById('workspace').focus(); },
    history: () => app.showDialog({ type: 'history' }),
    'close-dialog': () => app.closeDialog(),
    'clear-finished': () => {
      app.state.tasks = app.state.tasks.filter(task => ['queued', 'copying', 'verifying'].includes(task.state));
      app.renderQueue();
    },
    'task-details': element => app.showDialog({ type: 'task', id: element.dataset.id }),
    'cancel-task': element => {
      const task = app.state.tasks.find(item => item.id === element.dataset.id);
      if (!task || !['queued', 'copying', 'verifying'].includes(task.state)) return;
      task.state = 'cancelled'; app.renderQueue(); app.toast(app.t('transferCancelled'));
      if (app.state.dialog?.type === 'history') app.renderDialog();
    },
    'retry-task': element => retryTask(element.dataset.id),
    'open-task': element => {
      const task = app.state.tasks.find(item => item.id === element.dataset.id);
      if (task) app.openDestination(task);
    },
  };

  /** Route click actions before generic row selection. @param {MouseEvent} event Click. @returns {void} */
  function onClick(event) {
    const pane = event.target.closest('[data-pane]');
    if (pane) app.activatePane(pane.dataset.pane);
    const element = event.target.closest('[data-action]');
    if (app.state.menuSide && !event.target.closest('#target-menu, .target-trigger')) app.closeMenu();
    if (element && !element.disabled) {
      const action = paneActions[element.dataset.action] || targetActions[element.dataset.action] || globalActions[element.dataset.action];
      if (action) { event.preventDefault(); action(element); }
      return;
    }
    const row = event.target.closest('[data-row]');
    if (row) selectRow(row.dataset.side, row.dataset.row);
  }

  /** Apply text changes without erasing copy form inputs. @param {InputEvent} event Input. @returns {void} */
  function onInput(event) {
    const field = event.target;
    if (field.dataset.search) {
      app.state.panes[field.dataset.search].query = field.value; app.render();
    }
    if (field.closest('#copy-form')) syncCopyForm(field.name);
    if (field.closest('#target-form')) app.syncTargetForm();
  }

  /** Handle select/radio changes owned by each modal. @param {Event} event Change. @returns {void} */
  function onChange(event) {
    const field = event.target;
    if (field.closest('#copy-form')) syncCopyForm(field.name);
    if (field.closest('#target-form')) {
      app.syncTargetForm();
      if (field.name === 'kind') { app.state.dialog.form.address = ''; app.renderDialog(); }
    }
    if (field.closest('#delete-form')) { app.state.dialog.accountId = field.value; app.renderDialog(); }
  }

  /** Route forms to their own validated operation. @param {SubmitEvent} event Submission. @returns {void} */
  function onSubmit(event) {
    const handlers = { 'copy-form': submitCopy, 'delete-form': submitDelete,
      'target-form': app.submitTarget, 'account-form': app.submitAccount,
      'login-form': app.submitLogin, 'add-account-form': app.submitNewAccount };
    const handler = handlers[event.target.id];
    if (handler) { event.preventDefault(); handler(); }
  }

  /** Support keyboard navigation in a concrete-target menu. @param {KeyboardEvent} event Key. @returns {boolean} Handled. */
  function menuKey(event) {
    if (!app.state.menuSide) return false;
    if (event.key === 'Tab') { app.closeMenu(true); return false; }
    if (event.key === 'Escape') { app.closeMenu(true); event.preventDefault(); return true; }
    if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return false;
    const options = [...document.querySelectorAll('#target-menu [role="menuitem"]')];
    const current = options.indexOf(document.activeElement);
    const index = event.key === 'Home' ? 0 : event.key === 'End' ? options.length - 1 :
      (current + (event.key === 'ArrowDown' ? 1 : -1) + options.length) % options.length;
    options[index]?.focus(); event.preventDefault(); return true;
  }

  /** Run visible Commander shortcuts outside forms and modal dialogs. @param {KeyboardEvent} event Key. @returns {boolean} Handled. */
  function commandKey(event) {
    if (app.state.dialog || event.target.closest('input, select, textarea') || event.altKey || event.ctrlKey || event.metaKey) return false;
    const action = { F3: 'inspect', F5: 'copy', F8: 'delete' }[event.key];
    if (!action) return false;
    event.preventDefault();
    document.querySelector(`#command-bar [data-action="${action}"]:not(:disabled)`)?.click();
    return true;
  }

  /** Support arrow/enter/space navigation on tree rows. @param {KeyboardEvent} event Key. @returns {void} */
  function onKey(event) {
    if (menuKey(event) || commandKey(event)) return;
    const row = event.target.closest('[data-row]');
    if (!row || event.target.closest('[data-action]')) return;
    const side = row.dataset.side;
    const rows = app.rows(side);
    const index = rows.findIndex(item => item.node.id === row.dataset.row);
    const current = rows[index];
    if (!current) return;
    if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
      const next = event.key === 'Home' ? 0 : event.key === 'End' ? rows.length - 1 :
        Math.max(0, Math.min(rows.length - 1, index + (event.key === 'ArrowDown' ? 1 : -1)));
      event.preventDefault(); selectRow(side, rows[next].node.id);
    } else if (event.key === 'ArrowRight' && current.node.children) {
      event.preventDefault(); if (!current.open) expandNode(side, current.node.id);
      else if (rows[index + 1]) selectRow(side, rows[index + 1].node.id);
    } else if (event.key === 'ArrowLeft') {
      event.preventDefault();
      const parent = app.find(app.paneTarget(side).tree, current.node.id)?.ancestors.at(-1);
      if (current.open) expandNode(side, current.node.id); else if (parent) selectRow(side, parent.id);
    } else if (event.key === 'Enter') {
      event.preventDefault(); if (current.node.children) expandNode(side, current.node.id); else inspectSelection(side);
    } else if (event.key === ' ') { event.preventDefault(); selectRow(side, current.node.id); }
  }

  document.addEventListener('click', onClick);
  document.addEventListener('focusin', event => {
    const pane = event.target.closest('[data-pane]');
    if (pane) app.activatePane(pane.dataset.pane);
  });
  document.addEventListener('input', onInput);
  document.addEventListener('change', onChange);
  document.addEventListener('submit', onSubmit);
  document.addEventListener('keydown', onKey);
  document.addEventListener('dblclick', event => {
    const row = event.target.closest('[data-row]');
    if (!row || event.target.closest('[data-action]')) return;
    const node = app.find(app.paneTarget(row.dataset.side).tree, row.dataset.row)?.node;
    if (node?.children) expandNode(row.dataset.side, node.id); else inspectSelection(row.dataset.side);
  });
  document.getElementById('modal').addEventListener('cancel', event => { event.preventDefault(); app.closeDialog(); });
  document.getElementById('modal').addEventListener('click', event => {
    const rect = event.currentTarget.getBoundingClientRect();
    if (event.target === event.currentTarget && (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom)) app.closeDialog();
  });
  window.addEventListener('resize', () => app.closeMenu());
  app.render();
  setInterval(app.tick, app.state.demo.tick);
})();
