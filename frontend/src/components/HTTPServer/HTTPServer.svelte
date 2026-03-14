<script>
  import { GetHTTPServerConfig, SetHTTPServerConfig, StartHTTPServer, StopHTTPServer, GetHTTPServerStatus, GetHTTPServerUsers, AddHTTPServerUser, RemoveHTTPServerUser, UpdateHTTPServerUser } from '../../../wailsjs/go/main/App.js';

  let { t = {} } = $props();

  let httpForm = $state({ enabled: false, port: 8899, host: '127.0.0.1', token: '' });
  let httpStatus = $state({ running: false, url: '', token: '' });
  let httpSaving = $state(false);
  let httpMessage = $state('');
  let httpMessageType = $state('');
  let httpLoaded = $state(false);

  // User management
  let users = $state([]);
  let showAddUser = $state(false);
  let newUsername = $state('');
  let newRole = $state('viewer');
  let addingUser = $state(false);
  let confirmDelete = $state('');

  const roleLabels = { admin: 'Admin', operator: 'Operator', viewer: 'Viewer' };
  const roleColors = {
    admin: 'bg-red-50 text-red-700 border-red-100',
    operator: 'bg-amber-50 text-amber-700 border-amber-100',
    viewer: 'bg-gray-50 text-gray-600 border-gray-100'
  };
  const roleDescs = {
    admin: t.roleDescAdmin || '完全控制：创建、销毁、配置、用户管理',
    operator: t.roleDescOperator || '操作权限：创建、启停、SSH、部署（不可销毁/配置）',
    viewer: t.roleDescViewer || '只读权限：查看场景、资源、日志'
  };

  async function loadConfig() {
    if (httpLoaded) return;
    try {
      const [cfg, status, userList] = await Promise.all([
        GetHTTPServerConfig(),
        GetHTTPServerStatus(),
        GetHTTPServerUsers(),
      ]);
      httpForm = { enabled: cfg.enabled || false, port: cfg.port || 8899, host: cfg.host || '127.0.0.1', token: cfg.token || '' };
      httpStatus = { running: status.running || false, url: status.url || '', token: status.token || '' };
      users = userList || [];
      httpLoaded = true;
    } catch(e) {
      console.error('Failed to load HTTP server config:', e);
    }
  }

  async function handleStart() {
    httpMessage = '';
    httpSaving = true;
    try {
      await StartHTTPServer(httpForm.port, httpForm.host, httpForm.token);
      const status = await GetHTTPServerStatus();
      httpStatus = { running: status.running || false, url: status.url || '', token: status.token || '' };
      httpMessage = t.httpServerStartSuccess || 'HTTP Server started';
      httpMessageType = 'success';
    } catch(e) {
      httpMessage = (t.httpServerStartFailed || 'Start failed') + ': ' + (e.message || String(e));
      httpMessageType = 'error';
    } finally {
      httpSaving = false;
      setTimeout(() => { httpMessage = ''; }, 4000);
    }
  }

  async function handleStop() {
    httpMessage = '';
    httpSaving = true;
    try {
      await StopHTTPServer();
      httpStatus = { running: false, url: '', token: '' };
      httpMessage = t.httpServerStopSuccess || 'HTTP Server stopped';
      httpMessageType = 'success';
    } catch(e) {
      httpMessage = (t.httpServerStopFailed || 'Stop failed') + ': ' + (e.message || String(e));
      httpMessageType = 'error';
    } finally {
      httpSaving = false;
      setTimeout(() => { httpMessage = ''; }, 3000);
    }
  }

  async function handleSave() {
    httpMessage = '';
    httpSaving = true;
    try {
      await SetHTTPServerConfig(httpForm.enabled, httpForm.port, httpForm.host, httpForm.token);
      httpMessage = t.httpServerSaveSuccess || 'Config saved';
      httpMessageType = 'success';
    } catch(e) {
      httpMessage = String(e.message || e);
      httpMessageType = 'error';
    } finally {
      httpSaving = false;
      setTimeout(() => { httpMessage = ''; }, 3000);
    }
  }

  async function handleAddUser() {
    if (!newUsername.trim()) return;
    addingUser = true;
    try {
      const user = await AddHTTPServerUser(newUsername.trim(), newRole);
      users = [...users, user];
      newUsername = '';
      newRole = 'viewer';
      showAddUser = false;
    } catch(e) {
      httpMessage = String(e.message || e);
      httpMessageType = 'error';
      setTimeout(() => { httpMessage = ''; }, 3000);
    } finally {
      addingUser = false;
    }
  }

  async function handleRemoveUser(username) {
    try {
      await RemoveHTTPServerUser(username);
      users = users.filter(u => u.username !== username);
      confirmDelete = '';
    } catch(e) {
      httpMessage = String(e.message || e);
      httpMessageType = 'error';
      setTimeout(() => { httpMessage = ''; }, 3000);
    }
  }

  async function handleUpdateRole(username, role) {
    try {
      const updated = await UpdateHTTPServerUser(username, role, false);
      users = users.map(u => u.username === username ? updated : u);
    } catch(e) {
      httpMessage = String(e.message || e);
      httpMessageType = 'error';
      setTimeout(() => { httpMessage = ''; }, 3000);
    }
  }

  async function handleRegenerateToken(username) {
    try {
      const u = users.find(x => x.username === username);
      if (!u) return;
      const updated = await UpdateHTTPServerUser(username, u.role, true);
      users = users.map(x => x.username === username ? updated : x);
    } catch(e) {
      httpMessage = String(e.message || e);
      httpMessageType = 'error';
      setTimeout(() => { httpMessage = ''; }, 3000);
    }
  }

  function copyToClipboard(text) {
    navigator.clipboard.writeText(text).catch(() => {});
  }

  $effect(() => { loadConfig(); });
