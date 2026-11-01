/* Domain fixtures and browsing state; no real engine, filesystem or registry calls. */
(() => {
  'use strict';
  const app = window.ImagePort;
  let serial = 100;

  /** Return a session-unique fixture ID. @param {string} prefix Kind. @returns {string} ID. */
  app.uid = function uid(prefix) { serial += 1; return `${prefix}-${serial}`; };

  /** Escape untrusted values for HTML text/attributes. @param {*} value Input. @returns {string} HTML. */
  app.escape = function escape(value) {
    return String(value ?? '').replace(/[&<>"']/g, character => ({
      '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
    }[character]));
  };

  /** Deep-copy plain prototype records. @param {*} value JSON-safe value. @returns {*} Independent copy. */
  app.clone = function clone(value) { return JSON.parse(JSON.stringify(value)); };

  /** Read a preference without requiring storage access. @param {string} key Preference. @param {string} fallback Default. @returns {string} Value. */
  function preference(key, fallback) {
    try { return localStorage.getItem(`imageport-commander-${key}`) || fallback; }
    catch (error) { console.warn('Preference storage unavailable:', error.name); return fallback; }
  }

  /** Create a sample image version. @param {string} id ID. @param {string} image Name. @param {string} tag Version. @param {number} size MiB. @param {string[]} platforms Platforms. @returns {object} Node. */
  function version(id, image, tag, size, platforms = ['linux/amd64', 'linux/arm64']) {
    return { id, kind: 'version', name: tag, image, size, platforms, content: `${image}-${tag}` };
  }

  /** Create a browsable group. @param {string} id ID. @param {string} kind Kind. @param {string} name Name. @param {object[]} children Contents. @returns {object} Node. */
  app.group = function group(id, kind, name, children = []) { return { id, kind, name, children }; };

  /** Produce explicit, reusable access fixtures. @param {boolean} signedIn Session state. @returns {object[]} Identities. */
  app.demoAccounts = function demoAccounts(signedIn = true) {
    return [
      { id: app.uid('publisher'), labelKey: 'publisher', principal: 'release', permissions: ['read', 'write'], signedIn },
      { id: app.uid('viewer'), labelKey: 'viewer', principal: 'viewer', permissions: ['read'], signedIn },
      { id: app.uid('maintainer'), labelKey: 'maintainer', principal: 'maintenance', permissions: ['read', 'write', 'delete'], signedIn },
    ];
  };

  const localTree = [
    app.group('local-api', 'image', 'service-api', [
      version('local-api-240', 'service-api', '2.4.0', 184),
      version('local-api-230', 'service-api', '2.3.0', 176),
    ]),
    app.group('local-worker', 'image', 'worker', [version('local-worker-210', 'worker', '2.1.0', 96)]),
    app.group('local-postgres', 'image', 'postgres', [version('local-postgres-164', 'postgres', '16.4', 432, ['linux/arm64'])]),
    app.group('local-nginx', 'image', 'nginx', [version('local-nginx-127', 'nginx', '1.27', 67)]),
    app.group('local-redis', 'image', 'redis', [version('local-redis-74', 'redis', '7.4', 42)]),
  ];
  const companyTree = [
    app.group('project-edge', 'project', 'edge', [
      app.group('edge-api', 'image', 'service-api', [version('edge-api-230', 'service-api', '2.3.0', 176)]),
      app.group('edge-worker', 'image', 'worker', [version('edge-worker-200', 'worker', '2.0.0', 91)]),
      app.group('edge-gateway', 'image', 'gateway', [version('edge-gateway-180', 'gateway', '1.8.0', 58)]),
    ]),
    app.group('project-production', 'project', 'production', [
      app.group('production-api', 'image', 'service-api', [version('production-api-230', 'service-api', '2.3.0', 176)]),
    ]),
    app.group('project-lab', 'project', 'lab', []),
  ];
  const sampleArchive = app.group('archive-release', 'archive', 'release-bundle.tar', [
    app.group('archive-api', 'image', 'service-api', [version('archive-api-240', 'service-api', '2.4.0', 184)]),
    app.group('archive-worker', 'image', 'worker', [version('archive-worker-210', 'worker', '2.1.0', 96)]),
  ]);
  sampleArchive.size = 280;
  const targets = [
    { id: 'docker', kind: 'engine', labelKey: 'docker', address: 'desktop-linux', tree: localTree },
    { id: 'company', kind: 'registry', labelKey: 'company', address: 'https://registry.example.com',
      provider: 'Harbor', tree: companyTree, accounts: app.demoAccounts() },
    { id: 'staging', kind: 'registry', labelKey: 'staging', address: 'https://staging.example.com',
      provider: 'Harbor', tree: [app.group('staging-edge', 'project', 'edge', [])], accounts: app.demoAccounts() },
    { id: 'downloads', kind: 'filesystem', labelKey: 'downloads', address: '/Users/demo/Downloads',
      tree: [app.group('folder-releases', 'folder', 'Releases', [sampleArchive]),
        app.group('archive-nginx', 'archive', 'nginx-1.27.tar', [version('archive-nginx-127', 'nginx', '1.27', 67)]),
        { id: 'ordinary-file', kind: 'file', name: 'release-notes.txt', size: 0.004 }] },
    { id: 'exports', kind: 'filesystem', labelKey: 'exports', address: '/Users/demo/Images', tree: [] },
  ];
  targets.filter(target => target.accounts).forEach(target => { target.accountId = target.accounts[0].id; });

  /** Build independent pane state. @param {string} targetId Target. @param {string[]} expanded Open groups. @param {string|null} selected Selected row. @returns {object} Pane. */
  function pane(targetId, expanded, selected) {
    return { targetId, expanded: new Set(expanded), selected, query: '', view: 'tree', browseId: null, accountId: null };
  }
  app.state = {
    language: preference('language', 'en') === 'zh' ? 'zh' : 'en',
    theme: preference('theme', 'light') === 'dark' ? 'dark' : 'light', targets,
    panes: { left: pane('docker', ['local-api', 'local-worker'], 'local-api-240'),
      right: pane('company', ['project-edge', 'edge-api', 'edge-worker'], 'project-edge') },
    tasks: [], activePane: 'left', menuSide: null, dialog: null,
    demo: { duration: 5200, verifyDuration: 900, tick: 200 },
  };

  /** Resolve translated UI text. @param {string} key Message key. @param {object} values Placeholders. @returns {string} Text. */
  app.t = function translate(key, values = {}) {
    const message = app.messages[app.state.language][key] ?? app.messages.en[key] ?? key;
    return message.replace(/\{(\w+)\}/g, (match, name) => String(values[name] ?? match));
  };

  /** Resolve a target's display name. @param {object} target Target/account. @returns {string} Label. */
  app.label = function label(target) { return target ? (target.labelKey ? app.t(target.labelKey) : target.label) : app.t('noTarget'); };

  /** Find a saved target. @param {string} id ID. @returns {object|undefined} Target. */
  app.target = function target(id) { return app.state.targets.find(item => item.id === id); };

  /** Get a pane's target. @param {string} side Pane side. @returns {object|undefined} Target. */
  app.paneTarget = function paneTarget(side) { return app.target(app.state.panes[side].targetId); };

  /** Locate a node and its ancestry. @param {object[]} nodes Tree. @param {string} id ID. @param {object[]} ancestors Parents. @returns {object|null} Match. */
  app.find = function find(nodes, id, ancestors = []) {
    for (const node of nodes || []) {
      if (node.id === id) return { node, ancestors };
      const result = node.children && app.find(node.children, id, [...ancestors, node]);
      if (result) return result;
    }
    return null;
  };

  /** Collect descendants by kind. @param {object[]} nodes Nodes. @param {string} kind Kind. @returns {object[]} Matches. */
  app.collect = function collect(nodes, kind) {
    return (nodes || []).flatMap(node => [...(node.kind === kind ? [node] : []), ...app.collect(node.children, kind)]);
  };

  /** Resolve selected row with ancestry. @param {string} side Pane. @returns {object|null} Match. */
  app.selection = function selection(side) {
    return app.find(app.paneTarget(side)?.tree, app.state.panes[side].selected);
  };

  /** Get selected/current ancestry including list navigation. @param {string} side Pane. @returns {object[]} Nodes. */
  app.context = function context(side) {
    const paneState = app.state.panes[side];
    const match = app.selection(side) || app.find(app.paneTarget(side)?.tree, paneState.browseId);
    return match ? [...match.ancestors, match.node] : [];
  };

  /** Resolve the current project or folder. @param {string} side Pane. @param {string} kind Kind. @returns {object|undefined} Context. */
  app.container = function container(side, kind) { return app.context(side).filter(node => node.kind === kind).at(-1); };

  /** Resolve active account. @param {object} target Registry. @param {string} id Optional identity. @returns {object|undefined} Account. */
  app.account = function account(target, id) { return target?.accounts?.find(item => item.id === (id || target.accountId)); };

  /** Determine browse access. @param {object} target Target. @returns {boolean} Readable. */
  app.readable = function readable(target, accountId) { return target?.kind !== 'registry' || Boolean(app.account(target, accountId)?.signedIn); };

  /** Resolve one pane's independent identity. @param {string} side Pane. @returns {object|undefined} Account. */
  app.paneAccount = function paneAccount(side) { return app.account(app.paneTarget(side), app.state.panes[side].accountId); };

  /** Check the selected identity for one pane. @param {string} side Pane. @returns {boolean} Readable. */
  app.paneReadable = function paneReadable(side) { return Boolean(app.paneTarget(side)) && app.readable(app.paneTarget(side), app.state.panes[side].accountId); };

  /** Render size using honest fixture units. @param {number} value MiB. @returns {string} Size. */
  app.size = function size(value) { return value == null ? '—' : value < 1 ? `${Math.max(1, Math.round(value * 1024))} KB` : `${value} MB`; };

  /** Format a translated regular count noun. @param {number} count Count. @param {string} key Plural message. @returns {string} Count label. */
  app.quantity = function quantity(count, key) {
    const noun = app.t(key);
    return `${count} ${count === 1 && app.state.language === 'en' ? noun.replace(/s$/, '') : noun}`;
  };

  /** Persist a UI-only preference. @param {string} key Preference. @param {string} value Value. @returns {void} */
  app.savePreference = function savePreference(key, value) {
    try { localStorage.setItem(`imageport-commander-${key}`, value); }
    catch (error) { console.warn('Preference save unavailable:', error.name); app.toast(app.t('storageNotice')); }
  };
})();
