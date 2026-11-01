/* Accessible, dependency-free workspace rendering. All record values are escaped. */
(() => {
  'use strict';
  const app = window.ImagePort;
  const escapeHtml = app.escape;
  const paths = {
    box: '<path d="m12 3 9 5-9 5-9-5 9-5Z"/><path d="M3 8v9l9 5 9-5V8M12 13v9M7.5 5.5l9 5"/>',
    registry: '<rect x="3" y="3" width="18" height="7" rx="2"/><rect x="3" y="14" width="18" height="7" rx="2"/><path d="M7 6.5h.01M7 17.5h.01M11 6.5h6M11 17.5h6"/>',
    engine: '<rect x="3" y="4" width="18" height="13" rx="2"/><path d="M8 21h8M12 17v4m-4-10 3-3 3 3-3 3-3-3Z"/>',
    filesystem: '<path d="M3 7a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"/>',
    folder: '<path d="M3 7a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"/>',
    project: '<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>',
    image: '<path d="m12 3 9 5-9 5-9-5 9-5Z"/><path d="M3 8v9l9 5 9-5V8M12 13v9"/>',
    version: '<path d="M3 3h8l10 10-8 8L3 11V3Z"/><circle cx="7.5" cy="7.5" r="1"/>',
    archive: '<rect x="5" y="3" width="14" height="19" rx="2"/><path d="M10 3v3h4v3h-4v3h4v3h-4v3h4"/>',
    file: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6Zm0 0v6h6M8 13h8M8 17h5"/>',
    chevron: '<path d="m9 5 7 7-7 7"/>', down: '<path d="m6 9 6 6 6-6"/>',
    search: '<circle cx="10.5" cy="10.5" r="6.5"/><path d="m16 16 5 5"/>',
    plus: '<path d="M12 5v14M5 12h14"/>', close: '<path d="m6 6 12 12M6 18 18 6"/>',
    check: '<path d="m5 12 4 4L19 6"/>', arrow: '<path d="M4 12h16m-6-6 6 6-6 6"/>',
    swap: '<path d="M4 7h16m-4-4 4 4-4 4M20 17H4m4-4-4 4 4 4"/>',
    refresh: '<path d="M20 7v5h-5M4 17v-5h5"/><path d="M6 7a7 7 0 0 1 11-2l3 3M4 16l3 3a7 7 0 0 0 11-2"/>',
    tree: '<path d="M5 3v14a2 2 0 0 0 2 2h4M5 7h6"/><rect x="11" y="4" width="9" height="6" rx="1"/><rect x="11" y="16" width="9" height="6" rx="1"/>',
    list: '<path d="M9 6h12M9 12h12M9 18h12M3 6h.01M3 12h.01M3 18h.01"/>',
    trash: '<path d="M3 6h18M9 6V3h6v3M5 6l1 15h12l1-15M10 10v7M14 10v7"/>',
    settings: '<path d="M4 7h16M4 17h16"/><circle cx="9" cy="7" r="3"/><circle cx="15" cy="17" r="3"/>',
    clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    moon: '<path d="M20.9 13A9 9 0 0 1 11 3.1 9 9 0 1 0 20.9 13Z"/>',
    sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M2 12h2M20 12h2M5 5l1.5 1.5M17.5 17.5 19 19M19 5l-1.5 1.5M6.5 17.5 5 19"/>',
    info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v6M12 7h.01"/>',
    lock: '<rect x="5" y="10" width="14" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3M12 14v3"/>',
    external: '<path d="M14 3h7v7m0-7L11 13M10 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-5"/>',
  };

  /** Return a decorative SVG icon. @param {string} name Icon. @param {string} classes CSS classes. @returns {string} Markup. */
  app.icon = function icon(name, classes = '') {
    return `<svg class="icon ${escapeHtml(classes)}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths[name] || paths.box}</svg>`;
  };

  /** Render an accessible icon button. @param {string} action Action. @param {string} icon Icon. @param {string} label Label. @param {string} attrs Trusted attributes. @returns {string} Button. */
  app.iconButton = function iconButton(action, icon, label, attrs = '') {
    return `<button class="icon-button" type="button" data-action="${escapeHtml(action)}" aria-label="${escapeHtml(label)}" title="${escapeHtml(label)}" ${attrs}>${app.icon(icon)}</button>`;
  };

  /** Get platform description. @param {object} node Image. @returns {string} Text. */
  app.platformLabel = function platformLabel(node) {
    if (!node.platforms) return '—';
    return node.platforms.length > 1 ? app.quantity(node.platforms.length, 'platforms') : node.platforms[0].replace('linux/', '');
  };

  /** Readable row metadata. @param {object} node Node. @returns {string} Text. */
  function nodeDetail(node) {
    if (node.kind === 'version') return app.platformLabel(node);
    if (node.kind === 'image') return app.quantity(node.children.length, 'versions');
    if (node.kind === 'project') return app.quantity(node.children.length, 'items');
    return app.t(node.kind === 'archive' ? 'archive' : node.kind);
  }

  /** Match an item or descendant against search. @param {object} node Node. @param {string} query Search. @returns {boolean} Match. */
  function matches(node, query) {
    return !query || `${node.name} ${node.image || ''}`.toLowerCase().includes(query) ||
      Boolean(node.children?.some(child => matches(child, query)));
  }

  /** Flatten only visible hierarchy rows. @param {object[]} nodes Nodes. @param {object} pane Pane. @param {number} depth Indentation. @returns {object[]} Rows. */
  function walkRows(nodes, pane, depth = 0) {
    return (nodes || []).filter(node => matches(node, pane.query.toLowerCase())).flatMap(node => {
      const open = pane.expanded.has(node.id) || Boolean(pane.query);
      return [{ node, depth, open }, ...(node.children && open ? walkRows(node.children, pane, depth + 1) : [])];
    });
  }

  /** List visible nodes for either browsing mode. @param {string} side Pane. @returns {object[]} Rows. */
  app.rows = function rows(side) {
    const pane = app.state.panes[side];
    const target = app.paneTarget(side);
    if (!target || !app.paneReadable(side)) return [];
    if (pane.view === 'tree') return walkRows(target.tree, pane);
    const parent = app.find(target.tree, pane.browseId)?.node;
    return (parent?.children || target.tree).filter(node => matches(node, pane.query.toLowerCase()))
      .map(node => ({ node, depth: 0, open: false }));
  };

  /** Render a single selectable tree row. @param {object} row Visible row. @param {string} side Pane. @param {boolean} tabstop Keyboard entry row. @returns {string} Markup. */
  function treeRow(row, side, tabstop) {
    const pane = app.state.panes[side];
    const node = row.node;
    const selected = pane.selected === node.id;
    const expandable = Boolean(node.children);
    const expansion = expandable ? `aria-expanded="${row.open}"` : '';
    const arrow = expandable ? `<button type="button" tabindex="-1" class="disclosure ${row.open ? 'expanded' : ''}" data-action="expand" data-side="${side}" data-id="${escapeHtml(node.id)}" aria-label="${escapeHtml(app.t(row.open ? 'collapse' : 'expand'))} ${escapeHtml(node.name)}">${app.icon('chevron')}</button>` : '<span class="disclosure-space"></span>';
    return `<div class="tree-row ${selected ? 'selected' : ''} node-${node.kind}" role="row" aria-level="${row.depth + 1}" aria-selected="${selected}" ${expansion}
      tabindex="${tabstop ? '0' : '-1'}" id="row-${side}-${escapeHtml(node.id)}" data-row="${escapeHtml(node.id)}" data-side="${side}">
      <div class="name-cell" role="gridcell" style="--depth:${row.depth}">${arrow}${app.icon(node.kind)}<span class="node-name ${node.kind === 'version' ? 'mono' : ''}">${escapeHtml(node.name)}</span>${selected && node.kind === 'version' ? '<span class="selected-dot"></span>' : ''}</div>
      <div class="meta-cell" role="gridcell">${escapeHtml(nodeDetail(node))}</div>
      <div class="size-cell mono" role="gridcell">${escapeHtml(node.kind === 'version' || node.kind === 'archive' || node.kind === 'file' ? app.size(node.size) : '—')}</div>
    </div>`;
  }

  /** Render breadcrumbs independently from the target selector. @param {string} side Pane. @returns {string} Markup. */
  function breadcrumbs(side) {
    const pane = app.state.panes[side];
    const target = app.paneTarget(side);
    if (!target) return '';
    const match = app.find(target.tree, pane.browseId);
    const nodes = pane.view === 'list' && match ? [...match.ancestors, match.node] : app.context(side).filter(node => ['project', 'folder'].includes(node.kind));
    return `<button type="button" data-action="root" data-side="${side}" class="crumb-root">${app.icon(target.kind)}<span>${escapeHtml(target.kind === 'registry' ? target.address.replace(/^https?:\/\//, '') : target.kind === 'filesystem' ? target.address : app.label(target))}</span></button>${nodes.map(node => `<span class="crumb-divider">/</span><button type="button" data-action="navigate" data-side="${side}" data-id="${escapeHtml(node.id)}">${escapeHtml(node.name)}</button>`).join('')}`;
  }

  /** Render an empty or locked pane. @param {string} side Pane. @returns {string} Markup. */
  function emptyPane(side) {
    const target = app.paneTarget(side);
    const query = app.state.panes[side].query;
    const locked = target && !app.paneReadable(side);
    const title = locked ? 'signInRequired' : query ? 'noResults' : !target ? 'noTarget' : target.kind === 'filesystem' ? 'emptyFolder' : 'empty';
    return `<div class="empty-state"><span class="empty-icon">${app.icon(locked ? 'lock' : query ? 'search' : target?.kind || 'box')}</span><h3>${escapeHtml(app.t(title))}</h3>
      ${locked ? `<p>${escapeHtml(app.t('signedOutHint'))}</p>` : ''}
      ${locked ? `<button class="button primary small" data-action="login" data-side="${side}">${escapeHtml(app.t('signIn'))}</button>` : query ? `<button class="button small" data-action="clear-search" data-side="${side}">${escapeHtml(app.t('clearSearch'))}</button>` : ''}</div>`;
  }

  /** Render the selected item and location totals. @param {string} side Pane. @returns {string} Markup. */
  function paneFooter(side) {
    const node = app.selection(side)?.node;
    const label = node?.kind === 'version' ? `${node.image}:${node.name}` : node?.name;
    return `<span class="selection-summary">${escapeHtml(label || '—')}</span><span>${app.rows(side).length} ${escapeHtml(app.t('visible'))}${node?.size ? ` · ${escapeHtml(app.size(node.size))}` : ''}</span>`;
  }

  /** Render commands for the active pane, using the opposite pane as destination. @returns {void} */
  app.renderCommands = function renderCommands() {
    const side = app.state.activePane;
    const node = app.selection(side)?.node;
    const other = side === 'left' ? 'right' : 'left';
    const destination = app.paneTarget(other);
    const targetName = app.container(other, 'project')?.name || app.label(destination);
    const supported = destination && !(app.paneTarget(side)?.kind === 'filesystem' && destination.kind === 'filesystem');
    const copyable = node && ['version', 'archive'].includes(node.kind) && app.paneReadable(side) && supported;
    const deletable = node && (node.kind === 'archive' || (node.kind === 'version' && app.paneTarget(side)?.kind !== 'filesystem'));
    const copyLabel = app.t(destination?.kind === 'filesystem' ? 'exportTo' : 'copyTo', { name: targetName });
    const commands = [
      { action: 'inspect', label: app.t('inspect'), key: 'F3', enabled: node, icon: 'info' },
      { action: 'copy', label: copyLabel, key: 'F5', enabled: copyable, icon: 'arrow' },
      { action: 'delete', label: app.t('delete'), key: 'F8', enabled: deletable, icon: 'trash' },
    ];
    document.getElementById('command-bar').innerHTML = commands.map(command => `<button type="button" data-action="${command.action}" data-side="${side}" ${command.enabled ? '' : 'disabled'}>${app.icon(command.icon, command.action === 'copy' && side === 'right' ? 'reverse' : '')}<span>${escapeHtml(command.label)}</span><kbd>${command.key}</kbd></button>`).join('');
  };

  /** Keep command ownership aligned with the focused pane. @param {string} side Pane. @returns {void} */
  app.activatePane = function activatePane(side) {
    if (!['left', 'right'].includes(side) || app.state.activePane === side) return;
    app.state.activePane = side;
    document.querySelectorAll('[data-pane]').forEach(pane => pane.classList.toggle('active-pane', pane.dataset.pane === side));
    app.renderCommands();
  };

  /** Render one full location pane. @param {string} side Pane. @returns {string} Markup. */
  function paneMarkup(side) {
    const target = app.paneTarget(side);
    const pane = app.state.panes[side];
    const account = app.paneAccount(side);
    const rows = app.rows(side);
    return `<section class="location-pane ${app.state.activePane === side ? 'active-pane' : ''}" aria-label="${escapeHtml(app.t(side === 'left' ? 'left' : 'right'))}" data-pane="${side}">
      <div class="pane-heading"><span class="eyebrow">${escapeHtml(app.t(side === 'left' ? 'left' : 'right'))}</span><span class="connection ${app.paneReadable(side) ? '' : 'offline'}"><span></span>${escapeHtml(app.t(!target ? 'noTarget' : !app.paneReadable(side) ? 'signInRequired' : target?.kind === 'filesystem' ? 'available' : 'connected'))}</span></div>
      <div class="location-bar"><button class="target-trigger" id="target-${side}" type="button" data-action="targets" data-side="${side}" aria-haspopup="menu" aria-expanded="${app.state.menuSide === side}" aria-controls="target-menu"><span class="target-symbol ${target?.kind || ''}">${app.icon(target?.kind || 'plus')}</span><span class="target-heading"><strong>${escapeHtml(app.label(target))}</strong></span>${app.icon('down')}</button></div>
      <div class="access-line">${target?.kind === 'registry' ? `${app.icon('lock')}<span>${escapeHtml(app.t('account'))}: <strong>${escapeHtml(account ? app.label(account) : app.t('noAccounts'))}</strong></span><button class="text-button" data-action="accounts" data-side="${side}">${escapeHtml(app.t('change'))}</button>` : `<span class="mono">${escapeHtml(target?.address || '')}</span>`}</div>
      <div class="pane-tools"><label class="search-box">${app.icon('search')}<input id="search-${side}" type="search" data-search="${side}" value="${escapeHtml(pane.query)}" placeholder="${escapeHtml(app.t('search'))}" aria-label="${escapeHtml(app.t('search'))}" autocomplete="off" /></label>
      <div class="segmented">${['tree', 'list'].map(view => `<button class="${pane.view === view ? 'chosen' : ''}" type="button" data-action="view" data-side="${side}" data-view="${view}" aria-label="${escapeHtml(app.t(view))}" title="${escapeHtml(app.t(view))}" aria-pressed="${pane.view === view}">${app.icon(view)}</button>`).join('')}</div>${app.iconButton('refresh', 'refresh', app.t('refresh'), `data-side="${side}"`)}</div>
      <div class="breadcrumbs">${pane.browseId ? app.iconButton('up', 'arrow', app.t('up'), `data-side="${side}"`) : ''}${breadcrumbs(side)}</div>
      <div class="table-header"><span>${escapeHtml(app.t('name'))}</span><span>${escapeHtml(app.t('details'))}</span><span>${escapeHtml(app.t('size'))}</span></div>
      <div class="tree-content" role="treegrid" aria-label="${escapeHtml(app.label(target))}" data-tree="${side}">${rows.length ? rows.map((row, index) => treeRow(row, side, row.node.id === pane.selected || (!rows.some(item => item.node.id === pane.selected) && index === 0))).join('') : emptyPane(side)}</div>
      <footer class="pane-footer">${paneFooter(side)}</footer></section>`;
  }

  /** Save focus across workspace redraws. @returns {object|null} Focus descriptor. */
  function captureFocus() {
    const active = document.activeElement;
    return active?.id ? { id: active.id, start: active.selectionStart, end: active.selectionEnd } : null;
  }

  /** Render the complete workspace while preserving input focus. @returns {void} */
  app.render = function render() {
    const focus = captureFocus();
    document.documentElement.lang = app.state.language === 'zh' ? 'zh-CN' : 'en';
    document.documentElement.dataset.theme = app.state.theme;
    document.querySelector('.skip-link').textContent = app.t('workspace');
    document.getElementById('application').innerHTML = `<header class="app-header"><a class="brand" href="#workspace" aria-label="ImagePort">${app.icon('box')}ImagePort</a>
      <div class="toolbar">${app.iconButton('swap', 'swap', app.t('swap'))}<button class="button" data-action="history">${app.icon('clock')}<span>${escapeHtml(app.t('transfers'))}</span><span id="task-count" class="nav-count">${app.state.tasks.length}</span></button></div>
      <div class="global-controls"><button class="button manage-button" id="manage-targets" data-action="manage" aria-label="${escapeHtml(app.t('manage'))}" title="${escapeHtml(app.t('manage'))}">${app.icon('settings')}<span>${escapeHtml(app.t('manage'))}</span></button><span class="control-divider"></span><button class="language-button" data-action="language" aria-label="${escapeHtml(app.t('language'))}">${app.state.language === 'en' ? 'EN' : '中'}</button>${app.iconButton('theme', app.state.theme === 'light' ? 'moon' : 'sun', app.t('theme'))}</div></header>
      <main id="workspace" tabindex="-1"><div class="commander-grid">${paneMarkup('left')}${paneMarkup('right')}</div><section id="transfer-queue" class="transfer-section" aria-label="${escapeHtml(app.t('queue'))}"></section></main>
      <footer id="command-bar" class="command-bar" aria-label="${escapeHtml(app.t('actions'))}"></footer>`;
    app.renderCommands(); app.renderQueue();
    if (focus) {
      const restored = document.getElementById(focus.id);
      restored?.focus({ preventScroll: true });
      if (restored?.type === 'search' && focus.start != null) restored.setSelectionRange(focus.start, focus.end);
    }
  };

  /** Render an individual transfer row. @param {object} task Task. @returns {string} Markup. */
  app.taskRow = function taskRow(task) {
    const active = ['queued', 'copying', 'verifying'].includes(task.state);
    const action = task.state === 'completed' ? app.iconButton('open-task', 'external', app.t('openDestination'), `data-id="${escapeHtml(task.id)}"`) : active ? app.iconButton('cancel-task', 'close', app.t('cancel'), `data-id="${escapeHtml(task.id)}"`) : app.iconButton('retry-task', 'refresh', app.t('retry'), `data-id="${escapeHtml(task.id)}"`);
    return `<div class="transfer-row" data-task="${escapeHtml(task.id)}"><div class="task-image"><span class="task-symbol">${app.icon(task.route === 'export' ? 'archive' : 'image')}</span><span><strong class="mono">${escapeHtml(task.name)}</strong><small>${escapeHtml(app.t(task.route))} · ${escapeHtml(app.size(task.source.size))}</small></span></div><div class="task-route"><span>${escapeHtml(task.sourceLabel)} ${app.icon('arrow')} ${escapeHtml(task.targetLabel)}</span><small class="mono">${escapeHtml(task.reference)}</small></div><div class="task-state"><span class="state-badge ${escapeHtml(task.state)}">${task.state === 'completed' ? app.icon('check') : task.state === 'failed' ? app.icon('info') : ''}${escapeHtml(app.t(task.state))}</span></div><div class="task-progress"><div class="progress-track"><span style="width:${task.progress}%"></span></div><small>${task.state === 'completed' ? '100%' : task.state === 'cancelled' ? '—' : `${task.progress}%`}</small></div><div class="task-actions">${app.iconButton('task-details', 'info', app.t('details'), `data-id="${escapeHtml(task.id)}"`)}${action}</div></div>`;
  };

  /** Update transfer activity without replacing pane controls. @returns {void} */
  app.renderQueue = function renderQueue() {
    const section = document.getElementById('transfer-queue');
    if (!section) return;
    const active = app.state.tasks.filter(task => ['queued', 'copying', 'verifying'].includes(task.state)).length;
    section.hidden = !app.state.tasks.length;
    section.innerHTML = `<div class="queue-heading"><h2>${app.icon('swap')}${escapeHtml(app.t('queue'))}<span class="queue-count">${app.state.tasks.length}</span>${active ? `<span class="active-count">${active} ${escapeHtml(app.t('active'))}</span>` : ''}</h2><div><button class="text-button" data-action="clear-finished">${escapeHtml(app.t('clearFinished'))}</button><button class="text-button" data-action="history">${escapeHtml(app.t('history'))}${app.icon('external')}</button></div></div>
      <div class="queue-body">${app.state.tasks.length ? `<div class="transfer-header"><span>${escapeHtml(app.t('image'))}</span><span>${escapeHtml(app.t('route'))}</span><span>${escapeHtml(app.t('status'))}</span><span>${escapeHtml(app.t('progress'))}</span><span></span></div>${app.state.tasks.map(app.taskRow).join('')}` : `<div class="queue-empty">${escapeHtml(app.t('noTransfers'))}</div>`}</div>`;
    const count = document.getElementById('task-count');
    if (count) count.textContent = app.state.tasks.length;
  };

  /** Open the unified target menu next to its pane selector. @param {string} side Pane. @returns {void} */
  app.renderMenu = function renderMenu(side) {
    const menu = document.getElementById('target-menu');
    const trigger = document.getElementById(`target-${side}`);
    app.state.menuSide = side;
    const contents = ['engine', 'registry', 'filesystem'].map(kind => `<div class="menu-group-label">${escapeHtml(app.t(kind))}</div>${app.state.targets.filter(target => target.kind === kind).map(target => `<button type="button" role="menuitem" class="target-option ${app.state.panes[side].targetId === target.id ? 'current' : ''}" data-action="choose-target" data-side="${side}" data-id="${escapeHtml(target.id)}"><span class="option-icon ${kind}">${app.icon(kind)}</span><span class="option-name"><strong>${escapeHtml(app.label(target))}</strong><small class="mono">${escapeHtml(target.address)}</small></span>${app.state.panes[side].targetId === target.id ? app.icon('check') : ''}</button>`).join('')}`).join('');
    menu.innerHTML = `${contents}<div class="menu-footer"><button type="button" role="menuitem" data-action="new-target" data-side="${side}">${app.icon('plus')}${escapeHtml(app.t('add'))}</button><button type="button" role="menuitem" data-action="manage" data-side="${side}">${app.icon('settings')}${escapeHtml(app.t('manage'))}</button></div>`;
    menu.setAttribute('role', 'menu'); menu.setAttribute('aria-label', app.t('target'));
    menu.hidden = false;
    const rect = trigger.getBoundingClientRect();
    menu.style.left = `${Math.max(12, Math.min(rect.left, innerWidth - Math.min(388, innerWidth - 24) - 12))}px`;
    const below = innerHeight - rect.bottom - 18;
    const above = rect.top - 18;
    const openBelow = below >= 260 || below >= above;
    menu.style.maxHeight = `${Math.min(540, Math.max(100, openBelow ? below : above))}px`;
    menu.style.top = `${openBelow ? rect.bottom + 7 : Math.max(10, rect.top - menu.offsetHeight - 7)}px`;
    trigger.setAttribute('aria-expanded', 'true');
    menu.querySelector('.current')?.focus();
  };

  /** Close the target menu with optional focus restoration. @param {boolean} restore Restore trigger focus. @returns {void} */
  app.closeMenu = function closeMenu(restore = false) {
    const side = app.state.menuSide;
    document.getElementById('target-menu').hidden = true;
    app.state.menuSide = null;
    document.getElementById(`target-${side}`)?.setAttribute('aria-expanded', 'false');
    if (restore) document.getElementById(`target-${side}`)?.focus();
  };
})();
