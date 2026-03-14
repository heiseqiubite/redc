<script>
  import { onMount } from 'svelte';
  import { ListCases, GetResourceSummary, GetBalances, GetBills, ListTemplates, ListProjects, TestTerraformEndpoints, GetTotalRuntime, ListScheduledTasks, StartCase, StopCase } from '../../../wailsjs/go/main/App.js';
  import { toast } from '../../lib/toast.js';

  let { t, onTabChange = () => {} } = $props();

  // 翻译凭据错误信息
  function translateError(error) {
    if (!error) return '';
    // 中文厂商名映射到英文键名
    const providerMap = {
      '阿里云': 'Aliyun',
      '腾讯云': 'Tencent',
      '火山引擎': 'Volcengine',
      '华为云': 'Huawei',
      'UCloud': 'UCloud',
      'Vultr': 'Vultr',
      'AWS': 'AWS',
      'GCP': 'GCP',
      'Azure': 'Azure'
    };
    let provider = error.replace('未配置', '').replace('凭据', '').trim();
    const key = 'noCredentials' + (providerMap[provider] || provider);
    return t[key] || error;
  }
  
  // Dashboard state
  let stats = $state({
    totalCases: 0,
    runningCases: 0,
    stoppedCases: 0,
    errorCases: 0
  });
  
  let resourceSummary = $state([]);
  let balances = $state([]);
  let balancesLoading = $state(false);
  let bills = $state([]);
  let billsLoading = $state(false);
  let recentCases = $state([]);
  let loading = $state(true);
  
  // Real data for templates and projects
  let templateCount = $state(0);
  let projectCount = $state(0);
  
  // Network diagnostics
  let networkChecks = $state([]);
  let networkCheckLoading = $state(false);
  
  // Real data for runtime and scheduled tasks
  let totalRuntime = $state('0h');
  let scheduledTaskCount = $state(0);
  
  // Quick stats with real data — clickable navigation targets
  let quickStats = $derived([
    { label: t.scheduledTasks || '定时任务', value: String(scheduledTaskCount), icon: 'clock', tab: 'taskCenter' },
    { label: t.runtime, value: totalRuntime, icon: 'timer', tab: null },
    { label: t.templateCount, value: String(templateCount), icon: 'template', tab: 'localTemplates' },
    { label: t.projectCount, value: String(projectCount), icon: 'project', tab: 'credentials' }
  ]);
  
  onMount(async () => {
    await loadDashboardData();
    await runNetworkCheck();
    await loadRuntime();
  });
  
  async function loadDashboardData() {
    loading = true;
    try {
      // Load cases
      const cases = await ListCases();
      stats.totalCases = cases.length;
      stats.runningCases = cases.filter(c => c.state === 'running').length;
      stats.stoppedCases = cases.filter(c => c.state === 'stopped').length;
      stats.errorCases = cases.filter(c => c.state === 'error').length;
      
      // Get recent cases (last 5)
      recentCases = cases.slice(0, 5);
      
      // Load templates count
      try {
        const templates = await ListTemplates();
        templateCount = templates.length;
      } catch (e) {
        console.error('Failed to load templates:', e);
        templateCount = 0;
      }
      
      // Load projects count
      try {
        const projects = await ListProjects();
        projectCount = projects.length;
      } catch (e) {
        console.error('Failed to load projects:', e);
        projectCount = 0;
      }
      
      // Load resource summary
      try {
        resourceSummary = await GetResourceSummary();
      } catch (e) {
        console.error('Failed to load resource summary:', e);
      }
    } catch (e) {
      console.error('Failed to load dashboard data:', e);
    } finally {
      loading = false;
    }
  }
  
  async function queryBalances() {
    balancesLoading = true;
    balances = [];
    try {
      balances = await GetBalances(['aliyun', 'tencentcloud', 'volcengine', 'huaweicloud', 'ucloud', 'vultr']);
    } catch (e) {
      console.error('Failed to load balances:', e);
      balances = [];
    } finally {
      balancesLoading = false;
    }
  }
  
  async function queryBills() {
    billsLoading = true;
    bills = [];
    try {
      bills = await GetBills(['aws', 'vultr']);
    } catch (e) {
      console.error('Failed to load bills:', e);
      bills = [];
    } finally {
      billsLoading = false;
    }
  }
  
  function getStateColor(state) {
    const colors = {
      'running': 'text-emerald-600 bg-emerald-50',
      'stopped': 'text-slate-500 bg-slate-50',
      'error': 'text-red-600 bg-red-50',
      'created': 'text-blue-600 bg-blue-50'
    };
    return colors[state] || 'text-gray-600 bg-gray-50';
  }
  
  function navigateToCases() {
    onTabChange('cases');
  }
  
  async function runNetworkCheck() {
    networkCheckLoading = true;
    try {
      networkChecks = await TestTerraformEndpoints();
    } catch (e) {
      console.error('Failed to run network check:', e);
      networkChecks = [];
    } finally {
      networkCheckLoading = false;
    }
  }
  
  async function loadRuntime() {
    try {
      totalRuntime = await GetTotalRuntime();
    } catch (e) {
      console.error('Failed to load runtime:', e);
      totalRuntime = '0h';
    }
    
    try {
      const tasks = await ListScheduledTasks();
      scheduledTaskCount = tasks ? tasks.length : 0;
    } catch (e) {
      console.error('Failed to load scheduled tasks:', e);
      scheduledTaskCount = 0;
    }
  }

  // Case action loading states
  let actionLoading = $state({});

  async function handleStartCase(e, caseId) {
    e.stopPropagation();
    actionLoading[caseId] = 'start';
    actionLoading = actionLoading;
    try {
      await StartCase(caseId);
      toast.success(t.caseStarted || '场景已启动');
      await loadDashboardData();
    } catch (err) {
      toast.error(`${t.startFailed || '启动失败'}: ${err.message || err}`);
    } finally {
      delete actionLoading[caseId];
      actionLoading = actionLoading;
    }
  }

  async function handleStopCase(e, caseId) {
    e.stopPropagation();
    actionLoading[caseId] = 'stop';
    actionLoading = actionLoading;
    try {
      await StopCase(caseId);
      toast.success(t.caseStopped || '场景已停止');
      await loadDashboardData();
    } catch (err) {
      toast.error(`${t.stopFailed || '停止失败'}: ${err.message || err}`);
    } finally {
      delete actionLoading[caseId];
      actionLoading = actionLoading;
    }
  }

  function handleSSH(e, c) {
    e.stopPropagation();
    onTabChange('cases');
  }

  function navigateToCreate() {
    onTabChange('customDeployment');
  }

  function navigateToCredentials() {
    onTabChange('credentials');
  }
