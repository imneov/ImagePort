/* In-memory copy, conflict and cleanup operations for the interaction prototype. */
(() => {
  'use strict';
  const app = window.ImagePort;

  /** Resolve the image entry under review. @param {object} draft Review. @returns {object|undefined} Image. */
  app.copySource = function copySource(draft) { return draft.entries.find(entry => entry.id === draft.entryId); };

  /** Resolve the actual destination container. @param {object} draft Review. @returns {object|null} Container. */
  app.copyContainer = function copyContainer(draft) {
    const target = app.target(draft.targetId);
    if (!target) return null;
    if (target.kind === 'registry') return app.find(target.tree, draft.projectId)?.node || null;
    if (target.kind === 'filesystem' && draft.folderId) return app.find(target.tree, draft.folderId)?.node || null;
    return { children: target.tree };
  };

  /** Build a copy review from the source pane. @param {string} side Source side. @returns {object|null} Review. */
  app.prepareCopy = function prepareCopy(side) {
    const sourceTarget = app.paneTarget(side);
    const selection = app.selection(side);
    if (!selection || !['version', 'archive'].includes(selection.node.kind)) {
      app.toast(app.t(selection?.node.kind === 'file' ? 'imageFileOnly' : 'sourceNeeded')); return null;
    }
    const targetSide = side === 'left' ? 'right' : 'left';
    const target = app.paneTarget(targetSide);
    if (!target || (sourceTarget.kind === 'filesystem' && target.kind === 'filesystem')) {
      app.toast(app.t('unsupported')); return null;
    }
    const entries = selection.node.kind === 'version' ? [selection.node] : app.collect(selection.node.children, 'version');
    if (!entries.length) { app.toast(app.t('sourceNeeded')); return null; }
    const source = entries[0];
    return {
      type: 'copy', side, targetSide, sourceTarget: app.clone(sourceTarget),
      entries: app.clone(entries), entryId: source.id, targetId: target.id, targetAddress: target.address,
      projectId: app.container(targetSide, 'project')?.id || '',
      folderId: app.container(targetSide, 'folder')?.id || '', accountId: app.paneAccount(targetSide)?.id || '',
      sourceAccountId: app.paneAccount(side)?.id || '', name: source.image,
      tag: source.name, filename: `${source.image.replaceAll('/', '-')}-${source.name}.tar`,
      format: 'oci', platform: 'all', replace: false, error: '',
    };
  };

  /** Get selected source platforms. @param {object} draft Review. @returns {string[]} Platforms. */
  app.copyPlatforms = function copyPlatforms(draft) {
    return draft.platform === 'all' ? [...app.copySource(draft).platforms] : [draft.platform];
  };

  /** Describe the captured source with its project/archive ancestry. @param {object} draft Review. @returns {string} Reference. */
  app.sourceReference = function sourceReference(draft) {
    const source = app.copySource(draft);
    const target = draft.sourceTarget;
    const ancestors = app.find(target.tree, source.id)?.ancestors || [];
    if (target.kind === 'engine') return `${source.image}:${source.name}`;
    if (target.kind === 'registry') return `${target.address.replace(/^https?:\/\//, '')}/${ancestors.find(node => node.kind === 'project')?.name || '…'}/${source.image}:${source.name}`;
    return `${target.address}/${ancestors.filter(node => ['folder', 'archive'].includes(node.kind)).map(node => node.name).join('/')} / ${source.image}:${source.name}`;
  };

  /** Find an existing destination entry and compare its content. @param {object} draft Review. @returns {object} Conflict. */
  app.copyConflict = function copyConflict(draft) {
    const target = app.target(draft.targetId);
    const container = app.copyContainer(draft);
    if (!container) return { existing: null, same: false };
    const source = app.copySource(draft);
    if (target.kind === 'filesystem') return {
      existing: container.children.find(node => node.name === draft.filename), same: false,
    };
    const image = container.children.find(node => node.kind === 'image' && node.name === draft.name);
    const existing = image?.children.find(node => node.name === draft.tag);
    const platforms = app.copyPlatforms(draft);
    return { existing, same: Boolean(existing && existing.content === source.content &&
      platforms.every(platform => existing.platforms.includes(platform))) };
  };

  /** Preview a destination without exposing secrets. @param {object} draft Review. @returns {string} Reference/path. */
  app.copyReference = function copyReference(draft) {
    const target = app.target(draft.targetId);
    if (!target) return '';
    if (target.kind === 'filesystem') {
      const folder = app.find(target.tree, draft.folderId);
      const parts = folder ? [...folder.ancestors, folder.node].filter(node => node.kind === 'folder').map(node => node.name) : [];
      return [target.address.replace(/[\\/]$/, ''), ...parts, draft.filename].join('/');
    }
    const project = app.find(target.tree, draft.projectId)?.node.name;
    const prefix = target.kind === 'registry' ? `${target.address.replace(/^https?:\/\//, '')}/${project || '…'}/` : '';
    return `${prefix}${draft.name}:${draft.tag}`;
  };

  /** Validate copy input and authorization fixtures. @param {object} draft Review. @returns {string} Error key or empty. */
  app.validateCopy = function validateCopy(draft) {
    const target = app.target(draft.targetId);
    if (!target || target.address !== draft.targetAddress) return 'configChanged';
    if (target.kind === 'registry') {
      if (!app.copyContainer(draft)) return 'selectProjectError';
      const account = app.account(target, draft.accountId);
      if (!account?.signedIn) return 'signInRequired';
      if (!account.permissions.includes('write') && !app.copyConflict(draft).same) return 'writeDenied';
    }
    if (target.kind === 'filesystem') {
      if (!/^[^/\\\x00-\x1f]+\.tar$/i.test(draft.filename) || draft.filename.length > 200) return 'invalidFile';
    } else {
      if (!/^[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*(?:\/[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*)*$/.test(draft.name)) return 'invalidName';
      if (!/^[\w][\w.-]{0,127}$/.test(draft.tag)) return 'invalidTag';
    }
    if (!app.copySource(draft)?.platforms.includes(draft.platform) && draft.platform !== 'all') return 'unsupported';
    const conflict = app.copyConflict(draft);
    if (conflict.existing && !conflict.same && !draft.replace) return 'conflictConsent';
    return '';
  };

  /** Determine transport wording from target kinds. @param {object} draft Review. @returns {string} Message key. */
  app.copyRoute = function copyRoute(draft) {
    const destination = app.target(draft.targetId);
    if (destination.kind === 'filesystem') return 'export';
    if (draft.sourceTarget.kind === 'filesystem') return destination.kind === 'engine' ? 'import' : 'upload';
    if (draft.sourceTarget.kind === 'registry') return destination.kind === 'engine' ? 'pull' : 'registryCopy';
    return destination.kind === 'registry' ? 'upload' : 'copy';
  };

  /** Queue a validated immutable simulation. @param {object} draft Review. @returns {object} Task. */
  app.queueCopy = function queueCopy(draft) {
    const source = app.copySource(draft);
    const target = app.target(draft.targetId);
    const sourceProject = app.find(draft.sourceTarget.tree, source.id)?.ancestors.find(node => node.kind === 'project')?.name;
    const task = {
      id: app.uid('task'), draft: app.clone(draft), source: app.clone(source),
      targetSnapshot: { id: target.id, kind: target.kind, address: target.address, label: app.label(target) },
      sourceLabel: `${app.label(draft.sourceTarget)}${sourceProject ? ` / ${sourceProject}` : ''}`,
      targetLabel: `${app.label(target)}${target.kind === 'registry' ? ` / ${app.find(target.tree, draft.projectId)?.node.name || ''}` : ''}`,
      sourceReference: app.sourceReference(draft),
      sourceAccountLabel: draft.sourceTarget.kind === 'registry' ? app.label(app.account(draft.sourceTarget, draft.sourceAccountId)) : '—',
      accountLabel: target.kind === 'registry' ? app.label(app.account(target, draft.accountId)) : '—',
      reference: app.copyReference(draft), name: `${draft.name}:${draft.tag}`, route: app.copyRoute(draft),
      state: 'queued', progress: 0, started: Date.now(), error: '', resultId: null,
    };
    app.state.tasks.unshift(task);
    return task;
  };

  /** Insert or replace a single named image version. @param {object[]} children Destination list. @param {object} draft Review. @param {object} source Source. @returns {string} Version ID. */
  function insertVersion(children, draft, source) {
    let image = children.find(node => node.kind === 'image' && node.name === draft.name);
    if (!image) { image = app.group(app.uid('image'), 'image', draft.name); children.push(image); }
    const existing = image.children.find(node => node.name === draft.tag);
    const node = { ...app.clone(source), id: existing?.id || app.uid('version'), image: draft.name,
      name: draft.tag, platforms: app.copyPlatforms(draft) };
    if (existing) image.children[image.children.indexOf(existing)] = node;
    else image.children.unshift(node);
    return node.id;
  }

  /** Materialize a successful copy only in the sample tree. @param {object} task Task. @returns {string} Error key or empty. */
  function completeCopy(task) {
    const target = app.target(task.draft.targetId);
    if (!target || target.address !== task.targetSnapshot.address) return 'configChanged';
    const container = app.copyContainer(task.draft);
    if (!container) return 'missingProject';
    if (target.kind === 'registry' && !app.account(target, task.draft.accountId)?.signedIn) return 'signInRequired';
    const conflict = app.copyConflict(task.draft);
    if (conflict.existing && !conflict.same && !task.draft.replace) return 'conflictConsent';
    if (target.kind !== 'filesystem') task.resultId = insertVersion(container.children, task.draft, task.source);
    else task.resultId = exportArchive(container.children, task);
    return '';
  }

  /** Publish one sample archive to its selected folder. @param {object[]} children Folder. @param {object} task Task. @returns {string} Archive ID. */
  function exportArchive(children, task) {
    const existing = children.find(node => node.name === task.draft.filename);
    const contents = [];
    insertVersion(contents, task.draft, task.source);
    const archive = app.group(existing?.id || app.uid('archive'), 'archive', task.draft.filename, contents);
    archive.size = task.source.size;
    archive.format = task.draft.format;
    if (existing) children[children.indexOf(existing)] = archive;
    else children.unshift(archive);
    return archive.id;
  }

  /** Advance simulated transfers using elapsed time. @returns {void} */
  app.tick = function tick() {
    let changed = false;
    app.state.tasks.filter(task => ['queued', 'copying', 'verifying'].includes(task.state)).forEach(task => {
      const elapsed = Date.now() - task.started;
      const duration = app.state.demo.duration;
      task.state = elapsed < 350 ? 'queued' : elapsed < duration ? 'copying' : 'verifying';
      task.progress = Math.min(96, Math.round(elapsed / duration * 96));
      if (elapsed >= duration + app.state.demo.verifyDuration) {
        task.error = completeCopy(task);
        task.state = task.error ? 'failed' : 'completed';
        task.progress = task.error ? task.progress : 100;
        if (!task.error) app.toast(app.t('transferComplete', { name: task.name }));
        app.render();
      }
      changed = true;
    });
    if (changed) app.renderQueue();
  };

  /** Reveal a task's captured result without using current pane defaults. @param {object} task Task. @returns {void} */
  app.openDestination = function openDestination(task) {
    const target = app.target(task.draft.targetId);
    if (!target || target.address !== task.targetSnapshot.address) { app.toast(app.t('taskMissing')); return; }
    const side = task.draft.targetSide;
    app.chooseTarget(side, target.id);
    const pane = app.state.panes[side];
    const result = app.find(target.tree, task.resultId);
    pane.selected = result?.node.id || task.draft.projectId || null;
    result?.ancestors.forEach(node => pane.expanded.add(node.id));
    app.state.activePane = side;
    app.closeDialog(); app.render();
  };

  /** Remove a selected sample node without touching another location. @param {object} target Target. @param {string} nodeId Selection. @returns {boolean} Removed. */
  app.removeNode = function removeNode(target, nodeId) {
    const found = app.find(target.tree, nodeId);
    if (!found) return false;
    const siblings = found.ancestors.at(-1)?.children || target.tree;
    siblings.splice(siblings.indexOf(found.node), 1);
    Object.values(app.state.panes).filter(pane => pane.targetId === target.id && pane.selected === nodeId)
      .forEach(pane => { pane.selected = found.ancestors.at(-1)?.id || null; });
    return true;
  };

  /** Validate and normalize target configuration. @param {object} input Form. @param {string|null} id Existing target. @returns {object} Result/error. */
  app.validateTarget = function validateTarget(input, id) {
    const value = { label: input.label.trim(), kind: input.kind, address: input.address.trim(), provider: input.provider || 'OCI' };
    if (!value.label || value.label.length > 80 || !value.address || value.address.length > 500 ||
      !['registry', 'engine', 'filesystem'].includes(value.kind) || /[\x00-\x1f]/.test(value.address)) return { error: 'invalidTarget' };
    if (value.kind === 'registry') {
      try {
        const endpoint = new URL(value.address);
        if (!['http:', 'https:'].includes(endpoint.protocol) || endpoint.username || endpoint.password ||
          endpoint.pathname !== '/' || endpoint.search || endpoint.hash) return { error: 'invalidAddress' };
        value.address = endpoint.origin;
      } catch (error) { return { error: 'invalidAddress' }; }
    }
    if (value.kind === 'filesystem' && !/^(?:\/|[A-Za-z]:[\\/]|\\\\)/.test(value.address)) return { error: 'invalidPath' };
    if (app.state.targets.some(target => target.id !== id && target.kind === value.kind && target.address === value.address)) return { error: 'duplicateTarget' };
    return { value };
  };

  /** Apply a valid configuration while keeping identities separate. @param {object} value Validated form. @param {string|null} id Existing ID. @returns {object} Target. */
  app.saveTarget = function saveTarget(value, id) {
    let target = app.target(id);
    if (target) {
      const moved = target.address !== value.address;
      Object.assign(target, value); delete target.labelKey;
      if (moved) resetTargetLocation(target);
    }
    else {
      target = { ...value, id: app.uid('target'), tree: [] };
      if (value.kind === 'registry') {
        target.accounts = app.demoAccounts(false); target.accountId = target.accounts[0].id;
        target.tree = [app.group(app.uid('project'), 'project', 'library')];
      }
      app.state.targets.push(target);
    }
    return target;
  };

  /** Reset access/content when a configuration changes location. @param {object} target Updated target. @returns {void} */
  function resetTargetLocation(target) {
    target.tree = [];
    if (target.kind === 'registry') {
      target.accounts = app.demoAccounts(false); target.accountId = target.accounts[0].id;
      target.tree = [app.group(app.uid('project'), 'project', 'library')];
    }
    Object.values(app.state.panes).filter(pane => pane.targetId === target.id).forEach(pane => {
      pane.selected = null; pane.browseId = null; pane.expanded.clear(); pane.accountId = null;
    });
  }
})();
