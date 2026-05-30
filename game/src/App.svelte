<script>
  let token = localStorage.getItem('token')
  let user = null
  let planet = null
  let email = ''
  let password = ''
  let mode = 'login'
  let error = ''

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

  function buildingLabel(type) {
    const labels = {
      metal_mine: 'Metal Mine', crystal_mine: 'Crystal Mine',
      gas_mine: 'Gas Mine', solar_plant: 'Solar Plant',
    }
    return labels[type] || type
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

        <div class="buildings">
          <h2>Buildings</h2>
          {#each planet.buildings as building}
            <div class="building">
              <span class="bname">{buildingLabel(building.type)}</span>
              <span class="blevel">Lv.{building.level}</span>
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
  .coords { font-family: monospace; font-size: 0.75rem; color: #3a5a8a; margin-bottom: 1.5rem; }

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

  .buildings { margin-top: 1.5rem; }
  .buildings h2 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; }
  .building {
    display: flex; justify-content: space-between; padding: 0.5rem 0.75rem;
    background: #1a2340; border: 1px solid #243050; border-radius: 6px;
    margin-bottom: 0.5rem;
  }
  .bname { font-size: 0.85rem; }
  .blevel { font-size: 0.85rem; color: #8a9ab5; }
</style>
