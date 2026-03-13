<script>
  import { onMount } from 'svelte';
  import { ListPlugins, InstallPlugin, UninstallPlugin, EnablePlugin, DisablePlugin, UpdatePlugin, GetPluginConfig, SavePluginConfig, FetchPluginRegistry } from '../../../wailsjs/go/main/App.js';
  import { compareVersions } from '../../utils/version.js';

  let { t } = $props();

  let plugins = $state([]);
  let registryPlugins = $state([]);
  let loading = $state(false);
  let registryLoading = $state(false);
  let error = $state('');
  let installSource = $state('');
  let installing = $state(false);
  let actionLoading = $state('');
  let activeView = $state('installed'); // 'installed' | 'market'

  // Config modal
  let configModal = $state({ show: false, plugin: null, config: '', schema: null, saving: false });

  // Confirm modal
  let confirmModal = $state({ show: false, action: '', pluginName: '', message: '' });

  async function loadPlugins() {
    loading = true;
    error = '';
    try {
      plugins = await ListPlugins() || [];
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      loading = false;
    }
  }

  async function handleInstall() {
    if (!installSource.trim()) return;
    installing = true;
    error = '';
    try {
      await InstallPlugin(installSource.trim());
      installSource = '';
      await loadPlugins();
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      installing = false;
    }
  }

  async function handleToggle(p) {
    actionLoading = p.name;
    try {
      if (p.enabled) {
        await DisablePlugin(p.name);
      } else {
        await EnablePlugin(p.name);
      }
      await loadPlugins();
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      actionLoading = '';
    }
  }

  async function handleUpdate(p) {
    actionLoading = p.name + '-update';
    try {
      await UpdatePlugin(p.name);
      await loadPlugins();
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      actionLoading = '';
    }
  }

  function showUninstallConfirm(p) {
    confirmModal = {
      show: true,
      action: 'uninstall',
      pluginName: p.name,
      message: t.pluginConfirmUninstall?.replace('{name}', p.name) || `Uninstall ${p.name}?`
    };
  }

  async function handleConfirmAction() {
    if (confirmModal.action === 'uninstall') {
      actionLoading = confirmModal.pluginName + '-uninstall';
      try {
        await UninstallPlugin(confirmModal.pluginName);
        await loadPlugins();
      } catch (e) {
        error = e?.message || String(e);
      } finally {
        actionLoading = '';
      }
    }
    confirmModal = { show: false, action: '', pluginName: '', message: '' };
  }

  async function showConfig(p) {
    try {
      const configStr = await GetPluginConfig(p.name);
      configModal = {
        show: true,
        plugin: p,
        config: configStr || '{}',
        schema: p.config_schema || {},
        saving: false
      };
    } catch (e) {
      error = e?.message || String(e);
    }
  }

  async function saveConfig() {
    configModal.saving = true;
    try {
      await SavePluginConfig(configModal.plugin.name, configModal.config);
      configModal = { show: false, plugin: null, config: '', schema: null, saving: false };
      await loadPlugins();
    } catch (e) {
      error = e?.message || String(e);
      configModal.saving = false;
    }
  }

  onMount(() => {
    loadPlugins();
  });

  async function loadRegistry() {
    registryLoading = true;
    try {
      registryPlugins = await FetchPluginRegistry() || [];
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      registryLoading = false;
    }
  }

  async function handleInstallFromRegistry(url) {
    actionLoading = url;
    error = '';
    try {
      await InstallPlugin(url);
      await loadPlugins();
      await loadRegistry();
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      actionLoading = '';
    }
  }

  async function handleUpdateFromRegistry(name) {
    actionLoading = name + '-market-update';
    error = '';
    try {
      await UpdatePlugin(name);
      await loadPlugins();
      await loadRegistry();
    } catch (e) {
      error = e?.message || String(e);
    } finally {
      actionLoading = '';
    }
  }

  function switchView(view) {
    activeView = view;
    if (view === 'market' && registryPlugins.length === 0) {
      loadRegistry();
    }
  }

  const installedMap = $derived(new Map(plugins.map(p => [p.name, p.version])));

  const enabledPlugins = $derived(plugins.filter(p => p.enabled));
  const disabledPlugins = $derived(plugins.filter(p => !p.enabled));
</script>