</script>

<div class="space-y-4">
  <!-- Status Banner -->
  {#if httpStatus.running}
  <div class="bg-emerald-50 border border-emerald-100 rounded-xl px-5 py-3 flex items-center justify-between flex-wrap gap-2">
    <div class="flex items-center gap-2.5">
      <span class="inline-block w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
      <span class="text-[13px] text-emerald-700 font-semibold">{t.httpServerRunning || 'Running'}</span>
      <span class="text-[13px] text-emerald-600 font-mono">{httpStatus.url}</span>
    </div>
    <div class="flex items-center gap-2">
      <button class="h-7 px-3 text-[11px] font-medium rounded-lg bg-emerald-100 hover:bg-emerald-200 text-emerald-700 cursor-pointer transition-colors" onclick={() => copyToClipboard(httpStatus.url)}>
        <span class="flex items-center gap-1">
          <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0 0 13.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 0 1-.75.75H9.75a.75.75 0 0 1-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 0 1-2.25 2.25H6.75A2.25 2.25 0 0 1 4.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 0 1 1.927-.184" /></svg>
          {t.httpServerCopyUrl || 'Copy URL'}
        </span>
      </button>
      <button class="h-7 px-3 text-[11px] font-medium rounded-lg bg-emerald-100 hover:bg-emerald-200 text-emerald-700 cursor-pointer transition-colors" onclick={() => copyToClipboard(httpStatus.token)}>
        <span class="flex items-center gap-1">
          <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 0 1 3 3m3 0a6 6 0 0 1-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1 1 21.75 8.25z" /></svg>
          {t.httpServerCopyToken || 'Copy Token'}
        </span>
      </button>
    </div>
  </div>
  {:else}
  <div class="bg-gray-50 border border-gray-100 rounded-xl px-5 py-3 flex items-center gap-2.5">
    <span class="inline-block w-2.5 h-2.5 rounded-full bg-gray-300"></span>
    <span class="text-[13px] text-gray-500">{t.httpServerStopped || '服务未运行'}</span>
  </div>
  {/if}

  <!-- Config Card -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <div class="px-5 py-3 border-b border-gray-100">
      <h3 class="text-[13px] font-semibold text-gray-900">{t.httpServerConfig || '服务配置'}</h3>
      <p class="text-[11px] text-gray-500 mt-0.5">{t.httpServerDesc || '通过浏览器访问 RedC GUI（无需桌面应用）'}</p>
    </div>
    <div class="px-5 py-4 space-y-3">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label class="block text-[11px] font-medium text-gray-500 mb-1.5">{t.httpServerHost || '监听地址'}</label>
          <input type="text" bind:value={httpForm.host} placeholder="127.0.0.1"
            class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono" />
        </div>
        <div>
          <label class="block text-[11px] font-medium text-gray-500 mb-1.5">{t.httpServerPort || '端口'}</label>
          <input type="number" bind:value={httpForm.port} min="1024" max="65535" placeholder="8899"
            class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono" />
        </div>
      </div>
      <div>
        <label class="block text-[11px] font-medium text-gray-500 mb-1.5">{t.httpServerToken || 'Access Token'} <span class="text-gray-400 font-normal">({t.httpServerMasterToken || '主 Token · Admin 权限'})</span></label>
        <input type="text" bind:value={httpForm.token} placeholder={t.httpServerTokenHint || '留空自动生成'}
          class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono" />
      </div>

      {#if httpMessage}
      <p class="text-[12px] rounded-lg px-3 py-2 {httpMessageType === 'success' ? 'text-emerald-600 bg-emerald-50' : 'text-red-600 bg-red-50'}">{httpMessage}</p>
      {/if}

      <div class="flex gap-2 pt-1">
        <button onclick={handleSave} disabled={httpSaving}
          class="h-8 px-4 text-[12px] font-medium rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-700 transition-colors cursor-pointer disabled:opacity-50">{t.httpServerSaveConfig || '保存配置'}</button>
        {#if !httpStatus.running}
          <button onclick={handleStart} disabled={httpSaving}
            class="h-8 px-4 text-[12px] font-medium rounded-lg bg-gray-900 hover:bg-gray-800 text-white transition-colors cursor-pointer disabled:opacity-50 inline-flex items-center gap-1.5">
            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 0 1 0 1.971l-11.54 6.347a1.125 1.125 0 0 1-1.667-.985V5.653z" /></svg>
            {t.httpServerStart || '启动'}
          </button>
        {:else}
          <button onclick={handleStop} disabled={httpSaving}
            class="h-8 px-4 text-[12px] font-medium rounded-lg bg-red-500 hover:bg-red-600 text-white transition-colors cursor-pointer disabled:opacity-50 inline-flex items-center gap-1.5">
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 7.5A2.25 2.25 0 0 1 7.5 5.25h9a2.25 2.25 0 0 1 2.25 2.25v9a2.25 2.25 0 0 1-2.25 2.25h-9a2.25 2.25 0 0 1-2.25-2.25v-9z" /></svg>
            {t.httpServerStop || '停止'}
          </button>
        {/if}
      </div>
    </div>
  </div>

  <!-- User Management Card -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <div class="px-5 py-3 border-b border-gray-100 flex items-center justify-between">
      <div>
        <h3 class="text-[13px] font-semibold text-gray-900">{t.httpServerUsers || '用户权限管理'}</h3>
        <p class="text-[11px] text-gray-500 mt-0.5">{t.httpServerUsersDesc || '为不同用户分配 Admin / Operator / Viewer 角色'}</p>
      </div>
      <button onclick={() => { showAddUser = !showAddUser; }}
        class="h-7 px-3 text-[11px] font-medium rounded-lg bg-gray-900 hover:bg-gray-800 text-white cursor-pointer transition-colors inline-flex items-center gap-1">
        <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
        {t.httpServerAddUser || '添加用户'}
      </button>
    </div>

    <div class="px-5 py-4 space-y-3">
      <!-- Role Description -->
      <div class="grid grid-cols-3 gap-2">
        {#each ['admin', 'operator', 'viewer'] as role}
        <div class="rounded-lg border {roleColors[role]} px-3 py-2">
          <div class="text-[11px] font-semibold">{roleLabels[role]}</div>
          <div class="text-[10px] opacity-75 mt-0.5">{roleDescs[role]}</div>
        </div>
        {/each}
      </div>

      <!-- Add User Form -->
      {#if showAddUser}
      <div class="bg-gray-50 rounded-lg p-3 space-y-2">
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
          <div class="sm:col-span-1">
            <label class="block text-[10px] font-medium text-gray-500 mb-1">{t.httpServerUsername || '用户名'}</label>
            <input type="text" bind:value={newUsername} placeholder="e.g. operator1"
              class="w-full h-8 px-2.5 text-[12px] bg-white border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow" />
          </div>
          <div class="sm:col-span-1">
            <label class="block text-[10px] font-medium text-gray-500 mb-1">{t.httpServerRole || '角色'}</label>
            <select bind:value={newRole}
              class="w-full h-8 px-2.5 text-[12px] bg-white border-0 rounded-lg text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow cursor-pointer">
              <option value="viewer">Viewer</option>
              <option value="operator">Operator</option>
              <option value="admin">Admin</option>
            </select>
          </div>
          <div class="sm:col-span-1 flex items-end gap-2">
            <button onclick={handleAddUser} disabled={addingUser || !newUsername.trim()}
              class="h-8 px-3 text-[12px] font-medium rounded-lg bg-gray-900 hover:bg-gray-800 text-white cursor-pointer transition-colors disabled:opacity-50 inline-flex items-center gap-1">
              {#if addingUser}
                <svg class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
              {/if}
              {t.httpServerConfirmAdd || '确认添加'}
            </button>
            <button onclick={() => { showAddUser = false; newUsername = ''; }}
              class="h-8 px-3 text-[12px] font-medium rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600 cursor-pointer transition-colors">{t.cancel || '取消'}</button>
          </div>
        </div>
      </div>
      {/if}

      <!-- User List -->
      {#if users.length > 0}
      <div class="divide-y divide-gray-100">
        {#each users as user}
        <div class="py-2.5 first:pt-0 last:pb-0">
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center gap-2.5 min-w-0">
              <!-- Avatar -->
              <div class="w-7 h-7 rounded-full bg-gray-100 flex items-center justify-center flex-shrink-0">
                <svg class="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0" /></svg>
              </div>
              <div class="min-w-0">
                <div class="text-[13px] font-medium text-gray-900 truncate">{user.username}</div>
                <div class="text-[10px] text-gray-400 font-mono truncate" title={user.token}>{user.token?.slice(0, 12)}…</div>
              </div>
            </div>
            <div class="flex items-center gap-2 flex-shrink-0">
              <!-- Role Selector -->
              <select value={user.role} onchange={(e) => handleUpdateRole(user.username, e.target.value)}
                class="h-7 px-2 text-[11px] bg-gray-50 border-0 rounded-lg text-gray-700 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow cursor-pointer">
                <option value="viewer">Viewer</option>
                <option value="operator">Operator</option>
                <option value="admin">Admin</option>
              </select>
              <!-- Copy Token -->
              <button onclick={() => copyToClipboard(user.token)} title={t.httpServerCopyToken || 'Copy Token'}
                class="h-7 w-7 flex items-center justify-center rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 cursor-pointer transition-colors">
                <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0 0 13.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 0 1-.75.75H9.75a.75.75 0 0 1-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 0 1-2.25 2.25H6.75A2.25 2.25 0 0 1 4.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 0 1 1.927-.184" /></svg>
              </button>
              <!-- Regenerate Token -->
              <button onclick={() => handleRegenerateToken(user.username)} title={t.httpServerRegenToken || '重新生成 Token'}
                class="h-7 w-7 flex items-center justify-center rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 cursor-pointer transition-colors">
                <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182" /></svg>
              </button>
              <!-- Delete -->
              {#if confirmDelete === user.username}
              <button onclick={() => handleRemoveUser(user.username)}
                class="h-7 px-2 text-[11px] font-medium rounded-lg bg-red-500 hover:bg-red-600 text-white cursor-pointer transition-colors">{t.confirmDelete || '确认删除'}</button>
              <button onclick={() => { confirmDelete = ''; }}
                class="h-7 px-2 text-[11px] font-medium rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600 cursor-pointer transition-colors">{t.cancel || '取消'}</button>
              {:else}
              <button onclick={() => { confirmDelete = user.username; }} title={t.delete || '删除'}
                class="h-7 w-7 flex items-center justify-center rounded-lg hover:bg-red-50 text-gray-400 hover:text-red-500 cursor-pointer transition-colors">
                <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" /></svg>
              </button>
              {/if}
            </div>
          </div>
        </div>
        {/each}
      </div>
      {:else}
      <div class="text-center py-6">
        <svg class="w-8 h-8 mx-auto text-gray-200" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M15 19.128a9.38 9.38 0 0 0 2.625.372 9.337 9.337 0 0 0 4.121-.952 4.125 4.125 0 0 0-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 0 1 8.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0 1 11.964-3.07M12 6.375a3.375 3.375 0 1 1-6.75 0 3.375 3.375 0 0 1 6.75 0Zm8.25 2.25a2.625 2.625 0 1 1-5.25 0 2.625 2.625 0 0 1 5.25 0Z" /></svg>
        <p class="text-[12px] text-gray-400 mt-2">{t.httpServerNoUsers || '暂无用户，主 Token 默认拥有 Admin 权限'}</p>
        <p class="text-[11px] text-gray-400 mt-0.5">{t.httpServerNoUsersHint || '点击"添加用户"创建不同权限级别的访问账号'}</p>
      </div>
      {/if}
    </div>
  </div>

  <!-- Usage Guide -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <div class="px-5 py-3 border-b border-gray-100">
      <h3 class="text-[13px] font-semibold text-gray-900">{t.httpServerGuide || '使用说明'}</h3>
    </div>
    <div class="px-5 py-4 space-y-2 text-[12px] text-gray-600 leading-relaxed">
      <div class="flex items-start gap-2">
        <span class="text-gray-400 mt-0.5 flex-shrink-0">1.</span>
        <span>{t.httpServerGuide1 || '配置监听地址和端口，点击"启动"按钮'}</span>
      </div>
      <div class="flex items-start gap-2">
        <span class="text-gray-400 mt-0.5 flex-shrink-0">2.</span>
        <span>{t.httpServerGuide2 || '在浏览器中访问显示的 URL'}</span>
      </div>
      <div class="flex items-start gap-2">
        <span class="text-gray-400 mt-0.5 flex-shrink-0">3.</span>
        <span>{t.httpServerGuide3 || '使用 Access Token 进行认证登录'}</span>
      </div>
      <div class="flex items-start gap-2">
        <span class="text-gray-400 mt-0.5 flex-shrink-0">4.</span>
        <span>{t.httpServerGuide4 || '如需远程访问，将监听地址改为 0.0.0.0'}</span>
      </div>
      <div class="flex items-start gap-2">
        <span class="text-gray-400 mt-0.5 flex-shrink-0">5.</span>
        <span>{t.httpServerGuide5 || '添加用户并分配角色（Admin/Operator/Viewer），每个用户获得独立 Token'}</span>
      </div>
    </div>
  </div>
</div>
