<script>
  let token = localStorage.getItem('token')
  let user = null
  let planet = null
  let email = ''
  let password = ''
  let mode = 'login'
  let error = ''
  let upgrading = null

  async function handleSubmit() {
    error = ''
    const endpoint = mode === 'login' ? '/api/auth/login' : '/api/auth/register'
    try {
      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })
      const data = await res.json()
      if (!res.ok) { error = data.error || 'Request failed'; return }
      token = data.token
      user = data.user
      localStorage.setItem('token', token)
      loadPlanet()
    } catch (e) { error = e.message }
  }

  function logout() {
    token = null; user = null; planet = null
    localStorage.removeItem('token')
  }

  let pollInterval
  async function loadPlanet() {
    try {
      const res = await fetch('/api/planet/mine', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) throw new Error('Failed to load planet')
      planet = await res.json()
      error = ''
    } catch (e) { error = e.message }
  }

  function startPolling() {
    loadPlanet()
    pollInterval = setInterval(loadPlanet, 5000)
  }

  function stopPolling() {
    if (pollInterval) clearInterval(pollInterval)
  }

  function switchMode() {
    mode = mode === 'login' ? 'register' : 'login'
    error = ''
  }

  $: if (token && !user) startPolling()
  $: if (!token) stopPolling()

  function planetTypeLabel(type) {
    const labels = {
      terran: 'Terran', desert: 'Desert', ice: 'Ice',
      volcanic: 'Volcanic', gas_giant: 'Gas Giant',
    }
    return labels[type] || type
  }

  function planetTypeIcon(type) {
    const icons = {
      terran: '🌍', desert: '🏜️', ice: '❄️',
      volcanic: '🌋', gas_giant: '🪐',
    }
    return icons[type] || '🌍'
  }

  function buildingLabel(type) {
    const labels = {
      metal_mine: 'Metal Mine', crystal_mine: 'Crystal Mine',
      gas_mine: 'Gas Mine', solar_plant: 'Solar Plant',
      metal_storage: 'Metal Storage', crystal_storage: 'Crystal Storage',
      gas_storage: 'Gas Storage',
      robotics_factory: 'Robotics Facility', nanite_factory: 'Nanite Factory',
      terraformer: 'Terraformer',
      fusion_reactor: 'Fusion Reactor',
    }
    return labels[type] || type
  }

  function buildingCost(type, level) {
    const next = level + 1
    switch (type) {
      case 'metal_mine': return { metal: Math.floor(60 * Math.pow(1.5, next)), crystal: Math.floor(15 * Math.pow(1.5, next)), gas: 0 }
      case 'crystal_mine': return { metal: Math.floor(48 * Math.pow(1.6, next)), crystal: Math.floor(24 * Math.pow(1.6, next)), gas: 0 }
      case 'gas_mine': return { metal: Math.floor(225 * Math.pow(1.5, next)), crystal: Math.floor(75 * Math.pow(1.5, next)), gas: 0 }
      case 'solar_plant': return { metal: Math.floor(75 * Math.pow(1.5, next)), crystal: Math.floor(30 * Math.pow(1.5, next)), gas: 0 }
      case 'metal_storage': return { metal: Math.floor(1000 * Math.pow(2, next)), crystal: 0, gas: 0 }
      case 'crystal_storage': return { metal: Math.floor(1000 * Math.pow(2, next)), crystal: 0, gas: 0 }
      case 'gas_storage': return { metal: Math.floor(1000 * Math.pow(2, next)), crystal: 0, gas: 0 }
      case 'robotics_factory': return { metal: Math.floor(400 * Math.pow(2, next)), crystal: Math.floor(120 * Math.pow(2, next)), gas: Math.floor(200 * Math.pow(2, next)) }
      case 'nanite_factory': return { metal: Math.floor(1000000 * Math.pow(2, next)), crystal: Math.floor(500000 * Math.pow(2, next)), gas: Math.floor(100000 * Math.pow(2, next)) }
      case 'terraformer': return { metal: Math.floor(50000 * Math.pow(2, next)), crystal: Math.floor(50000 * Math.pow(2, next)), gas: Math.floor(50000 * Math.pow(2, next)) }
      case 'fusion_reactor': return { metal: Math.floor(200 * Math.pow(2, next)), crystal: Math.floor(150 * Math.pow(2, next)), gas: Math.floor(50 * Math.pow(2, next)) }
    }
    return { metal: 0, crystal: 0, gas: 0 }
  }

  function canAfford(type, level) {
    if (!planet) return false
    const cost = buildingCost(type, level)
    return planet.metal >= cost.metal && planet.crystal >= (cost.crystal || 0) && planet.gas >= (cost.gas || 0)
  }

  function isQueued(type) {
    return planet && planet.queue && planet.queue.some(q => q.building_type === type)
  }

  function toggleUpgrade(building) {
    if (isQueued(building.type)) return
    if (upgrading === building.type) { upgrading = null; return }
    upgrading = building.type
  }

  async function startUpgrade(type) {
    try {
      const res = await fetch(`/api/buildings/${type}/upgrade`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) {
        const data = await res.json()
        error = data.error || 'Upgrade failed'
        return
      }
      upgrading = null
      await loadPlanet()
    } catch (e) { error = e.message }
  }

  async function cancelUpgrade(buildingType) {
    try {
      const res = await fetch(`/api/buildings/${buildingType}/cancel`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) {
        const data = await res.json()
        error = data.error || 'Cancel failed'
        return
      }
      await loadPlanet()
    } catch (e) { error = e.message }
  }

  async function startDeconstruct(type) {
    try {
      const res = await fetch(`/api/buildings/${type}/deconstruct`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) {
        const data = await res.json()
        error = data.error || 'Deconstruct failed'
        return
      }
      await loadPlanet()
    } catch (e) { error = e.message }
  }
