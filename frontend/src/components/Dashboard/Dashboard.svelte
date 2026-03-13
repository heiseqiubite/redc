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
  
  // Quick stats with real data
  let quickStats = $derived([
    { label: t.scheduledTasks || '定时任务', value: String(scheduledTaskCount), change: '0', trend: 'neutral', mock: false },
    { label: t.runtime, value: totalRuntime, change: '0', trend: 'neutral', mock: false },
    { label: t.templateCount, value: String(templateCount), change: '0', trend: 'neutral', mock: false },
    { label: t.projectCount, value: String(projectCount), change: '0', trend: 'neutral', mock: false }
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
  <!-- Stats Cards -->
  <div class="grid grid-cols-4 gap-3">
    <div class="bg-white rounded-xl border border-gray-100 p-5">
      <div class="flex items-center justify-between mb-3">
        <div class="w-10 h-10 rounded-lg bg-blue-50 flex items-center justify-center">
          <svg class="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
          </svg>
        </div>
      </div>
      <div class="text-[28px] font-bold text-gray-900">{stats.totalCases}</div>
      <div class="text-[13px] text-gray-500 mt-1">{t.totalScenes || '总场景数'}</div>
    </div>
    
    <div class="bg-white rounded-xl border border-gray-100 p-5">
      <div class="flex items-center justify-between mb-3">
        <div class="w-10 h-10 rounded-lg bg-emerald-50 flex items-center justify-center">
          <svg class="w-5 h-5 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
        </div>
      </div>
      <div class="text-[28px] font-bold text-emerald-600">{stats.runningCases}</div>
      <div class="text-[13px] text-gray-500 mt-1">{t.runningScenes || '运行中'}</div>
    </div>
    
    <div class="bg-white rounded-xl border border-gray-100 p-5">
      <div class="flex items-center justify-between mb-3">
        <div class="w-10 h-10 rounded-lg bg-slate-50 flex items-center justify-center">
          <svg class="w-5 h-5 text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
      </div>
      <div class="text-[28px] font-bold text-slate-600">{stats.stoppedCases}</div>
      <div class="text-[13px] text-gray-500 mt-1">{t.stoppedScenes || '已停止'}</div>
    </div>
    
    <div class="bg-white rounded-xl border border-gray-100 p-5">
      <div class="flex items-center justify-between mb-3">
        <div class="w-10 h-10 rounded-lg bg-red-50 flex items-center justify-center">
          <svg class="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
      </div>
      <div class="text-[28px] font-bold text-red-600">{stats.errorCases}</div>
      <div class="text-[13px] text-gray-500 mt-1">{t.errorScenes || '异常'}</div>
    </div>
  </div>
  
  <!-- Quick Stats -->
  <div class="grid grid-cols-4 gap-3">
    {#each quickStats as stat}
      <div class="bg-white rounded-xl border border-gray-100 p-4">
        <div class="text-[12px] text-gray-500 mb-2">{stat.label}</div>
        <div class="flex items-end justify-between">
          <div class="text-[20px] font-bold text-gray-900">{stat.value}</div>
          {#if stat.trend !== 'neutral'}
            <div class="text-[11px] font-medium {stat.trend === 'up' ? 'text-emerald-600' : 'text-red-600'}">
              {stat.change}
            </div>
          {/if}
        </div>
      </div>
    {/each}
  </div>
  
  <!-- Main Content Grid -->
  <div class="grid grid-cols-3 gap-4">
    <!-- Recent Cases -->
    <div class="col-span-2 bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[15px] font-semibold text-gray-900">{t.recentScenes || '最近场景'}</h3>
        <div class="flex items-center gap-2">
          <button
            class="inline-flex items-center gap-1 h-7 px-2.5 text-white bg-blue-600 hover:bg-blue-700 text-[11px] font-medium rounded-lg transition-colors"
            onclick={navigateToCreate}
          >
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
            {t.quickCreate || '快速创建'}
          </button>
          <button 
            class="text-[12px] text-blue-600 hover:text-blue-700 font-medium"
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
          <div class="px-5 py-12 text-center">
            <div class="text-[13px] text-gray-400 mb-3">{t.noRecentScenes || '暂无场景'}</div>
            <button
              class="inline-flex items-center gap-1.5 px-3 py-1.5 text-[12px] text-blue-600 bg-blue-50 hover:bg-blue-100 rounded-lg transition-colors font-medium"
              onclick={navigateToCreate}
            >
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
              {t.createFirstScene || '创建第一个场景'}
            </button>
          </div>
        {:else}
          {#each recentCases as c}
            <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
            <div class="px-5 py-3 hover:bg-gray-50 transition-colors cursor-pointer group" onclick={navigateToCases}>
              <div class="flex items-center justify-between">
                <div class="flex-1 min-w-0">
                  <div class="text-[13px] font-medium text-gray-900 truncate">{c.name}</div>
                  <div class="text-[11px] text-gray-500 mt-0.5">{c.type} · {c.stateTime}</div>
                </div>
                <div class="flex items-center gap-2">
                  <!-- Quick action buttons (visible on hover) -->
                  <div class="hidden group-hover:flex items-center gap-1">
                    {#if c.state === 'running'}
                      <button
                        class="h-6 px-2 text-[10px] font-medium text-red-600 bg-red-50 hover:bg-red-100 rounded transition-colors disabled:opacity-50"
                        onclick={(e) => handleStopCase(e, c.id)}
                        disabled={actionLoading[c.id]}
                        title={t.stop || '停止'}
                      >
                        {actionLoading[c.id] === 'stop' ? '...' : (t.stop || '停止')}
                      </button>
                      <button
                        class="h-6 px-2 text-[10px] font-medium text-blue-600 bg-blue-50 hover:bg-blue-100 rounded transition-colors"
                        onclick={(e) => handleSSH(e, c)}
                        title="SSH"
                      >
                        SSH
                      </button>
                    {:else if c.state === 'stopped' || c.state === 'created'}
                      <button
                        class="h-6 px-2 text-[10px] font-medium text-emerald-600 bg-emerald-50 hover:bg-emerald-100 rounded transition-colors disabled:opacity-50"
                        onclick={(e) => handleStartCase(e, c.id)}
                        disabled={actionLoading[c.id]}
                        title={t.start || '启动'}
                      >
                        {actionLoading[c.id] === 'start' ? '...' : (t.start || '启动')}
                      </button>
                    {/if}
                  </div>
                  <span class="inline-flex items-center gap-1.5 px-2.5 py-1 text-[11px] font-medium rounded-full {getStateColor(c.state)}">
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
    
    <!-- Network Diagnostics -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[15px] font-semibold text-gray-900">{t.networkCheck || '网络诊断'}</h3>
        <button
          class="h-7 px-2.5 text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 text-[10px] font-medium rounded-lg transition-colors disabled:opacity-50"
          onclick={runNetworkCheck}
          disabled={networkCheckLoading}
        >
          {networkCheckLoading ? t.networkChecking : t.retest}
        </button>
      </div>
      <div class="p-4">
        {#if networkCheckLoading}
          <div class="text-center py-8 text-[13px] text-gray-400">
            {t.loading || '加载中...'}
          </div>
        {:else if networkChecks.length === 0}
          <div class="text-center py-8 text-[13px] text-gray-400">
            {t.noNetworkData || '暂无网络诊断数据'}
          </div>
        {:else}
          <div class="space-y-2.5">
            {#each networkChecks as item}
              <div class="flex items-center justify-between py-2">
                <div class="flex items-center gap-2 flex-1 min-w-0">
                  <div class="w-2 h-2 rounded-full flex-shrink-0 {item.ok ? 'bg-emerald-500' : 'bg-red-500'}"></div>
                  <span class="text-[11px] text-gray-700 truncate">{item.name}</span>
                </div>
                <div class="flex items-center gap-2 flex-shrink-0">
                  <span class="text-[10px] text-gray-500">{item.latencyMs}ms</span>
                  <span class="text-[10px] font-medium {item.ok ? 'text-emerald-600' : 'text-red-600'} w-10 text-right">
                    {item.ok ? 'OK' : 'FAIL'}
                  </span>
                </div>
              </div>
            {/each}
          </div>
          {#if networkChecks.some(item => !item.ok)}
            <div class="mt-3 pt-3 border-t border-gray-100">
              <button
                class="text-[11px] text-blue-600 hover:text-blue-700 font-medium"
                onclick={navigateToCredentials}
              >
                {t.goToCredentials || '前往凭据管理'} →
              </button>
            </div>
          {/if}
        {/if}
      </div>
    </div>
  </div>
  
  <!-- Resource Summary & Balances & Bill -->
  <div class="grid grid-cols-3 gap-4">
    <!-- Resource Summary -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-4 border-b border-gray-100">
        <h3 class="text-[15px] font-semibold text-gray-900">{t.resourceSummary || '资源概览'}</h3>
      </div>
      <div class="p-5">
        {#if loading}
          <div class="text-center py-8 text-[13px] text-gray-400">
            {t.loading || '加载中...'}
          </div>
        {:else if resourceSummary.length === 0}
          <div class="text-center py-8 text-[13px] text-gray-400">
            {t.noResources || '暂无资源'}
          </div>
        {:else}
          <div class="space-y-3">
            {#each resourceSummary.slice(0, 6) as resource}
              <div class="flex items-center justify-between">
                <span class="text-[12px] text-gray-600">{resource.type}</span>
                <span class="text-[13px] font-medium text-gray-900">{resource.count}</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
    
    <!-- Account Balances -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[15px] font-semibold text-gray-900">{t.accountBalance || '账户余额'}</h3>
        <button 
          onclick={queryBalances}
          disabled={balancesLoading}
          class="h-7 px-2.5 text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 text-[10px] font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {balancesLoading ? (t.loading || '加载中...') : (t.queryBalance || '查询')}
        </button>
      </div>
      <div class="p-5">
        {#if balancesLoading}
          <div class="text-center py-8 text-[13px] text-gray-400">
            {t.loading || '加载中...'}
          </div>
        {:else if balances.length === 0}
          <div class="text-center py-4 text-[13px] text-gray-400">
            <div>{t.clickToQueryBalance || '点击上方按钮查询账户余额'}</div>
          </div>
        {:else}
          <div class="space-y-3">
            {#each balances as balance}
              <div class="flex items-center justify-between">
                <span class="text-[12px] text-gray-600">{balance.provider}</span>
                {#if balance.error}
                  <span class="text-[11px] text-gray-500">{translateError(balance.error)}</span>
                {:else}
                  <span class="text-[13px] font-medium text-gray-900">{balance.currency} {balance.amount}</span>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
    
    <!-- Current Month Bill -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[15px] font-semibold text-gray-900">{t.currentMonthBill || '当月账单'}</h3>
        <button 
          onclick={queryBills}
          disabled={billsLoading}
          class="h-7 px-2.5 text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 text-[10px] font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {billsLoading ? (t.loading || '加载中...') : (t.queryBill || '查询')}
        </button>
      </div>
      <div class="p-5">
        {#if billsLoading}
          <div class="text-center py-8 text-[13px] text-gray-400">
            {t.loading || '加载中...'}
          </div>
        {:else if bills.length === 0}
          <div class="text-center py-4 text-[13px] text-gray-400">
            <div>{t.clickToQueryBill || '点击上方按钮查询当月账单'}</div>
            <div class="mt-2 text-[11px] text-amber-500">{t.billCostWarning || 'AWS Cost Explorer USE1-APIRequest 调用每次约 $0.01'}</div>
          </div>
        {:else}
          <div class="space-y-3">
            {#each bills as bill}
              <div class="flex items-center justify-between pb-2 border-b border-gray-50 last:border-0">
                <span class="text-[12px] text-gray-500 uppercase">{bill.provider}</span>
                {#if bill.error}
                  <span class="text-[11px] text-gray-500">{translateError(bill.error)}</span>
                {:else}
                  <span class="text-[14px] font-medium text-gray-900">{bill.currency} {bill.totalAmount}</span>
                {/if}
              </div>
            {/each}
            {#if bills.length > 0 && bills[0].startDate}
              <div class="pt-2 mt-2 border-t border-gray-100">
                <span class="text-[10px] text-gray-400">{bills[0].startDate} ~ {bills[0].endDate}</span>
              </div>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>
