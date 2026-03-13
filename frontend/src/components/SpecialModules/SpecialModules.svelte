<script>
  import { onMount } from 'svelte';
  import { loadUserdataTemplates } from '../../lib/userdataTemplates.js';
  
  let { t, onTabChange } = $props();
  let templates = $state([]);
  let loading = $state(true);
  let searchQuery = $state('');
  let activeCategory = $state('all');
  let selectedTemplate = $state(null);
  let copied = $state(false);

  onMount(async () => {
    templates = await loadUserdataTemplates();
    loading = false;
  });

  const categories = $derived(() => {
    const cats = new Map();
    cats.set('all', { key: 'all', label: t.userdataAll || '全部', count: templates.length });
    for (const tmpl of templates) {
      const cat = tmpl.category || 'other';
      if (!cats.has(cat)) {
        cats.set(cat, { key: cat, label: categoryLabel(cat), count: 0 });
      }
      cats.get(cat).count++;
    }
    return [...cats.values()];
  });

  const filteredTemplates = $derived(() => {
    let list = templates;
    if (activeCategory !== 'all') {
      list = list.filter(t => t.category === activeCategory);
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter(t =>
        (t.nameZh || '').toLowerCase().includes(q) ||
        t.name.toLowerCase().includes(q) ||
        (t.description || '').toLowerCase().includes(q) ||
        (t.cveId || '').toLowerCase().includes(q)
      );
    }
    return list;
  });

  function categoryLabel(cat) {
    const map = {
      vulhub: 'Vulhub',
      c2: 'C2',
      ai: 'AI',
      basic: t.userdataCatBasic || '基础环境',
      security: t.userdataCatSecurity || '安全工具',
      other: t.userdataCatOther || '其他'
    };
    return map[cat] || cat;
  }

  function categoryIcon(cat) {
    const map = {
      vulhub: '🐛', c2: '🎯', ai: '🤖',
      basic: '📦', security: '🔒', other: '📋'
    };
    return map[cat] || '📋';
  }

  function selectTemplate(tmpl) {
    selectedTemplate = tmpl;
    copied = false;
  }

  async function copyScript() {
    if (!selectedTemplate) return;
    try {
      await navigator.clipboard.writeText(selectedTemplate.script);
      copied = true;
      setTimeout(() => copied = false, 2000);
    } catch (e) {
      console.error('Failed to copy:', e);
    }
  }
</script>

