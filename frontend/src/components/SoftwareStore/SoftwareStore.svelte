<script>
  import { onMount, onDestroy } from 'svelte';
  import { ListCases, GetF8xCatalog, GetF8xCategories, GetF8xPresets, GetF8xStatus, EnsureF8x, RunF8xInstall, GetF8xInstallHistory, GetF8xRunningTasks, RefreshF8xCatalog } from '../../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff, BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';

  let { t, onTabChange } = $props();

  let catalog = $state([]);
  let categories = $state([]);
  let presets = $state([]);
  let cases = $state([]);
  let loading = $state(true);
  let catalogError = $state('');

  let selectedCaseID = $state('');
  let activeCategory = $state('all');
  let searchQuery = $state('');
  let selectedModules = $state(new Set());
  let f8xStatus = $state(null);
  let checkingStatus = $state(false);

  let installing = $state(false);
  let installLog = $state([]);
  let showLog = $state(false);
  let currentTaskID = $state('');
  let installHistory = $state([]);

  onMount(async () => {
    try {
      const [cat, cats, pre, caseList] = await Promise.all([
        GetF8xCatalog(),
        GetF8xCategories(),
        GetF8xPresets(),
        ListCases()
      ]);
      catalog = cat || [];
      categories = cats || [];
      presets = pre || [];
      cases = (caseList || []).filter(c => c.state === 'running');
      if (cases.length > 0) selectedCaseID = cases[0].id;
      if (catalog.length === 0) {
        catalogError = t.f8xCatalogError || '无法加载工具目录，请检查网络连接后点击 ⟳ 刷新';
      }
    } catch (e) {
      console.error('Failed to load f8x catalog:', e);
      catalogError = t.f8xCatalogError || '无法加载工具目录，请检查网络连接后点击 ⟳ 刷新';
    }
    loading = false;
  });

  // Listen for f8x events
  let cleanupOutput, cleanupDone;
  onMount(() => {
    cleanupOutput = EventsOn('f8x:output', (data) => {
      if (data.taskID === currentTaskID) {
        installLog = [...installLog, { type: data.type, text: data.text }];
      }
    });
    cleanupDone = EventsOn('f8x:done', (data) => {
      if (data.taskID === currentTaskID) {
        installing = false;
        installLog = [...installLog, { type: data.status === 'success' ? 'success' : 'error', text: data.status === 'success' ? '\n✅ Installation completed successfully!' : `\n❌ Installation failed: ${data.error || 'Unknown error'}` }];
        loadHistory();
      }
    });
  });
  onDestroy(() => {
    if (cleanupOutput) EventsOff('f8x:output');
    if (cleanupDone) EventsOff('f8x:done');
  });

  async function checkF8xStatus() {
    if (!selectedCaseID) return;
    checkingStatus = true;
    try {
      f8xStatus = await GetF8xStatus(selectedCaseID);
    } catch (e) {
      f8xStatus = { error: e.toString() };
    }
    checkingStatus = false;
  }

  async function loadHistory() {
    if (!selectedCaseID) return;
    try {
      installHistory = await GetF8xInstallHistory(selectedCaseID) || [];
    } catch(e) { /* ignore */ }
  }

  $effect(() => {
    if (selectedCaseID) {
      checkF8xStatus();
      loadHistory();
    }
  });

  const filteredModules = $derived(() => {
    let list = catalog;
    if (activeCategory !== 'all') {
      list = list.filter(m => m.category === activeCategory);
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter(m =>
        m.name.toLowerCase().includes(q) ||
        m.nameZh.toLowerCase().includes(q) ||
        (m.description || '').toLowerCase().includes(q) ||
        (m.descriptionZh || '').toLowerCase().includes(q) ||
        (m.tags || []).some(tag => tag.includes(q))
      );
    }
    return list;
  });

  function toggleModule(id) {
    const next = new Set(selectedModules);
    if (next.has(id)) next.delete(id); else next.add(id);
    selectedModules = next;
  }

  function selectPreset(preset) {
    const next = new Set();
    for (const flag of preset.flags) {
      const mod = catalog.find(m => m.flag === flag);
      if (mod) next.add(mod.id);
    }
    selectedModules = next;
  }

  async function installSelected() {
    if (selectedModules.size === 0 || !selectedCaseID || installing) return;
    const flags = [];
    for (const id of selectedModules) {
      const mod = catalog.find(m => m.id === id);
      if (mod) flags.push(mod.flag);
    }
    installing = true;
    installLog = [];
    showLog = true;
    try {
      currentTaskID = await RunF8xInstall(selectedCaseID, flags);
    } catch (e) {
      installing = false;
      installLog = [{ type: 'error', text: 'Failed to start: ' + e.toString() }];
    }
  }

  async function installSingle(mod) {
    if (!selectedCaseID || installing) return;
    installing = true;
    installLog = [];
    showLog = true;
    try {
      currentTaskID = await RunF8xInstall(selectedCaseID, [mod.flag]);
    } catch (e) {
      installing = false;
      installLog = [{ type: 'error', text: 'Failed to start: ' + e.toString() }];
    }
  }

  let catalogSource = $state('');
  let refreshing = $state(false);

  async function refreshCatalog() {
    refreshing = true;
    catalogError = '';
    try {
      const result = await RefreshF8xCatalog();
      if (result.success) {
        catalogSource = `${result.source} · v${result.version} · ${result.count} tools`;
      } else {
        catalogSource = '';
        catalogError = result.error || (t.f8xCatalogError || '无法加载工具目录，请检查网络连接');
      }
      const [cat, cats, pre] = await Promise.all([
        GetF8xCatalog(), GetF8xCategories(), GetF8xPresets()
      ]);
      catalog = cat || [];
      categories = cats || [];
      presets = pre || [];
      if (catalog.length > 0) catalogError = '';
    } catch(e) {
      catalogError = e.toString();
    }
    refreshing = false;
  }

  function categoryIcon(catId) {
    const icons = {
      'basic': '⚙️', 'development': '💻', 'pentest-recon': '🔍',
      'pentest-exploit': '💥', 'pentest-post': '🎯', 'blue-team': '🛡️',
      'red-infra': '🏗️', 'vuln-env': '🎪', 'misc': '🧰'
    };
    return icons[catId] || '📦';
  }