<div class="p-6 space-y-6">
  <!-- Error banner -->
  {#if error}
    <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-xl flex items-center justify-between">
      <span class="text-sm">{error}</span>
      <button class="text-red-500 hover:text-red-700 cursor-pointer" onclick={() => error = ''}>✕</button>
    </div>
  {/if}

  <!-- View tabs -->
  <div class="flex gap-1 bg-gray-100 rounded-lg p-1 w-fit">
    <button
      onclick={() => switchView('installed')}
      class="px-4 py-1.5 text-sm rounded-md transition-colors cursor-pointer {activeView === 'installed' ? 'bg-white text-gray-900 shadow-sm font-medium' : 'text-gray-500 hover:text-gray-700'}"
    >{t.pluginInstalled || '已安装'}</button>
    <button
      onclick={() => switchView('market')}
      class="px-4 py-1.5 text-sm rounded-md transition-colors cursor-pointer {activeView === 'market' ? 'bg-white text-gray-900 shadow-sm font-medium' : 'text-gray-500 hover:text-gray-700'}"
    >{t.pluginMarket || '插件市场'}</button>
  </div>

  {#if activeView === 'installed'}
  <!-- Stats cards -->
  <div class="grid grid-cols-3 gap-4">
    <div class="bg-white rounded-xl border border-gray-100 p-4 text-center">
      <div class="text-2xl font-bold text-gray-900">{plugins.length}</div>
      <div class="text-xs text-gray-500 mt-1">{t.pluginTotal || '总计'}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4 text-center">
      <div class="text-2xl font-bold text-emerald-600">{enabledPlugins.length}</div>
      <div class="text-xs text-gray-500 mt-1">{t.pluginEnabled || '已启用'}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4 text-center">
      <div class="text-2xl font-bold text-gray-400">{disabledPlugins.length}</div>
      <div class="text-xs text-gray-500 mt-1">{t.pluginDisabled || '已禁用'}</div>
    </div>
  </div>

  <!-- Install form -->
  <div class="bg-white rounded-xl border border-gray-100 p-4">
    <h3 class="text-sm font-medium text-gray-700 mb-3">{t.pluginInstallTitle || '安装插件'}</h3>
    <div class="flex gap-2">
      <input
        type="text"
        bind:value={installSource}
        placeholder={t.pluginInstallPlaceholder || 'Git 仓库地址或本地路径'}
        class="flex-1 px-3 py-2 text-sm border border-gray-200 rounded-lg bg-gray-50 focus:outline-none focus:ring-1 focus:ring-gray-900 focus:border-gray-900"
        onkeydown={(e) => e.key === 'Enter' && handleInstall()}
      />
      <button
        onclick={handleInstall}
        disabled={installing || !installSource.trim()}
        class="px-4 py-2 bg-gray-900 text-white text-sm rounded-lg hover:bg-gray-800 disabled:opacity-50 cursor-pointer transition-colors"
      >
        {installing ? (t.pluginInstalling || '安装中...') : (t.pluginInstall || '安装')}
      </button>
    </div>
  </div>

  <!-- Plugin list -->
  <div class="bg-white rounded-xl border border-gray-100">
    <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
      <h3 class="text-sm font-medium text-gray-700">{t.pluginInstalled || '已安装插件'}</h3>
      <button
        onclick={loadPlugins}
        disabled={loading}
        class="text-xs text-gray-500 hover:text-gray-700 cursor-pointer"
      >
        {loading ? '...' : '↻'}
      </button>
    </div>

    {#if loading && plugins.length === 0}
      <div class="p-8 text-center text-gray-400 text-sm">{t.loading || '加载中...'}</div>
    {:else if plugins.length === 0}
      <div class="p-8 text-center">
        <div class="text-3xl mb-2">🔌</div>
        <div class="text-sm text-gray-500">{t.pluginEmpty || '暂无已安装插件'}</div>
        <div class="text-xs text-gray-400 mt-1">{t.pluginEmptyHint || '通过上方输入框安装第一个插件'}</div>
      </div>
    {:else}
      <div class="divide-y divide-gray-50">
        {#each plugins as p}
          <div class="px-4 py-3 hover:bg-gray-50/50 transition-colors">
            <div class="flex items-start justify-between gap-3">
              <!-- Left: info -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium text-gray-900 truncate">{p.name}</span>
                  <span class="text-xs text-gray-400">v{p.version}</span>
                  {#if p.category}
                    <span class="text-xs px-1.5 py-0.5 bg-gray-100 text-gray-500 rounded">{p.category}</span>
                  {/if}
                </div>
                <div class="text-xs text-gray-500 mt-0.5 truncate">{p.description}</div>
                {#if p.tags?.length}
                  <div class="flex gap-1 mt-1 flex-wrap">
                    {#each p.tags as tag}
                      <span class="text-xs px-1.5 py-0.5 bg-blue-50 text-blue-600 rounded">#{tag}</span>
                    {/each}
                  </div>
                {/if}
              </div>

              <!-- Right: actions -->
              <div class="flex items-center gap-2 shrink-0">
                <!-- Toggle -->
                <button
                  onclick={() => handleToggle(p)}
                  disabled={actionLoading === p.name}
                  class="relative w-9 h-5 rounded-full transition-colors cursor-pointer {p.enabled ? 'bg-emerald-500' : 'bg-gray-300'}"
                  title={p.enabled ? (t.pluginClickDisable || '点击禁用') : (t.pluginClickEnable || '点击启用')}
                >
                  <span class="absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform {p.enabled ? 'translate-x-4' : 'translate-x-0'}"></span>
                </button>

                <!-- Config button -->
                {#if p.config_schema && Object.keys(p.config_schema).length > 0}
                  <button
                    onclick={() => showConfig(p)}
                    class="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg cursor-pointer transition-colors"
                    title={t.pluginConfig || '配置'}
                  >
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
                    </svg>
                  </button>
                {/if}

                <!-- Update button -->
                <button
                  onclick={() => handleUpdate(p)}
                  disabled={actionLoading === p.name + '-update'}
                  class="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg cursor-pointer transition-colors"
                  title={t.pluginUpdate || '更新'}
                >
                  {#if actionLoading === p.name + '-update'}
                    <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                  {:else}
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                  {/if}
                </button>

                <!-- Uninstall button -->
                <button
                  onclick={() => showUninstallConfirm(p)}
                  disabled={actionLoading === p.name + '-uninstall'}
                  class="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg cursor-pointer transition-colors"
                  title={t.pluginUninstall || '卸载'}
                >
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
  {/if}

  {#if activeView === 'market'}
  <!-- Plugin Market -->
  <div class="bg-white rounded-xl border border-gray-100">
    <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
      <h3 class="text-sm font-medium text-gray-700">{t.pluginMarket || '插件市场'}</h3>
      <button
        onclick={loadRegistry}
        disabled={registryLoading}
        class="text-xs text-gray-500 hover:text-gray-700 cursor-pointer"
      >
        {registryLoading ? '...' : '↻'}
      </button>
    </div>

    {#if registryLoading && registryPlugins.length === 0}
      <div class="p-8 text-center text-gray-400 text-sm">{t.loading || '加载中...'}</div>
    {:else if registryPlugins.length === 0}
      <div class="p-8 text-center">
        <div class="text-3xl mb-2">🌐</div>
        <div class="text-sm text-gray-500">{t.pluginMarketEmpty || '暂无可用插件'}</div>
        <div class="text-xs text-gray-400 mt-1">{t.pluginMarketSource || '插件来源: redc.wgpsec.org'}</div>
      </div>
    {:else}
      <div class="divide-y divide-gray-50">
        {#each registryPlugins as rp}
          <div class="px-4 py-3 hover:bg-gray-50/50 transition-colors">
            <div class="flex items-start justify-between gap-3">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium text-gray-900 truncate">{rp.name}</span>
                  <span class="text-xs text-gray-400">v{rp.version}</span>
                  {#if rp.category}
                    <span class="text-xs px-1.5 py-0.5 bg-gray-100 text-gray-500 rounded">{rp.category}</span>
                  {/if}
                </div>
                <div class="text-xs text-gray-500 mt-0.5">{rp.description}</div>
                {#if rp.author}
                  <div class="text-xs text-gray-400 mt-0.5">by {rp.author}</div>
                {/if}
                {#if rp.tags?.length}
                  <div class="flex gap-1 mt-1 flex-wrap">
                    {#each rp.tags as tag}
                      <span class="text-xs px-1.5 py-0.5 bg-blue-50 text-blue-600 rounded">#{tag}</span>
                    {/each}
                  </div>
                {/if}
              </div>
              <div class="shrink-0">
                {#if installedMap.has(rp.name)}
                  {#if compareVersions(rp.version, installedMap.get(rp.name)) > 0}
                    <div class="flex items-center gap-2">
                      <span class="text-xs text-gray-400">v{installedMap.get(rp.name)} → v{rp.version}</span>
                      <button
                        onclick={() => handleUpdateFromRegistry(rp.name)}
                        disabled={actionLoading === rp.name + '-market-update'}
                        class="px-3 py-1.5 text-sm bg-amber-500 text-white rounded-lg hover:bg-amber-600 disabled:opacity-50 cursor-pointer transition-colors"
                      >
                        {actionLoading === rp.name + '-market-update' ? (t.pluginUpdating || '更新中...') : (t.pluginUpdate || '更新')}
                      </button>
                    </div>
                  {:else}
                    <span class="px-3 py-1.5 text-xs text-emerald-600 bg-emerald-50 rounded-lg">{t.pluginAlreadyInstalled || '已安装'}</span>
                  {/if}
                {:else}
                  <button
                    onclick={() => handleInstallFromRegistry(rp.url)}
                    disabled={actionLoading === rp.url}
                    class="px-3 py-1.5 text-sm bg-gray-900 text-white rounded-lg hover:bg-gray-800 disabled:opacity-50 cursor-pointer transition-colors"
                  >
                    {actionLoading === rp.url ? (t.pluginInstalling || '安装中...') : (t.pluginInstall || '安装')}
                  </button>
                {/if}
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Manual install -->
  <div class="bg-white rounded-xl border border-gray-100 p-4">
    <h3 class="text-sm font-medium text-gray-700 mb-3">{t.pluginInstallManual || '手动安装'}</h3>
    <div class="flex gap-2">
      <input
        type="text"
        bind:value={installSource}
        placeholder={t.pluginInstallPlaceholder || 'Git 仓库地址或本地路径'}
        class="flex-1 px-3 py-2 text-sm border border-gray-200 rounded-lg bg-gray-50 focus:outline-none focus:ring-1 focus:ring-gray-900 focus:border-gray-900"
        onkeydown={(e) => e.key === 'Enter' && handleInstall()}
      />
      <button
        onclick={handleInstall}
        disabled={installing || !installSource.trim()}
        class="px-4 py-2 bg-gray-900 text-white text-sm rounded-lg hover:bg-gray-800 disabled:opacity-50 cursor-pointer transition-colors"
      >
        {installing ? (t.pluginInstalling || '安装中...') : (t.pluginInstall || '安装')}
      </button>
    </div>
  </div>
  {/if}
</div>

<!-- Config Modal -->
{#if configModal.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => configModal = { show: false, plugin: null, config: '', schema: null, saving: false }}>
    <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4" onclick={(e) => e.stopPropagation()}>
      <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-sm font-medium text-gray-900">{t.pluginConfig || '插件配置'} — {configModal.plugin?.name}</h3>
        <button class="text-gray-400 hover:text-gray-600 cursor-pointer" onclick={() => configModal = { show: false, plugin: null, config: '', schema: null, saving: false }}>✕</button>
      </div>
      <div class="p-4">
        {#if configModal.schema && Object.keys(configModal.schema).length > 0}
          <div class="text-xs text-gray-500 mb-3">{t.pluginConfigFields || '配置字段'}:</div>
          <div class="space-y-1 mb-3">
            {#each Object.entries(configModal.schema) as [key, field]}
              <div class="text-xs text-gray-400">
                <span class="font-mono text-gray-600">{key}</span>
                {#if field.required}<span class="text-red-400">*</span>{/if}
                — {field.description || field.type}
              </div>
            {/each}
          </div>
        {/if}
        <textarea
          bind:value={configModal.config}
          rows="8"
          class="w-full px-3 py-2 text-sm font-mono border border-gray-200 rounded-lg bg-gray-50 focus:outline-none focus:ring-1 focus:ring-gray-900 focus:border-gray-900 resize-none"
          placeholder="JSON config..."
        ></textarea>
      </div>
      <div class="px-4 py-3 border-t border-gray-100 flex justify-end gap-2">
        <button
          onclick={() => configModal = { show: false, plugin: null, config: '', schema: null, saving: false }}
          class="px-3 py-1.5 text-sm text-gray-600 border border-gray-300 rounded-lg hover:bg-gray-50 cursor-pointer"
        >{t.cancel || '取消'}</button>
        <button
          onclick={saveConfig}
          disabled={configModal.saving}
          class="px-3 py-1.5 text-sm bg-gray-900 text-white rounded-lg hover:bg-gray-800 disabled:opacity-50 cursor-pointer"
        >{configModal.saving ? (t.saving || '保存中...') : (t.save || '保存')}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Confirm Modal -->
{#if confirmModal.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => confirmModal = { show: false, action: '', pluginName: '', message: '' }}>
    <div class="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4" onclick={(e) => e.stopPropagation()}>
      <div class="p-4 text-center">
        <div class="text-3xl mb-2">⚠️</div>
        <div class="text-sm text-gray-700">{confirmModal.message}</div>
      </div>
      <div class="px-4 py-3 border-t border-gray-100 flex justify-end gap-2">
        <button
          onclick={() => confirmModal = { show: false, action: '', pluginName: '', message: '' }}
          class="px-3 py-1.5 text-sm text-gray-600 border border-gray-300 rounded-lg hover:bg-gray-50 cursor-pointer"
        >{t.cancel || '取消'}</button>
        <button
          onclick={handleConfirmAction}
          class="px-3 py-1.5 text-sm bg-red-600 text-white rounded-lg hover:bg-red-700 cursor-pointer"
        >{t.pluginConfirmBtn || '确认卸载'}</button>
      </div>
    </div>
  </div>
{/if}