</script>

<div class="space-y-3">
  <!-- Stats Row: 4 scene stats + 4 quick stats in one row -->
  <div class="grid grid-cols-4 sm:grid-cols-8 gap-3">
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.totalScenes || '总场景数'}</div>
      <div class="text-[22px] font-bold text-gray-900">{stats.totalCases}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.runningScenes || '运行中'}</div>
      <div class="text-[22px] font-bold text-emerald-600">{stats.runningCases}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.stoppedScenes || '已停止'}</div>
      <div class="text-[22px] font-bold text-gray-400">{stats.stoppedCases}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.errorScenes || '异常'}</div>
      <div class="text-[22px] font-bold text-red-500">{stats.errorCases}</div>
    </div>
    {#each quickStats as stat}
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
      <div 
        class="bg-white rounded-xl border border-gray-100 p-4 {stat.tab ? 'cursor-pointer hover:border-gray-200 hover:shadow-sm transition-all' : ''}"
        onclick={() => stat.tab && onTabChange(stat.tab)}
      >
        <div class="text-[11px] text-gray-500 mb-1">{stat.label}</div>
        <div class="text-[22px] font-bold text-gray-900">{stat.value}</div>
      </div>
    {/each}
  </div>
  
  <!-- Main Content Grid (unified single grid) -->
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-3">
    <!-- Left Column: Recent Cases + Balance/Bill -->
    <div class="lg:col-span-2 space-y-3">
    <!-- Recent Cases -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[13px] font-semibold text-gray-900">{t.recentScenes || '最近场景'}</h3>
        <div class="flex items-center gap-2">
          <button
            class="inline-flex items-center gap-1 h-7 px-2.5 text-white bg-gray-900 hover:bg-gray-800 text-[11px] font-medium rounded-lg transition-colors cursor-pointer"
            onclick={navigateToCreate}
          >
            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" /></svg>
            {t.quickCreate || '快速创建'}
          </button>
          <button 
            class="text-[11px] text-gray-500 hover:text-gray-700 font-medium cursor-pointer"
            onclick={navigateToCases}
          >
            {t.viewAll || '查看全部'} →
          </button>
        </div>
      </div>
      <div class="divide-y divide-gray-50">
        {#if loading}
          <div class="px-5 py-8 text-center text-[13px] text-gray-400">
            {t.loading || '加载中...'}
          </div>
        {:else if recentCases.length === 0}
          <div class="px-5 py-10 text-center">
            <svg class="w-8 h-8 mx-auto mb-2 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
              <path stroke-linecap="round" stroke-linejoin="round" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
            </svg>
            <div class="text-[12px] text-gray-400 mb-3">{t.noRecentScenes || '暂无场景'}</div>
            <button
              class="inline-flex items-center gap-1.5 h-7 px-3 text-[11px] text-white bg-gray-900 hover:bg-gray-800 rounded-lg transition-colors font-medium cursor-pointer"
              onclick={navigateToCreate}
            >
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" /></svg>
              {t.createFirstScene || '创建第一个场景'}
            </button>
          </div>
        {:else}
          {#each recentCases as c}
            <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
            <div class="px-4 py-2.5 hover:bg-gray-50/50 transition-colors cursor-pointer" onclick={navigateToCases}>
              <div class="flex items-center justify-between">
                <div class="flex-1 min-w-0">
                  <div class="text-[12px] font-medium text-gray-900 truncate">{c.name}</div>
                  <div class="text-[10px] text-gray-400 mt-0.5">{c.type} · {c.stateTime}</div>
                </div>
                <div class="flex items-center gap-1.5">
                  {#if c.state === 'running'}
                    <button
                      class="w-6 h-6 flex items-center justify-center text-red-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors disabled:opacity-50 cursor-pointer"
                      onclick={(e) => handleStopCase(e, c.id)}
                      disabled={actionLoading[c.id]}
                      title={t.stop || '停止'}
                    >
                      {#if actionLoading[c.id] === 'stop'}
                        <div class="w-3 h-3 border-2 border-red-200 border-t-red-500 rounded-full animate-spin"></div>
                      {:else}
                        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 7.5A2.25 2.25 0 017.5 5.25h9a2.25 2.25 0 012.25 2.25v9a2.25 2.25 0 01-2.25 2.25h-9a2.25 2.25 0 01-2.25-2.25v-9z" /></svg>
                      {/if}
                    </button>
                    <button
                      class="w-6 h-6 flex items-center justify-center text-gray-400 hover:text-gray-700 hover:bg-gray-100 rounded transition-colors cursor-pointer"
                      onclick={(e) => handleSSH(e, c)}
                      title="SSH"
                    >
                      <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" /></svg>
                    </button>
                  {:else if c.state === 'stopped' || c.state === 'created'}
                    <button
                      class="w-6 h-6 flex items-center justify-center text-emerald-400 hover:text-emerald-600 hover:bg-emerald-50 rounded transition-colors disabled:opacity-50 cursor-pointer"
                      onclick={(e) => handleStartCase(e, c.id)}
                      disabled={actionLoading[c.id]}
                      title={t.start || '启动'}
                    >
                      {#if actionLoading[c.id] === 'start'}
                        <div class="w-3 h-3 border-2 border-emerald-200 border-t-emerald-500 rounded-full animate-spin"></div>
                      {:else}
                        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z" /></svg>
                      {/if}
                    </button>
                  {/if}
                  <span class="inline-flex items-center gap-1 px-2 py-0.5 text-[10px] font-medium rounded-full {getStateColor(c.state)}">
                    <span class="w-1.5 h-1.5 rounded-full bg-current"></span>
                    {c.state}
                  </span>
                </div>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <!-- Balance + Bill sub-grid under Recent Cases -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <!-- Account Balances -->
      <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
        <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
          <h3 class="text-[13px] font-semibold text-gray-900">{t.accountBalance || '账户余额'}</h3>
          <button 
            onclick={queryBalances}
            disabled={balancesLoading}
            class="h-6 px-2 text-gray-500 hover:text-gray-700 hover:bg-gray-50 text-[10px] font-medium rounded transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1"
          >
            <svg class="w-3 h-3 {balancesLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg>
            {balancesLoading ? (t.loading || '...') : (t.queryBalance || '查询')}
          </button>
        </div>
        <div class="px-4 py-3">
          {#if balancesLoading}
            <div class="text-center py-4 text-[12px] text-gray-400">
              {t.loading || '加载中...'}
            </div>
          {:else if balances.length === 0}
            <div class="text-center py-4 text-[12px] text-gray-400">
              {t.clickToQueryBalance || '点击上方按钮查询账户余额'}
            </div>
          {:else}
            <div class="divide-y divide-gray-50">
              {#each balances as balance}
                <div class="flex items-center justify-between py-1.5">
                  <span class="text-[11px] text-gray-500">{balance.provider}</span>
                  {#if balance.error}
                    <span class="text-[10px] text-gray-400">{translateError(balance.error)}</span>
                  {:else}
                    <span class="text-[12px] font-medium text-gray-900 tabular-nums">{balance.currency} {balance.amount}</span>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>

      <!-- Current Month Bill -->
      <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
        <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
          <h3 class="text-[13px] font-semibold text-gray-900">{t.currentMonthBill || '当月账单'}</h3>
          <button 
            onclick={queryBills}
            disabled={billsLoading}
            class="h-6 px-2 text-gray-500 hover:text-gray-700 hover:bg-gray-50 text-[10px] font-medium rounded transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1"
          >
            <svg class="w-3 h-3 {billsLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg>
            {billsLoading ? (t.loading || '...') : (t.queryBill || '查询')}
          </button>
        </div>
        <div class="px-4 py-3">
          {#if billsLoading}
            <div class="text-center py-4 text-[12px] text-gray-400">
              {t.loading || '加载中...'}
            </div>
          {:else if bills.length === 0}
            <div class="text-center py-2">
              <div class="text-[12px] text-gray-400">{t.clickToQueryBill || '点击上方按钮查询当月账单'}</div>
              <div class="mt-1 text-[10px] text-amber-500">{t.billCostWarning || 'AWS Cost Explorer 每次查询约 $0.01'}</div>
            </div>
          {:else}
            <div class="divide-y divide-gray-50">
              {#each bills as bill}
                <div class="flex items-center justify-between py-1.5">
                  <span class="text-[11px] text-gray-500 uppercase">{bill.provider}</span>
                  {#if bill.error}
                    <span class="text-[10px] text-gray-400">{translateError(bill.error)}</span>
                  {:else}
                    <span class="text-[12px] font-medium text-gray-900 tabular-nums">{bill.currency} {bill.totalAmount}</span>
                  {/if}
                </div>
              {/each}
              {#if bills.length > 0 && bills[0].startDate}
                <div class="pt-1.5">
                  <span class="text-[10px] text-gray-400">{bills[0].startDate} ~ {bills[0].endDate}</span>
                </div>
              {/if}
            </div>
          {/if}
        </div>
      </div>
    </div>
    </div> <!-- close left column wrapper -->
    
    <!-- Right column: Network + Quick Links + Resource -->
    <div class="space-y-3">
      <!-- Network Diagnostics -->
      <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
        <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
          <h3 class="text-[13px] font-semibold text-gray-900">{t.networkCheck || '网络诊断'}</h3>
          <button
            class="h-6 px-2 text-gray-500 hover:text-gray-700 hover:bg-gray-50 text-[10px] font-medium rounded transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1"
            onclick={runNetworkCheck}
            disabled={networkCheckLoading}
          >
            <svg class="w-3 h-3 {networkCheckLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg>
            {networkCheckLoading ? t.networkChecking : t.retest}
          </button>
        </div>
        <div class="px-4 py-3">
          {#if networkCheckLoading}
            <div class="text-center py-6 text-[12px] text-gray-400">
              {t.loading || '加载中...'}
            </div>
          {:else if networkChecks.length === 0}
            <div class="text-center py-6 text-[12px] text-gray-400">
              {t.noNetworkData || '暂无网络诊断数据'}
            </div>
          {:else}
            <div class="divide-y divide-gray-50">
              {#each networkChecks as item}
                <div class="flex items-center justify-between py-1.5">
                  <div class="flex items-center gap-2 flex-1 min-w-0">
                    <div class="w-1.5 h-1.5 rounded-full flex-shrink-0 {item.ok ? 'bg-emerald-500' : 'bg-red-500'}"></div>
                    <span class="text-[11px] text-gray-600 truncate">{item.name}</span>
                  </div>
                  <div class="flex items-center gap-2 flex-shrink-0">
                    <span class="text-[10px] text-gray-400 tabular-nums">{item.latencyMs}ms</span>
                    <span class="text-[10px] font-medium {item.ok ? 'text-emerald-500' : 'text-red-500'} w-8 text-right">
                      {item.ok ? 'OK' : 'FAIL'}
                    </span>
                  </div>
                </div>
              {/each}
            </div>
            {#if networkChecks.some(item => !item.ok)}
              <div class="mt-2 pt-2 border-t border-gray-100">
                <button
                  class="text-[10px] text-gray-500 hover:text-gray-700 font-medium cursor-pointer"
                  onclick={navigateToCredentials}
                >
                  {t.goToCredentials || '前往凭据管理'} →
                </button>
              </div>
            {/if}
          {/if}
        </div>
      </div>

      <!-- Quick Links -->
      <div class="bg-white rounded-xl border border-gray-100 p-4">
        <h3 class="text-[13px] font-semibold text-gray-900 mb-3">{t.quickLinks || '快捷入口'}</h3>
        <div class="grid grid-cols-2 gap-2">
          <button class="flex items-center gap-2 px-3 py-2 text-[11px] text-gray-600 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer" onclick={() => onTabChange('aiChat')}>
            <svg class="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155" /></svg>
            {t.aiChat || 'AI 对话'}
          </button>
          <button class="flex items-center gap-2 px-3 py-2 text-[11px] text-gray-600 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer" onclick={() => onTabChange('registry')}>
            <svg class="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" /></svg>
            {t.registry || '模板仓库'}
          </button>
          <button class="flex items-center gap-2 px-3 py-2 text-[11px] text-gray-600 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer" onclick={() => onTabChange('sshManager')}>
            <svg class="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" /></svg>
            {t.ssh || 'SSH 管理'}
          </button>
          <button class="flex items-center gap-2 px-3 py-2 text-[11px] text-gray-600 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer" onclick={() => onTabChange('credentials')}>
            <svg class="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" /></svg>
            {t.credentials || '凭据管理'}
          </button>
        </div>
      </div>

      <!-- Resource Summary -->
      <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
        <div class="px-4 py-3 border-b border-gray-100">
          <h3 class="text-[13px] font-semibold text-gray-900">{t.resourceSummary || '资源概览'}</h3>
        </div>
        <div class="px-4 py-3">
          {#if loading}
            <div class="text-center py-4 text-[12px] text-gray-400">
              {t.loading || '加载中...'}
            </div>
          {:else if resourceSummary.length === 0}
            <div class="text-center py-4 text-[12px] text-gray-400">
              {t.noResources || '暂无资源'}
            </div>
          {:else}
            <div class="divide-y divide-gray-50">
              {#each resourceSummary.slice(0, 8) as resource}
                <div class="flex items-center justify-between py-1.5">
                  <span class="text-[11px] text-gray-500">{resource.type}</span>
                  <span class="text-[12px] font-medium text-gray-900 tabular-nums">{resource.count}</span>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>
