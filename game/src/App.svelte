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

  function canAffordShip(ship) {
    if (!planet) return false
    const qty = buildQuantities[ship.type] || 1
    return planet.metal >= ship.metal * qty && planet.crystal >= ship.crystal * qty && planet.gas >= ship.gas * qty
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

  let galaxyView = null
  let galaxyPage = 1
  let galaxyData = null
  let positions = null

  async function loadGalaxies() {
    try {
      const res = await fetch('/api/galaxy', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) throw new Error('Failed to load galaxies')
      galaxyView = await res.json()
    } catch (e) { error = e.message }
  }

  async function loadSystems(galaxyID, page) {
    try {
      const res = await fetch(`/api/galaxy/systems/${galaxyID}?page=${page}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) throw new Error('Failed to load systems')
      return await res.json()
    } catch (e) { error = e.message; return null }
  }

  async function loadPositions(systemID) {
    try {
      const res = await fetch(`/api/galaxy/positions/${systemID}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) throw new Error('Failed to load positions')
      positions = await res.json()
    } catch (e) { error = e.message }
  }

  async function showGalaxyTab() {
    positions = null
    galaxyData = null
    galaxyPage = 1
    await loadGalaxies()
    if (galaxyView && galaxyView.length > 0) {
      galaxyData = await loadSystems(galaxyView[0].id, galaxyPage)
    }
  }

  async function selectGalaxy(id) {
    galaxyPage = 1
    galaxyData = await loadSystems(id, galaxyPage)
  }

  async function galaxyPageNext() {
    if (galaxyData && galaxyPage < galaxyData.total_pages) {
      galaxyPage++
      galaxyData = await loadSystems(selectedGalaxy, galaxyPage)
    }
  }

  async function galaxyPagePrev() {
    if (galaxyPage > 1) {
      galaxyPage--
      galaxyData = await loadSystems(selectedGalaxy, galaxyPage)
    }
  }

  async function selectSystem(systemID) {
    await loadPositions(systemID)
  }

  let selectedGalaxy = 1

  let shipyardData = null
  let buildQuantities = {}

  async function loadShipyard() {
    try {
      const res = await fetch('/api/shipyard', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) throw new Error('Failed to load shipyard')
      shipyardData = await res.json()
      shipyardData.ships.forEach(s => { buildQuantities[s.type] = buildQuantities[s.type] || 1 })
    } catch (e) { error = e.message }
  }

  async function buildShips(shipType) {
    const qty = buildQuantities[shipType] || 1
    try {
      const res = await fetch('/api/shipyard/build', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ ship_type: shipType, quantity: qty })
      })
      if (!res.ok) {
        const data = await res.json()
        error = data.error || 'Build failed'
        return
      }
      await loadShipyard()
      await loadPlanet()
    } catch (e) { error = e.message }
  }

  let fleetData = null
  let fleetShips = null

  let dispatchForm = {
    shipQuantities: {},
    targetGalaxy: '',
    targetSystem: '',
    targetPosition: '',
    mission: 'attack',
    speed: 100
  }

  async function loadFleet() {
    try {
      const res = await fetch('/api/fleets', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) throw new Error('Failed to load fleets')
      fleetData = await res.json()
      dispatchForm.shipQuantities = {}
      const shipRes = await fetch('/api/shipyard', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (shipRes.ok) {
        fleetShips = await shipRes.json()
        fleetShips.ships.forEach(s => { dispatchForm.shipQuantities[s.type] = 0 })
      }
    } catch (e) { error = e.message }
  }

  async function dispatchFleet() {
    const ships = {}
    Object.entries(dispatchForm.shipQuantities).forEach(([type, qty]) => {
      if (qty > 0) ships[type] = qty
    })
    if (Object.keys(ships).length === 0) { error = 'Select at least one ship'; return }
    try {
      const res = await fetch('/api/fleets/dispatch', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ships,
          target_galaxy: parseInt(dispatchForm.targetGalaxy),
          target_system: parseInt(dispatchForm.targetSystem),
          target_position: parseInt(dispatchForm.targetPosition),
          mission: dispatchForm.mission,
          speed: dispatchForm.speed
        })
      })
      if (!res.ok) {
        const data = await res.json()
        error = data.error || 'Dispatch failed'
        return
      }
      await loadFleet()
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

        <div class="player-stats">
          <span class="vip-badge">VIP {planet.vip_level}</span>
          <span class="rank-badge">Rank {planet.rank}</span>
        </div>

        <button class="galaxy-toggle" on:click={showGalaxyTab}>Galaxy</button>
        <button class="shipyard-toggle" on:click={loadShipyard}>Shipyard</button>
        <button class="fleet-toggle" on:click={loadFleet}>Fleet</button>

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

        {#if positions}
          <div class="galaxy-section">
            <button class="back-btn" on:click={() => { positions = null; galaxyData = null }}>← Back to Systems</button>
            <div class="position-grid">
              {#each positions.positions as pos}
                <div class="position-card" class:occupied={pos.state === 'occupied'}>
                  <span class="pos-num">#{pos.position}</span>
                  {#if pos.state === 'occupied'}
                    <span class="pos-name">{pos.planet_name}</span>
                    <span class="pos-player">Player {pos.player_id}</span>
                  {:else}
                    <span class="pos-empty">Empty</span>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {:else if galaxyData}
          <div class="galaxy-section">
            <h3>Galaxy Map</h3>
            <div class="galaxy-controls">
              <select bind:value={selectedGalaxy} on:change={(e) => selectGalaxy(parseInt(e.target.value))}>
                {#each galaxyView || [] as g}
                  <option value={g.id}>{g.name}</option>
                {/each}
              </select>
            </div>
            <div class="system-list">
              {#each galaxyData.systems as sys}
                <button class="system-row" on:click={() => selectSystem(sys.id)}>
                  <span class="sys-num">System {sys.system_num}</span>
                  <span class="sys-occ">{sys.occupied_count}/15 occupied</span>
                </button>
              {/each}
            </div>
            <div class="pagination">
              <button disabled={galaxyPage <= 1} on:click={galaxyPagePrev}>Prev</button>
              <span>Page {galaxyPage} of {galaxyData.total_pages}</span>
              <button disabled={galaxyPage >= galaxyData.total_pages} on:click={galaxyPageNext}>Next</button>
            </div>
          </div>
        {/if}

        {#if shipyardData}
          <div class="shipyard-section">
            <h3>Shipyard {shipyardData.shipyard_level > 0 ? `Lv.${shipyardData.shipyard_level}` : '(not built)'}</h3>
            <div class="ship-grid">
              {#each shipyardData.ships as ship}
                <div class="ship-card">
                  <div class="ship-header">
                    <span class="ship-name">{ship.name}</span>
                    <span class="ship-qty">Owned: {ship.quantity}</span>
                  </div>
                  <div class="ship-stats">
                    <span class="stat">⚡{ship.speed}</span>
                    <span class="stat">📦{ship.cargo}</span>
                    <span class="stat">⛽{ship.fuel}</span>
                  </div>
                  <div class="ship-cost">
                    <span class="cost metal">M {ship.metal}</span>
                    <span class="cost crystal">C {ship.crystal}</span>
                    {#if ship.gas > 0}
                      <span class="cost gas">G {ship.gas}</span>
                    {/if}
                  </div>
                  <div class="ship-build">
                    <input type="number" min="1" bind:value={buildQuantities[ship.type]} class="qty-input" />
                    <button class="btn-build" disabled={!canAffordShip(ship)} on:click={() => buildShips(ship.type)}>Build</button>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}
        {#if fleetData}
          <div class="fleet-section">
            <h3>My Fleets</h3>
            {#if fleetData.fleets && fleetData.fleets.length > 0}
              {#each fleetData.fleets as fleet}
                <div class="fleet-card">
                  <div class="fleet-header">
                    <span class="fleet-id">Fleet #{fleet.id}</span>
                    <span class="fleet-mission">{fleet.mission}</span>
                    <span class="fleet-status">{fleet.status}</span>
                  </div>
                  <div class="fleet-coords">
                    {fleet.origin_galaxy}:{fleet.origin_system}:{fleet.origin_position}
                    &rarr;
                    {fleet.target_galaxy}:{fleet.target_system}:{fleet.target_position}
                  </div>
                  <div class="fleet-ships">
                    {#each Object.entries(fleet.ships || {}) as [type, qty]}
                      <span class="fleet-ship">{type}: {qty}</span>
                    {/each}
                  </div>
                  {#if fleet.arrives_at}
                    <div class="fleet-arrival">Arrives: {new Date(fleet.arrives_at).toLocaleString()}</div>
                  {/if}
                </div>
              {/each}
            {:else}
              <p class="fleet-empty">No active fleets</p>
            {/if}

            <h3>Dispatch Fleet</h3>
            <div class="dispatch-form">
              <div class="form-row">
                <label>Origin</label>
                <input type="text" value="[{planet.galaxy}:{planet.system}:{planet.position}]" disabled />
              </div>
              <div class="form-row">
                <label>Target Galaxy</label>
                <input type="number" min="1" bind:value={dispatchForm.targetGalaxy} />
              </div>
              <div class="form-row">
                <label>Target System</label>
                <input type="number" min="1" bind:value={dispatchForm.targetSystem} />
              </div>
              <div class="form-row">
                <label>Target Position</label>
                <input type="number" min="1" max="15" bind:value={dispatchForm.targetPosition} />
              </div>
              <div class="form-row">
                <label>Mission</label>
                <select bind:value={dispatchForm.mission}>
                  <option value="attack">Attack</option>
                  <option value="transport">Transport</option>
                  <option value="deploy">Deploy</option>
                  <option value="espionage">Espionage</option>
                  <option value="colonize">Colonize</option>
                  <option value="expedition">Expedition</option>
                  <option value="recycle">Recycle</option>
                </select>
              </div>
              <div class="form-row">
                <label>Speed</label>
                <div class="speed-group">
                  <input type="range" min="10" max="100" step="10" bind:value={dispatchForm.speed} />
                  <span class="speed-val">{dispatchForm.speed}%</span>
                </div>
              </div>
              {#if fleetShips}
                <div class="form-ships">
                  <label>Ships</label>
                  {#each fleetShips.ships as ship}
                    <div class="dispatch-ship-row">
                      <span class="dship-name">{ship.name}</span>
                      <span class="dship-owned">Owned: {ship.quantity}</span>
                      <input type="number" min="0" max={ship.quantity} bind:value={dispatchForm.shipQuantities[ship.type]} class="dship-qty" />
                    </div>
                  {/each}
                </div>
              {/if}
              <button class="btn-dispatch" on:click={dispatchFleet}>Dispatch</button>
            </div>
          </div>
        {/if}
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

  .player-stats {
    display: flex; justify-content: center; align-items: center;
    gap: 0.75rem; margin-bottom: 1.5rem; font-size: 0.85rem;
  }
  .vip-badge {
    padding: 0.15rem 0.5rem; border-radius: 4px; font-size: 0.8rem;
    background: #2a1a4a; border: 1px solid #4a2a6a; color: #b074d4;
  }
  .rank-badge {
    padding: 0.15rem 0.5rem; border-radius: 4px; font-size: 0.8rem;
    background: #1a3a2a; border: 1px solid #2a5a3a; color: #74d4a8;
  }

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

  .galaxy-toggle {
    display: block; margin: 1rem auto; padding: 0.5rem 1rem;
    background: #1a2a4a; border: 1px solid #2a4a6a; border-radius: 6px;
    color: #8ab5d4; font-size: 0.85rem; cursor: pointer;
  }
  .galaxy-toggle:hover { background: #2a3a5a; }

  .galaxy-section { margin-top: 1.5rem; text-align: left; }
  .galaxy-section h3 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-align: center; }
  .back-btn {
    padding: 0.3rem 0.6rem; background: #1a2a3a; border: 1px solid #2a4a4a;
    border-radius: 4px; color: #74a8c8; font-size: 0.75rem; cursor: pointer; margin-bottom: 0.5rem;
  }
  .galaxy-controls { text-align: center; margin-bottom: 0.75rem; }
  .galaxy-controls select {
    padding: 0.4rem 0.6rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 4px; color: #c8d6e5; font-size: 0.85rem;
  }
  .system-list { display: flex; flex-direction: column; gap: 0.35rem; }
  .system-row {
    display: flex; justify-content: space-between; align-items: center;
    padding: 0.5rem 0.75rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 6px; color: #c8d6e5; cursor: pointer; font-size: 0.85rem;
  }
  .system-row:hover { background: #1e2a4a; }
  .sys-occ { font-size: 0.75rem; color: #5a7a9a; }
  .pagination {
    display: flex; justify-content: center; align-items: center; gap: 0.75rem;
    margin-top: 0.75rem;
  }
  .pagination button {
    padding: 0.3rem 0.6rem; background: #1a2a4a; border: 1px solid #2a4a6a;
    border-radius: 4px; color: #8ab5d4; cursor: pointer; font-size: 0.75rem;
  }
  .pagination button:disabled { opacity: 0.4; cursor: not-allowed; }
  .pagination span { font-size: 0.75rem; color: #5a7a9a; }
  .position-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.5rem; }
  .position-card {
    padding: 0.5rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 6px; text-align: center; font-size: 0.8rem;
  }
  .position-card.occupied { border-color: #3a6a4a; background: #1a2a1a; }
  .pos-num { display: block; font-weight: 600; color: #5a7a9a; font-size: 0.75rem; margin-bottom: 0.25rem; }
  .pos-name { display: block; color: #8ac88a; }
  .pos-player { display: block; font-size: 0.65rem; color: #5a7a9a; }
  .pos-empty { color: #5a5a6a; font-style: italic; }

  .shipyard-toggle {
    display: block; margin: 1rem auto; padding: 0.5rem 1rem;
    background: #2a3a1a; border: 1px solid #4a6a2a; border-radius: 6px;
    color: #8ad474; font-size: 0.85rem; cursor: pointer;
  }
  .shipyard-toggle:hover { background: #3a4a2a; }
  .shipyard-section { margin-top: 1.5rem; }
  .shipyard-section h3 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-align: center; }
  .ship-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem; }
  .ship-card {
    padding: 0.5rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 6px; font-size: 0.8rem;
  }
  .ship-header { display: flex; justify-content: space-between; margin-bottom: 0.3rem; }
  .ship-name { font-weight: 600; color: #c8d6e5; }
  .ship-qty { font-size: 0.7rem; color: #5a7a9a; }
  .ship-stats { display: flex; gap: 0.5rem; font-size: 0.7rem; color: #8ab5d4; margin-bottom: 0.3rem; }
  .ship-cost { display: flex; gap: 0.5rem; font-size: 0.7rem; margin-bottom: 0.3rem; }
  .cost.metal { color: #d4a574; }
  .cost.crystal { color: #74a8d4; }
  .cost.gas { color: #74d4a8; }
  .ship-build { display: flex; gap: 0.25rem; }
  .qty-input {
    width: 50px; padding: 0.2rem; background: #0a0e1a; border: 1px solid #243050;
    border-radius: 3px; color: #c8d6e5; font-size: 0.75rem; text-align: center;
  }
  .btn-build {
    flex: 1; padding: 0.25rem; background: #2a5a3a; border: none;
    border-radius: 3px; color: #a8e8a8; font-size: 0.7rem; cursor: pointer;
  }
  .btn-build:disabled { opacity: 0.4; cursor: not-allowed; }
  .btn-build:hover:not(:disabled) { background: #3a6a4a; }

  .fleet-toggle {
    display: block; margin: 1rem auto; padding: 0.5rem 1rem;
    background: #2a1a3a; border: 1px solid #4a2a6a; border-radius: 6px;
    color: #b074d4; font-size: 0.85rem; cursor: pointer;
  }
  .fleet-toggle:hover { background: #3a2a4a; }
  .fleet-section { margin-top: 1.5rem; text-align: left; }
  .fleet-section h3 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-align: center; }
  .fleet-card {
    padding: 0.5rem 0.75rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 6px; margin-bottom: 0.5rem; font-size: 0.8rem;
  }
  .fleet-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.3rem; }
  .fleet-id { font-weight: 600; color: #c8d6e5; }
  .fleet-mission {
    font-size: 0.7rem; padding: 0.1rem 0.4rem; border-radius: 3px;
    background: #1a2a3a; border: 1px solid #2a4a5a; color: #74a8c8;
  }
  .fleet-status {
    font-size: 0.7rem; padding: 0.1rem 0.4rem; border-radius: 3px;
    background: #1a3a2a; border: 1px solid #2a5a3a; color: #74d4a8;
  }
  .fleet-coords { font-family: monospace; font-size: 0.75rem; color: #5a7a9a; margin-bottom: 0.3rem; }
  .fleet-ships { display: flex; flex-wrap: wrap; gap: 0.3rem; margin-bottom: 0.3rem; }
  .fleet-ship {
    font-size: 0.7rem; padding: 0.1rem 0.3rem; background: #0a0e1a;
    border-radius: 3px; color: #8ab5d4;
  }
  .fleet-arrival { font-size: 0.7rem; color: #5aaa5a; }
  .fleet-empty { font-size: 0.8rem; color: #5a5a6a; text-align: center; padding: 1rem; }
  .dispatch-form { display: flex; flex-direction: column; gap: 0.5rem; }
  .form-row { display: flex; align-items: center; gap: 0.5rem; }
  .form-row label { width: 100px; font-size: 0.75rem; color: #8a9ab5; text-align: right; flex-shrink: 0; }
  .form-row input, .form-row select {
    flex: 1; padding: 0.4rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 4px; color: #c8d6e5; font-size: 0.8rem;
  }
  .form-row input:disabled { opacity: 0.5; }
  .speed-group { flex: 1; display: flex; align-items: center; gap: 0.5rem; }
  .speed-group input[type="range"] { flex: 1; }
  .speed-val { font-size: 0.8rem; color: #8ab5d4; min-width: 40px; }
  .form-ships { margin-top: 0.5rem; }
  .form-ships > label { display: block; font-size: 0.75rem; color: #8a9ab5; margin-bottom: 0.3rem; }
  .dispatch-ship-row {
    display: flex; align-items: center; gap: 0.5rem;
    padding: 0.3rem 0; border-bottom: 1px solid #1a2340;
  }
  .dship-name { flex: 1; font-size: 0.8rem; color: #c8d6e5; }
  .dship-owned { font-size: 0.7rem; color: #5a7a9a; }
  .dship-qty {
    width: 60px; padding: 0.2rem; background: #0a0e1a; border: 1px solid #243050;
    border-radius: 3px; color: #c8d6e5; font-size: 0.75rem; text-align: center;
  }
  .btn-dispatch {
    padding: 0.5rem; background: #4a2a6a; border: none;
    border-radius: 6px; color: #c8a8e8; font-size: 0.85rem; font-weight: 600; cursor: pointer;
  }
  .btn-dispatch:hover { background: #5a3a7a; }
</style>