</script>

<div class="space-y-4">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <p class="text-[11px] text-gray-500">{t.f8xStoreDesc || '基于 f8x 的一站式工具安装平台，支持 150+ 渗透/开发/运维工具'}</p>
    </div>
    <div class="flex items-center gap-2">
      {#if catalogSource}
        <span class="text-[10px] px-2 py-0.5 rounded-full bg-blue-50 text-blue-600">{catalogSource}</span>
      {/if}
      <button onclick={refreshCatalog} disabled={refreshing} class="text-[11px] text-gray-400 hover:text-red-500 transition-colors disabled:opacity-40" title="刷新在线目录">
        {refreshing ? '⟳...' : '⟳'}
      </button>
      <button onclick={() => BrowserOpenURL('https://github.com/ffffffff0x/f8x')} class="text-[11px] text-gray-400 hover:text-red-500 transition-colors">
        GitHub ↗
      </button>
    </div>
  </div>

  <!-- Catalog Error Banner -->
  {#if catalogError}
    <div class="bg-red-50 border border-red-200 rounded-xl px-5 py-3 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.732-.833-2.5 0L4.268 16.5c-.77.833.192 2.5 1.732 2.5z"/></svg>
        <span class="text-[12px] text-red-700">{catalogError}</span>
      </div>
      <button onclick={refreshCatalog} disabled={refreshing} class="text-[11px] px-3 py-1 rounded-lg bg-red-100 hover:bg-red-200 text-red-700 disabled:opacity-40 transition-colors">
        {refreshing ? '⟳...' : '⟳ 重试'}
      </button>
    </div>
  {/if}

  <!-- Target VPS Selector -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <div class="px-5 py-3 flex items-center justify-between gap-4">
      <div class="flex items-center gap-3 flex-1">
        <span class="text-[12px] font-medium text-gray-700 whitespace-nowrap">{t.f8xTargetVPS || '目标主机'}</span>
        <select bind:value={selectedCaseID} class="text-[12px] border border-gray-200 rounded-lg px-3 py-1.5 bg-gray-50 focus:outline-none focus:ring-1 focus:ring-red-300 flex-1 max-w-xs">
          {#if cases.length === 0}
            <option value="">{t.f8xNoCases || '无可用主机（请先部署场景）'}</option>
          {:else}
            {#each cases as c}
              <option value={c.id}>{c.name || c.id} ({c.type})</option>
            {/each}
          {/if}
        </select>
        <button onclick={checkF8xStatus} disabled={!selectedCaseID || checkingStatus} class="text-[11px] px-3 py-1.5 rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600 disabled:opacity-40 transition-colors flex items-center gap-1.5">
          {#if checkingStatus}
            <div class="w-3 h-3 border-[1.5px] border-gray-300 border-t-gray-600 rounded-full animate-spin"></div>
            {t.f8xChecking || '检测中...'}
          {:else}
            {t.f8xCheckStatus || '检测状态'}
          {/if}
        </button>
        {#if f8xStatus && !checkingStatus}
          <span class="text-[11px] px-2.5 py-1 rounded-lg {f8xStatus.error ? 'bg-red-50 text-red-600 border border-red-200' : f8xStatus.deployed ? 'bg-green-50 text-green-700 border border-green-200' : 'bg-amber-50 text-amber-600 border border-amber-200'}">
            {f8xStatus.error ? '⚠ 连接失败' : f8xStatus.deployed ? `✓ f8x ${f8xStatus.version || '已部署'}` : '✗ f8x 未部署'}
          </span>
        {/if}
      </div>

      <div class="flex items-center gap-2">
        {#if selectedModules.size > 0}
          <span class="text-[11px] text-gray-500">{t.f8xSelected || '已选'} {selectedModules.size}</span>
          <button onclick={() => selectedModules = new Set()} class="text-[11px] text-gray-400 hover:text-gray-600">
            {t.f8xClearSelection || '清除'}
          </button>
        {/if}
        <button onclick={installSelected} disabled={selectedModules.size === 0 || !selectedCaseID || installing} class="text-[12px] px-4 py-1.5 rounded-lg bg-red-600 hover:bg-red-700 text-white disabled:opacity-40 transition-colors font-medium">
          {installing ? (t.f8xInstalling || '安装中...') : (t.f8xBatchInstall || '批量安装')}
        </button>
      </div>
    </div>
  </div>

  <!-- Presets -->
  <div class="flex items-center gap-2 flex-wrap">
    <span class="text-[11px] text-gray-500 mr-1">{t.f8xPresets || '快捷预设'}:</span>
    {#each presets as preset}
      <button onclick={() => selectPreset(preset)} class="text-[11px] px-3 py-1 rounded-full border border-gray-200 hover:border-red-300 hover:bg-red-50 text-gray-600 hover:text-red-600 transition-colors">
        {preset.nameZh || preset.name}
      </button>
    {/each}
  </div>

  <!-- Search + Category Tabs -->
  <div class="flex items-center gap-3">
    <div class="relative flex-1 max-w-xs">
      <input type="text" bind:value={searchQuery} placeholder={t.f8xSearch || '搜索工具...'} class="w-full text-[12px] border border-gray-200 rounded-lg pl-8 pr-3 py-1.5 bg-white focus:outline-none focus:ring-1 focus:ring-red-300" />
      <svg class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
    </div>
    <div class="flex items-center gap-1 flex-wrap flex-1">
      <button onclick={() => activeCategory = 'all'} class="text-[11px] px-2.5 py-1 rounded-lg transition-colors {activeCategory === 'all' ? 'bg-red-600 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}">
        {t.f8xAll || '全部'} ({catalog.length})
      </button>
      {#each categories as cat}
        <button onclick={() => activeCategory = cat.id} class="text-[11px] px-2.5 py-1 rounded-lg transition-colors {activeCategory === cat.id ? 'bg-red-600 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}">
          {categoryIcon(cat.id)} {cat.nameZh || cat.name} ({cat.count})
        </button>
      {/each}
    </div>
  </div>

  <!-- Tool Grid -->
  {#if loading}
    <div class="flex items-center justify-center py-12">
      <div class="w-5 h-5 border-2 border-red-200 border-t-red-600 rounded-full animate-spin"></div>
    </div>
  {:else}
    <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-3">
      {#each filteredModules() as mod}
        {@const isSelected = selectedModules.has(mod.id)}
        {@const isBatch = (mod.tags || []).includes('batch')}
        <div class="group bg-white rounded-xl border transition-all cursor-pointer {isSelected ? 'border-red-400 ring-1 ring-red-200 bg-red-50/30' : 'border-gray-100 hover:border-gray-200 hover:shadow-sm'}" onclick={() => toggleModule(mod.id)}>
          <div class="p-3">
            <div class="flex items-start justify-between mb-1.5">
              <h4 class="text-[12px] font-semibold text-gray-900 leading-tight">{mod.name}</h4>
              <div class="flex items-center gap-1">
                {#if isBatch}
                  <span class="text-[9px] px-1.5 py-0.5 rounded bg-amber-50 text-amber-600 font-medium">SUITE</span>
                {/if}
                <input type="checkbox" checked={isSelected} onclick={(e) => { e.stopPropagation(); toggleModule(mod.id); }} class="w-3.5 h-3.5 rounded border-gray-300 text-red-600 focus:ring-red-500 cursor-pointer" />
              </div>
            </div>
            <p class="text-[10px] text-gray-500 leading-relaxed line-clamp-2 mb-2">{mod.descriptionZh || mod.description}</p>
            <div class="flex items-center justify-between">
              <span class="text-[9px] text-gray-400 font-mono">{mod.flag}</span>
              <button onclick={(e) => { e.stopPropagation(); installSingle(mod); }} disabled={!selectedCaseID || installing} class="text-[10px] px-2 py-0.5 rounded bg-gray-100 hover:bg-red-100 text-gray-500 hover:text-red-600 disabled:opacity-30 transition-colors opacity-0 group-hover:opacity-100">
                {t.f8xInstall || '安装'}
              </button>
            </div>
          </div>
        </div>
      {/each}
    </div>

    {#if filteredModules().length === 0}
      <div class="text-center py-8 text-gray-400 text-[12px]">
        {t.f8xNoResults || '没有匹配的工具'}
      </div>
    {/if}
  {/if}

  <!-- Install Log Drawer -->
  {#if showLog}
    <div class="bg-gray-900 rounded-xl border border-gray-700 overflow-hidden">
      <div class="px-4 py-2 border-b border-gray-700 flex items-center justify-between">
        <div class="flex items-center gap-2">
          <span class="text-[12px] text-gray-300 font-medium">{t.f8xInstallLog || '安装日志'}</span>
          {#if installing}
            <div class="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
          {/if}
        </div>
        <div class="flex items-center gap-2">
          <button onclick={() => installLog = []} class="text-[10px] text-gray-500 hover:text-gray-300">{t.f8xClearLog || '清空'}</button>
          <button onclick={() => showLog = false} class="text-[10px] text-gray-500 hover:text-gray-300">✕</button>
        </div>
      </div>
      <div class="p-4 max-h-64 overflow-y-auto font-mono text-[11px] leading-relaxed whitespace-pre-wrap break-all">
        {#each installLog as entry}
          <span class="{entry.type === 'error' ? 'text-red-400' : entry.type === 'success' ? 'text-green-400' : entry.type === 'info' ? 'text-blue-400' : 'text-gray-300'}">{entry.text}</span>
        {/each}
        {#if installLog.length === 0}
          <span class="text-gray-600">{t.f8xWaitingOutput || '等待输出...'}</span>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Install History -->
  {#if installHistory.length > 0}
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-3 border-b border-gray-100">
        <h3 class="text-[13px] font-semibold text-gray-900">{t.f8xHistory || '安装历史'}</h3>
      </div>
      <div class="divide-y divide-gray-50">
        {#each installHistory.slice().reverse().slice(0, 10) as record}
          <div class="px-5 py-2 flex items-center justify-between">
            <div class="flex items-center gap-3">
              <span class="w-2 h-2 rounded-full {record.status === 'success' ? 'bg-green-400' : record.status === 'failed' ? 'bg-red-400' : 'bg-yellow-400'}"></span>
              <span class="text-[11px] font-mono text-gray-700">{record.flags}</span>
            </div>
            <div class="flex items-center gap-3">
              <span class="text-[10px] text-gray-400">{record.startedAt ? new Date(record.startedAt).toLocaleString() : ''}</span>
              <span class="text-[10px] px-2 py-0.5 rounded-full {record.status === 'success' ? 'bg-green-50 text-green-600' : 'bg-red-50 text-red-600'}">
                {record.status}
              </span>
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>