</script>

<div class="app">
  {#if token}
    <main class="dashboard">
      <header>
        <span class="user-email">{user ? user.email : ''}</span>
        <button class="logout" on:click={logout}>Logout</button>
      </header>

      {#if planet}
        <h1 class="name">{planet.name}</h1>
        <p class="coords">[{planet.galaxy}:{planet.system}:{planet.position}]</p>
        <p class="planet-meta">
          <span class="type-badge type-{planet.type}">{planetTypeIcon(planet.type)} {planetTypeLabel(planet.type)}</span>
          <span class="temperature">{planet.temperature}°C</span>
        </p>
        <p class="fields">Fields: {planet.fields_used}/{planet.max_fields}</p>

        <div class="resources">
          <div class="res metal">
            <span class="label">Metal</span>
            <span class="val">{planet.metal}</span>
            <span class="rate">+{planet.production.metal.toFixed(1)}/s</span>
            <div class="storage-bar">
              <div class="fill" style="width: {Math.min(100, planet.metal / planet.storage.metal * 100)}%"></div>
            </div>
            <span class="cap">{planet.metal}/{planet.storage.metal}</span>
          </div>
          <div class="res crystal">
            <span class="label">Crystal</span>
            <span class="val">{planet.crystal}</span>
            <span class="rate">+{planet.production.crystal.toFixed(1)}/s</span>
            <div class="storage-bar">
              <div class="fill" style="width: {Math.min(100, planet.crystal / planet.storage.crystal * 100)}%"></div>
            </div>
            <span class="cap">{planet.crystal}/{planet.storage.crystal}</span>
          </div>
          <div class="res gas">
            <span class="label">Gas</span>
            <span class="val">{planet.gas}</span>
            <span class="rate">+{planet.production.gas.toFixed(1)}/s</span>
            <div class="storage-bar">
              <div class="fill" style="width: {Math.min(100, planet.gas / planet.storage.gas * 100)}%"></div>
            </div>
            <span class="cap">{planet.gas}/{planet.storage.gas}</span>
          </div>
          <div class="res energy" class:negative={planet.energy < 0}>
            <span class="label">{planet.energy < 0 ? '⚡' : '⚡'} Energy</span>
            <span class="val" class:red={planet.energy < 0}>{planet.energy}</span>
            <span class="rate">+{planet.production.energy.toFixed(1)}/s</span>
          </div>
        </div>

        {#if planet.queue && planet.queue.length > 0}
          <div class="queue">
            <h2>Construction</h2>
            {#each planet.queue as entry}
              <div class="queue-item">
                <span class="qname">{buildingLabel(entry.building_type)}</span>
                <span class="qlevel">
                  {#if entry.status === 'deconstruct'}
                    Deconstruct
                  {:else}
                    Lv.{entry.target_level}
                  {/if}
                </span>
                <span class="qtime">{(new Date(entry.completes_at) - new Date()) / 1000 > 0 ? Math.ceil((new Date(entry.completes_at) - new Date()) / 1000) + 's' : 'Complete'}</span>
                <button class="btn-cancel-queue" on:click={() => cancelUpgrade(entry.building_type)}>Cancel</button>
              </div>
            {/each}
          </div>
        {/if}

        <div class="buildings">
          <h2>Buildings</h2>
          {#each planet.buildings as building}
            {@const cost = buildingCost(building.type, building.level)}
            <div class="building" class:queued={isQueued(building.type)}>
              <div class="b-info">
                <span class="bname">{buildingLabel(building.type)}</span>
                <span class="blevel">Lv.{building.level}</span>
              </div>
              <div class="b-actions">
                {#if upgrading === building.type}
                  <div class="cost-card">
                    <div class="cost-row">
                      <span class="cost-icon metal-icon">M</span>
                      <span class="cost-val" class:insufficient={planet.metal < cost.metal}>{cost.metal}</span>
                    </div>
                    {#if cost.crystal > 0}
                      <div class="cost-row">
                        <span class="cost-icon crystal-icon">C</span>
                        <span class="cost-val" class:insufficient={planet.crystal < cost.crystal}>{cost.crystal}</span>
                      </div>
                    {/if}
                    {#if cost.gas > 0}
                      <div class="cost-row">
                        <span class="cost-icon gas-icon">G</span>
                        <span class="cost-val" class:insufficient={planet.gas < cost.gas}>{cost.gas}</span>
                      </div>
                    {/if}
                    <div class="cost-actions">
                      <button class="btn-confirm" disabled={!canAfford(building.type, building.level)} on:click={() => startUpgrade(building.type)}>Upgrade</button>
                      <button class="btn-cancel" on:click={() => { upgrading = null }}>X</button>
                    </div>
                  </div>
                {:else if !isQueued(building.type)}
                  <button class="btn-upgrade" on:click={() => toggleUpgrade(building)}>+</button>
                  <button class="btn-deconstruct" on:click={() => startDeconstruct(building.type)}>−</button>
                {:else}
                  <span class="q-badge">Build</span>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {:else if error}
        <p class="error">{error}</p>
      {:else}
        <p class="loading">Loading planet...</p>
      {/if}
    </main>
  {:else}
    <main class="auth">
      <h1>Galaxy Empire</h1>
      <form on:submit|preventDefault={handleSubmit}>
        <input bind:value={email} type="email" placeholder="Email" required />
        <input bind:value={password} type="password" placeholder="Password" required minlength="6" />
        <button type="submit">{mode === 'login' ? 'Login' : 'Register'}</button>
      </form>
      {#if error}<p class="error">{error}</p>{/if}
      <button class="switch" on:click={switchMode}>
        {mode === 'login' ? 'Create Account' : 'Already have an account?'}
      </button>
    </main>
  {/if}
</div>

<style>
  :global(*) { margin: 0; padding: 0; box-sizing: border-box; }
  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #0a0e1a;
    color: #c8d6e5;
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
  }

  .app { width: 100%; max-width: 420px; padding: 1rem; }

  .auth {
    background: #141b2d; border: 1px solid #1e2a4a;
    border-radius: 12px; padding: 2rem; text-align: center;
  }
  .auth h1 { font-size: 1.5rem; color: #e8eef5; margin-bottom: 1.5rem; }

  form { display: flex; flex-direction: column; gap: 0.75rem; }
  input {
    padding: 0.75rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 6px; color: #c8d6e5; font-size: 0.875rem; outline: none;
  }
  input:focus { border-color: #3a6ab5; }
  input::placeholder { color: #3a5a8a; }
  button[type="submit"] {
    padding: 0.75rem; background: #2a4a8a; border: none;
    border-radius: 6px; color: #e8eef5; font-size: 0.875rem; font-weight: 600; cursor: pointer;
  }
  button[type="submit"]:hover { background: #3a5aaa; }
  .switch { margin-top: 0.75rem; background: none; border: none; color: #5a8ab5; cursor: pointer; font-size: 0.8rem; }
  .switch:hover { color: #7aaad5; }
  .error { color: #d47474; margin-top: 0.75rem; font-size: 0.875rem; }
  .loading { color: #5a7fb5; font-size: 0.875rem; }

  .dashboard { background: #141b2d; border: 1px solid #1e2a4a; border-radius: 12px; padding: 2rem; text-align: center; }
  header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
  .user-email { font-size: 0.8rem; color: #5a7fb5; }
  .logout {
    padding: 0.4rem 0.75rem; background: #2a1a1a; border: 1px solid #4a2020;
    border-radius: 4px; color: #d47474; cursor: pointer; font-size: 0.75rem;
  }
  .logout:hover { background: #3a2020; }
  .name { font-size: 1.5rem; color: #e8eef5; margin-bottom: 0.25rem; }
  .coords { font-family: monospace; font-size: 0.75rem; color: #3a5a8a; margin-bottom: 0.25rem; }
  .planet-meta {
    display: flex; justify-content: center; align-items: center; gap: 0.75rem;
    margin-bottom: 1.5rem; font-size: 0.85rem;
  }
  .type-badge {
    display: inline-flex; align-items: center; gap: 0.3rem;
    padding: 0.15rem 0.5rem; border-radius: 4px; font-size: 0.8rem;
    border: 1px solid;
  }
  .type-terran { background: #1a2a1a; border-color: #2a4a2a; color: #8ac88a; }
  .type-desert { background: #2a1a0a; border-color: #4a3a1a; color: #d4a574; }
  .type-ice { background: #0a1a2a; border-color: #1a3a5a; color: #74a8d4; }
  .type-volcanic { background: #2a0a0a; border-color: #5a1a1a; color: #d47474; }
  .type-gas_giant { background: #1a0a2a; border-color: #3a1a5a; color: #a874d4; }
  .temperature { font-family: monospace; color: #5a7a9a; font-size: 0.8rem; }
  .fields { font-size: 0.75rem; color: #6a8a6a; margin-bottom: 1.5rem; }

  .resources { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
  .res {
    background: #1a2340; border: 1px solid #243050; border-radius: 8px;
    padding: 0.75rem; display: flex; flex-direction: column; gap: 0.15rem;
  }
  .label { font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; }
  .metal .label { color: #d4a574; }
  .crystal .label { color: #74a8d4; }
  .gas .label { color: #74d4a8; }
  .energy .label { color: #d4d474; }
  .val { font-size: 1.25rem; font-weight: 600; }
  .rate { font-size: 0.7rem; color: #3a6a3a; }
  .energy .rate { color: #6a6a3a; }
  .energy.negative { border-color: #5a2020; }
  .energy.negative .label { color: #d47474; }
  .energy.negative .val { color: #d47474; }
  .energy.negative .rate { color: #5a3a3a; }
  .red { color: #d47474; }
  .cap { font-size: 0.65rem; color: #5a5a6a; }
  .storage-bar { height: 4px; background: #0a0e1a; border-radius: 2px; margin: 0.15rem 0; overflow: hidden; }
  .storage-bar .fill { height: 100%; background: #2a5a3a; border-radius: 2px; transition: width 0.5s; }
  .crystal .storage-bar .fill { background: #2a3a6a; }
  .gas .storage-bar .fill { background: #3a5a3a; }

  .queue { margin-top: 1.5rem; }
  .queue h2 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; }
  .queue-item {
    display: flex; justify-content: space-between; align-items: center;
    padding: 0.5rem 0.75rem; background: #1a2830; border: 1px solid #1a3a3a;
    border-radius: 6px; margin-bottom: 0.5rem;
  }
  .qname { font-size: 0.85rem; color: #74c8d4; }
  .qlevel { font-size: 0.8rem; color: #5a8a9a; }
  .qtime { font-size: 0.8rem; color: #5aaa5a; font-family: monospace; }
  .q-badge {
    font-size: 0.65rem; background: #1a3a3a; color: #5aaa5a;
    padding: 0.15rem 0.4rem; border-radius: 3px; text-transform: uppercase;
  }

  .buildings { margin-top: 1.5rem; }
  .buildings h2 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; }
  .building {
    display: flex; justify-content: space-between; align-items: center;
    padding: 0.5rem 0.75rem; background: #1a2340; border: 1px solid #243050; border-radius: 6px;
    margin-bottom: 0.5rem; position: relative;
  }
  .building.queued { opacity: 0.5; border-color: #1a3a3a; }
  .b-info { display: flex; flex-direction: column; align-items: flex-start; gap: 0.1rem; }
  .bname { font-size: 0.85rem; }
  .blevel { font-size: 0.75rem; color: #5a7a9a; }
  .b-actions { display: flex; align-items: center; gap: 0.5rem; }
  .btn-upgrade {
    width: 28px; height: 28px; border-radius: 50%;
    background: #2a4a3a; border: 1px solid #3a6a4a;
    color: #5aaa5a; font-size: 1rem; cursor: pointer; display: flex; align-items: center; justify-content: center;
  }
  .btn-upgrade:hover { background: #3a5a4a; }
  .cost-card {
    position: absolute; top: 100%; right: 0; z-index: 10;
    background: #1a2840; border: 1px solid #2a4a6a; border-radius: 8px;
    padding: 0.5rem 0.75rem; display: flex; flex-direction: column; gap: 0.25rem;
    min-width: 120px; box-shadow: 0 4px 12px rgba(0,0,0,0.4);
  }
  .cost-row { display: flex; align-items: center; gap: 0.35rem; }
  .cost-icon {
    display: inline-flex; align-items: center; justify-content: center;
    width: 16px; height: 16px; border-radius: 50%; font-size: 0.6rem; font-weight: 700;
  }
  .metal-icon { background: #5a3a1a; color: #d4a574; }
  .crystal-icon { background: #1a3a5a; color: #74a8d4; }
  .gas-icon { background: #1a4a3a; color: #74d4a8; }
  .cost-val { font-size: 0.8rem; color: #c8d6e5; }
  .cost-val.insufficient { color: #d47474; }
  .cost-actions { display: flex; gap: 0.35rem; margin-top: 0.35rem; }
  .btn-confirm {
    flex: 1; padding: 0.3rem 0.5rem; background: #2a5a3a; border: none;
    border-radius: 4px; color: #a8e8a8; font-size: 0.75rem; cursor: pointer;
  }
  .btn-confirm:hover:not(:disabled) { background: #3a6a4a; }
  .btn-confirm:disabled { opacity: 0.4; cursor: not-allowed; }
  .btn-cancel {
    padding: 0.3rem 0.5rem; background: #4a2020; border: none;
    border-radius: 4px; color: #d47474; font-size: 0.75rem; cursor: pointer;
  }
  .btn-cancel:hover { background: #5a3030; }
  .btn-cancel-queue {
    padding: 0.2rem 0.4rem; background: #4a2020; border: 1px solid #6a3030;
    border-radius: 4px; color: #d47474; font-size: 0.65rem; cursor: pointer; margin-left: 0.5rem;
  }
  .btn-cancel-queue:hover { background: #5a3030; }
  .btn-deconstruct {
    width: 28px; height: 28px; border-radius: 50%;
    background: #4a2a2a; border: 1px solid #6a3a3a;
    color: #d47474; font-size: 1rem; cursor: pointer; display: flex; align-items: center; justify-content: center;
  }
  .btn-deconstruct:hover { background: #5a3a3a; }
</style>