<div class="p-6 h-full flex flex-col gap-4">
  {#if loading}
    <div class="flex items-center justify-center h-32">
      <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
    </div>
  {:else if templates.length === 0}
    <!-- Empty state -->
    <div class="bg-white rounded-xl border border-gray-100 p-8 text-center">
      <div class="text-3xl mb-3">📜</div>
      <p class="text-sm text-gray-600 mb-2">{t.noUserdataTemplatesHint || '暂无 Userdata 脚本模板'}</p>
      <p class="text-xs text-gray-400 mb-4">{t.noUserdataHint2 || '请先从模板仓库拉取包含 userdata 脚本的模板'}</p>
      <button
        class="px-4 py-2 bg-gray-900 text-white text-sm rounded-lg hover:bg-gray-800 cursor-pointer transition-colors"
        onclick={() => onTabChange && onTabChange('registry')}
      >
        {t.noUserdataTemplatesHintButton || '前往模板仓库'}
      </button>
    </div>
  {:else}
    <!-- Search -->
    <div class="flex gap-3">
      <input
        type="text"
        bind:value={searchQuery}
        placeholder={t.userdataSearchPlaceholder || '搜索脚本名称、CVE 编号...'}
        class="flex-1 px-3 py-2 text-sm border border-gray-200 rounded-lg bg-gray-50 focus:outline-none focus:ring-1 focus:ring-gray-900 focus:border-gray-900"
      />
    </div>

    <!-- Category filter -->
    <div class="flex gap-1.5 flex-wrap">
      {#each categories() as cat}
        <button
          onclick={() => { activeCategory = cat.key; selectedTemplate = null; }}
          class="px-3 py-1.5 text-xs rounded-lg transition-colors cursor-pointer {activeCategory === cat.key ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}"
        >
          {cat.label}
          <span class="ml-1 opacity-60">{cat.count}</span>
        </button>
      {/each}
    </div>

    <!-- Content: list + detail -->
    <div class="flex-1 flex gap-4 min-h-0">
      <!-- Left: script list -->
      <div class="w-1/3 min-w-[200px] bg-white rounded-xl border border-gray-100 overflow-hidden flex flex-col">
        <div class="px-3 py-2 border-b border-gray-50 text-xs text-gray-400">
          {filteredTemplates().length} {t.userdataItems || '个脚本'}
        </div>
        <div class="flex-1 overflow-y-auto">
          {#if filteredTemplates().length === 0}
            <div class="p-4 text-center text-xs text-gray-400">{t.noResults || '无匹配结果'}</div>
          {:else}
            {#each filteredTemplates() as tmpl}
              <button
                onclick={() => selectTemplate(tmpl)}
                class="w-full px-3 py-2.5 text-left border-b border-gray-50 hover:bg-gray-50 transition-colors cursor-pointer {selectedTemplate?.name === tmpl.name ? 'bg-gray-50 border-l-2 border-l-gray-900' : 'border-l-2 border-l-transparent'}"
              >
                <div class="flex items-center gap-2">
                  <span class="text-sm">{categoryIcon(tmpl.category)}</span>
                  <div class="flex-1 min-w-0">
                    <div class="text-[13px] font-medium text-gray-900 truncate">{tmpl.nameZh || tmpl.name}</div>
                    <div class="flex items-center gap-1.5 mt-0.5">
                      <span class="text-[11px] text-gray-400">{tmpl.type || 'bash'}</span>
                      {#if tmpl.cveId}
                        <span class="text-[11px] text-amber-600 font-medium">{tmpl.cveId}</span>
                      {/if}
                    </div>
                  </div>
                </div>
              </button>
            {/each}
          {/if}
        </div>
      </div>

      <!-- Right: detail -->
      <div class="flex-1 bg-white rounded-xl border border-gray-100 overflow-hidden flex flex-col">
        {#if selectedTemplate}
          <!-- Header -->
          <div class="px-4 py-3 border-b border-gray-100">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-sm font-medium text-gray-900">{selectedTemplate.nameZh || selectedTemplate.name}</h3>
                <div class="flex items-center gap-2 mt-1">
                  <span class="text-xs px-1.5 py-0.5 bg-gray-100 text-gray-500 rounded">{categoryLabel(selectedTemplate.category)}</span>
                  {#if selectedTemplate.cveId}
                    <span class="text-xs px-1.5 py-0.5 bg-amber-50 text-amber-700 rounded font-medium">{selectedTemplate.cveId}</span>
                  {/if}
                  {#if selectedTemplate.level}
                    <span class="text-xs px-1.5 py-0.5 rounded {selectedTemplate.level === 'critical' ? 'bg-red-50 text-red-600' : 'bg-amber-50 text-amber-600'}">
                      {selectedTemplate.level === 'critical' ? (t.severityCritical || '严重') : (t.severityHigh || '高危')}
                    </span>
                  {/if}
                </div>
              </div>
              <button
                onclick={copyScript}
                class="px-3 py-1.5 text-sm rounded-lg transition-colors cursor-pointer {copied ? 'bg-emerald-500 text-white' : 'bg-gray-900 text-white hover:bg-gray-800'}"
              >
                {copied ? (t.copiedSuccess || '已复制') : (t.copyScript || '复制脚本')}
              </button>
            </div>
          </div>

          <!-- Meta info -->
          {#if selectedTemplate.description || selectedTemplate.environment}
            <div class="px-4 py-3 border-b border-gray-50 space-y-2">
              {#if selectedTemplate.description}
                <p class="text-xs text-gray-600">{selectedTemplate.description}</p>
              {/if}
              {#if selectedTemplate.environment}
                <div class="flex gap-3 text-xs text-gray-500">
                  {#if selectedTemplate.environment.port}
                    <span>端口: <span class="text-gray-700">{selectedTemplate.environment.port}</span></span>
                  {/if}
                  {#if selectedTemplate.environment.image}
                    <span>镜像: <span class="text-gray-700 font-mono">{selectedTemplate.environment.image}</span></span>
                  {/if}
                </div>
              {/if}
              {#if selectedTemplate.installNotes}
                <p class="text-xs text-amber-600">⚠ {selectedTemplate.installNotes}</p>
              {/if}
            </div>
          {/if}

          <!-- Script -->
          <div class="flex-1 overflow-auto">
            <pre class="px-4 py-3 text-[12px] text-gray-100 bg-gray-900 font-mono leading-relaxed h-full overflow-auto m-0 rounded-none">{selectedTemplate.script}</pre>
          </div>
        {:else}
          <!-- No selection -->
          <div class="flex-1 flex items-center justify-center">
            <div class="text-center">
              <div class="text-3xl mb-2">📜</div>
              <p class="text-sm text-gray-500">{t.userdataSelectHint || '选择一个脚本查看详情'}</p>
            </div>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>
