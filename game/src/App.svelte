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
      const credRes = await fetch('/api/nebula/credits-balance', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (credRes.ok) {
        const credData = await credRes.json()
        planet.credits = credData.balance || 0
      }
      error = ''
    } catch (e) { error = e.message }
  }

  function startPolling() {
    loadPlanet()
    loadNotificationUnread()
    loadDiscoverer()
    loadTutorial()
    loadGems()
    if (!notificationSocket) connectNotificationSSE()
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

  function temperatureGasHint(temp) {
    if (temp <= -60) return '❄️ Gas +2.5'
    if (temp <= -30) return '❄️ Gas +2.0'
    if (temp <= 0) return '❄️ Gas +1.5'
    return ''
  }

  function temperatureSolarHint(temp) {
    if (temp >= 80) return '☀️ Solar +3.0'
    if (temp >= 60) return '☀️ Solar +2.0'
    if (temp >= 40) return '☀️ Solar +1.0'
    return ''
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
      small_shield_dome: 'Small Shield Dome',
      large_shield_dome: 'Large Shield Dome',
      moon_base: 'Moon Base',
      pioneer_lab: 'Pioneer Lab',
      wormhole_generator: 'Wormhole Generator',
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
      case 'small_shield_dome': return { metal: 20000, crystal: 10000, gas: 0 }
      case 'large_shield_dome': return { metal: 100000, crystal: 50000, gas: 20000 }
      case 'moon_base': return { metal: Math.floor(20000 * Math.pow(2, next)), crystal: Math.floor(10000 * Math.pow(2, next)), gas: Math.floor(5000 * Math.pow(2, next)) }
      case 'pioneer_lab': return { metal: Math.floor(20000 * Math.pow(2, next)), crystal: Math.floor(40000 * Math.pow(2, next)), gas: Math.floor(20000 * Math.pow(2, next)) }
      case 'wormhole_generator': return { metal: Math.floor(1600000 * Math.pow(2, next)), crystal: Math.floor(3200000 * Math.pow(2, next)), gas: Math.floor(1600000 * Math.pow(2, next)) }
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
  let currentSystemNum = 0
  let fleetView = null
  let chatView = null
  let messagingView = null
  let unreadCount = 0
  let messagingTab = 'inbox'
  let messagingConv = []
  let composeMsg = { receiver_id: '', content: '' }
  let allianceView = null
  let allianceTab = 'info'
  let bulletins = []
  let bulletinForm = { title: '', content: '' }
  let buddyView = null
  let buddyTab = 'list'
  let addFriendId = ''
  let renameMode = false
  let renameName = ''
  let notificationPanel = false
  let notifications = []
  let notificationUnread = 0
  let notificationSocket = null
  let notificationTotal = 0

  async function loadAlliance() {
    try {
      const res = await fetch('/api/alliance/my', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) { allianceView = null; return }
      allianceView = await res.json()
      await loadBulletins()
    } catch (e) { allianceView = null }
  }

  async function leaveAlliance() {
    try {
      const res = await fetch('/api/alliance/leave', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) { const d = await res.json(); error = d.error; return }
      allianceView = null; bulletins = []
    } catch (e) { error = e.message }
  }

  async function transferFounder(targetId) {
    try {
      const res = await fetch('/api/alliance/transfer', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ target_player_id: parseInt(targetId) })
      })
      if (!res.ok) { const d = await res.json(); error = d.error; return }
      await loadAlliance()
    } catch (e) { error = e.message }
  }

  async function loadBulletins() {
    try {
      const res = await fetch('/api/alliance/bulletins', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) bulletins = await res.json()
    } catch (e) { /* ignore */ }
  }

  async function postBulletin() {
    if (!bulletinForm.title.trim() || !bulletinForm.content.trim()) return
    try {
      const res = await fetch('/api/alliance/bulletin', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(bulletinForm)
      })
      if (!res.ok) { const d = await res.json(); error = d.error; return }
      bulletinForm = { title: '', content: '' }
      await loadBulletins()
    } catch (e) { error = e.message }
  }

  async function deleteBulletin(id) {
    try {
      await fetch(`/api/alliance/bulletins/${id}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      await loadBulletins()
    } catch (e) { /* ignore */ }
  }

  let chatSocket = null
  let chatMessages = []
  let chatInput = ''
  let chatChannel = 'global'
  let chatConnected = false

  async function loadChat() {
    await loadChatMessages()
    if (!chatSocket) connectChatSSE()
  }

  async function loadChatMessages() {
    try {
      const res = await fetch(`/api/chat/messages?channel=${chatChannel}&limit=30`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        const data = await res.json()
        chatMessages = (data.messages || []).reverse()
      }
    } catch (e) { /* ignore */ }
  }

  function connectChatSSE() {
    if (chatSocket) chatSocket.close()
    const url = `/api/chat/stream?token=${token}`
    chatSocket = new EventSource(url)
    chatConnected = true
    chatSocket.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data)
        if (msg.channel === chatChannel || (chatChannel === 'alliance' && msg.channel === 'alliance')) {
          chatMessages = [...chatMessages, msg]
        }
      } catch (err) { /* ignore */ }
    }
    chatSocket.onerror = () => { chatConnected = false }
  }

  async function sendChat() {
    if (!chatInput.trim()) return
    try {
      const res = await fetch('/api/chat/send', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ channel: chatChannel, content: chatInput.trim() })
      })
      if (!res.ok) { const d = await res.json(); error = d.error; return }
      chatInput = ''
    } catch (e) { error = e.message }
  }

  async function loadInbox() {
    try {
      const res = await fetch('/api/chat/private/inbox?limit=20', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        const data = await res.json()
        messagingConv = data.messages || []
      }
    } catch (e) { /* ignore */ }
  }

  async function loadOutbox() {
    try {
      const res = await fetch('/api/chat/private/outbox?limit=20', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        const data = await res.json()
        messagingConv = data.messages || []
      }
    } catch (e) { /* ignore */ }
  }

  async function loadUnreadCount() {
    try {
      const res = await fetch('/api/chat/private/unread-count', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        const data = await res.json()
        unreadCount = data.unread_count || 0
      }
    } catch (e) { /* ignore */ }
  }

  async function sendPrivateMsg() {
    if (!composeMsg.content.trim() || !composeMsg.receiver_id) return
    try {
      const res = await fetch('/api/chat/private/send', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          receiver_id: parseInt(composeMsg.receiver_id),
          content: composeMsg.content.trim()
        })
      })
      if (!res.ok) { const d = await res.json(); error = d.error; return }
      composeMsg = { receiver_id: '', content: '' }
      messagingTab = 'inbox'
      await loadInbox()
    } catch (e) { error = e.message }
  }

  async function markMsgRead(id) {
    try {
      await fetch('/api/chat/private/read', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ message_id: id })
      })
      await loadUnreadCount()
    } catch (e) { /* ignore */ }
  }

  async function loadBuddyList() {
    try {
      const res = await fetch('/api/friend/list', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) buddyView = await res.json()
    } catch (e) { /* ignore */ }
  }

  async function addFriend() {
    if (!addFriendId) return
    try {
      const res = await fetch('/api/friend/add', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ friend_id: parseInt(addFriendId) })
      })
      if (!res.ok) { const d = await res.json(); error = d.error; return }
      addFriendId = ''
      await loadBuddyList()
    } catch (e) { error = e.message }
  }

  async function removeFriend(friendId) {
    try {
      await fetch('/api/friend/remove', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ friend_id: friendId })
      })
      await loadBuddyList()
    } catch (e) { /* ignore */ }
  }

  async function startRename() {
    if (!planet) return
    renameName = planet.name
    renameMode = true
  }

  async function doRename() {
    if (!planet || !renameName.trim()) return
    try {
      const res = await fetch(`/api/planet/${planet.id}/rename`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: renameName.trim() })
      })
      if (!res.ok) { const d = await res.json(); error = d.error; return }
      renameMode = false
      await loadPlanet()
    } catch (e) { error = e.message }
  }

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
    fleetView = null
    await loadGalaxies()
    if (galaxyView && galaxyView.length > 0) {
      galaxyData = await loadSystems(galaxyView[0].id, galaxyPage)
    }
  }

  async function selectGalaxy(id) {
    galaxyPage = 1
    fleetView = null
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

  async function loadFleetsForView() {
    try {
      const res = await fetch('/api/fleets', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) throw new Error('Failed to load fleets')
      const raw = await res.json()
      if (Array.isArray(raw)) {
        fleetView = raw
      } else if (raw && Array.isArray(raw.fleets)) {
        fleetView = raw.fleets
      } else {
        fleetView = []
      }
    } catch (e) { error = e.message; fleetView = [] }
  }

  async function selectSystem(systemID, systemNum) {
    currentSystemNum = systemNum
    await Promise.all([
      loadPositions(systemID),
      loadFleetsForView()
    ])
  }

  function getFleetIndicators(pos) {
    if (!fleetView || !Array.isArray(fleetView)) return { indicators: [], count: 0, tooltip: '' }

    const isPlayerPos = planet &&
      planet.galaxy === selectedGalaxy &&
      planet.system === currentSystemNum &&
      planet.position === pos.position

    const atThisPos = fleetView.filter(f =>
      f.target_galaxy === selectedGalaxy &&
      f.target_system === currentSystemNum &&
      f.target_position === pos.position
    )

    const indicators = []
    const lines = []
    let totalShips = 0

    for (const f of atThisPos) {
      const sc = Object.values(f.ships || {}).reduce((a, b) => a + b, 0)
      totalShips += sc
      if (f.status === 'stationed') {
        if (!indicators.includes('⚔️')) indicators.push('⚔️')
        lines.push(`${f.mission} fleet stationed — ${sc} ships`)
      } else if (f.status === 'returning') {
        if (!indicators.includes('↩')) indicators.push('↩')
        lines.push(`${f.mission} fleet returning — ETA ${f.arrives_at ? formatFleetETA(f.arrives_at) : 'now'}`)
      } else if (f.status === 'in_transit') {
        if (!indicators.includes('→')) indicators.push('→')
        lines.push(`${f.mission} fleet incoming — ${sc} ships, ETA ${f.arrives_at ? formatFleetETA(f.arrives_at) : 'now'}`)
      }
    }

    if (isPlayerPos && planet?.id) {
      for (const f of fleetView) {
        if (f.origin_planet_id === planet.id && f.status === 'in_transit') {
          if (!indicators.includes('←')) indicators.push('←')
          lines.push(`${f.mission} fleet outgoing to [${f.target_galaxy}:${f.target_system}:${f.target_position}]`)
        }
      }
    }

    return { indicators, count: atThisPos.length, tooltip: lines.join('\n') }
  }

  let selectedGalaxy = 1

  let shipyardData = null
  let buildQuantities = {}
  let lastBuildResult = null
  let defenseData = null
  let defenseBuildQuantities = {}
  let lastDefenseBuildResult = null
  let moonData = null
  let moonBuildQuantities = {}
  let wormholeLinkForm = { targetGalaxy: '', targetSystem: '', targetPosition: '' }
  let ironBehemothQty = 1
  let lastIronBehemothResult = null

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
      lastBuildResult = await res.json()
      await loadShipyard()
      await loadPlanet()
      setTimeout(() => { lastBuildResult = null }, 5000)
    } catch (e) { error = e.message }
  }

  async function loadDefense() {
    try {
      const res = await fetch('/api/defense', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) throw new Error('Failed to load defense')
      defenseData = await res.json()
      defenseData.defenses.forEach(d => { defenseBuildQuantities[d.type] = defenseBuildQuantities[d.type] || 1 })
    } catch (e) { error = e.message }
  }

  async function buildDefense(defenseType) {
    const qty = defenseBuildQuantities[defenseType] || 1
    try {
      const res = await fetch('/api/defense/build', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ defense_type: defenseType, quantity: qty })
      })
      if (!res.ok) {
        const data = await res.json()
        error = data.error || 'Build failed'
        return
      }
      lastDefenseBuildResult = await res.json()
      await loadDefense()
      await loadPlanet()
      setTimeout(() => { lastDefenseBuildResult = null }, 5000)
    } catch (e) { error = e.message }
  }

  function setQuantity(shipType, qty) {
    buildQuantities[shipType] = Math.max(1, qty)
    buildQuantities = buildQuantities
  }

  function maxAfford(ship) {
    if (!planet) return 1
    let max = 999999999
    if (ship.metal > 0) max = Math.min(max, Math.floor(planet.metal / ship.metal))
    if (ship.crystal > 0) max = Math.min(max, Math.floor(planet.crystal / ship.crystal))
    if (ship.gas > 0) max = Math.min(max, Math.floor(planet.gas / ship.gas))
    return max
  }

  function setDefenseQuantity(defenseType, qty) {
    defenseBuildQuantities[defenseType] = Math.max(1, qty)
    defenseBuildQuantities = defenseBuildQuantities
  }

  function maxAffordDefense(def) {
    if (!planet) return 1
    let max = 999999999
    if (def.metal > 0) max = Math.min(max, Math.floor(planet.metal / def.metal))
    if (def.crystal > 0) max = Math.min(max, Math.floor(planet.crystal / def.crystal))
    if (def.gas > 0) max = Math.min(max, Math.floor(planet.gas / def.gas))
    return max
  }

  function formatBuildTime(seconds) {
    if (!seconds || seconds <= 0) return ''
    if (seconds < 60) return `${Math.round(seconds)}s`
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${Math.round(seconds % 60)}s`
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    return `${h}h ${m}m`
  }

  async function loadMoonBuildings() {
    if (!planet) return
    try {
      const res = await fetch(`/api/moon/${planet.galaxy}/${planet.system}/${planet.position}/buildings`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) throw new Error('Failed to load moon buildings')
      moonData = await res.json()
    } catch (e) { error = e.message }
  }

  async function upgradeMoonBuilding(type) {
    if (!planet) return
    try {
      const res = await fetch(`/api/moon/${planet.galaxy}/${planet.system}/${planet.position}/buildings/${type}/upgrade`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) {
        const data = await res.json()
        error = data.error || 'Upgrade failed'
        return
      }
      await loadMoonBuildings()
      await loadPlanet()
    } catch (e) { error = e.message }
  }

  async function linkWormholes() {
    if (!planet) return
    try {
      const res = await fetch('/api/wormhole/link', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          source_galaxy: planet.galaxy,
          source_system: planet.system,
          source_position: planet.position,
          target_galaxy: parseInt(wormholeLinkForm.targetGalaxy),
          target_system: parseInt(wormholeLinkForm.targetSystem),
          target_position: parseInt(wormholeLinkForm.targetPosition)
        })
      })
      if (!res.ok) {
        const data = await res.json()
        error = data.error || 'Link failed'
        return
      }
      await loadMoonBuildings()
      error = ''
    } catch (e) { error = e.message }
  }

  async function buildIronBehemoth() {
    if (!planet) return
    try {
      const res = await fetch(`/api/moon/${planet.galaxy}/${planet.system}/${planet.position}/build-iron-behemoth`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ quantity: ironBehemothQty })
      })
      if (!res.ok) {
        const data = await res.json()
        error = data.error || 'Build failed'
        return
      }
      lastIronBehemothResult = await res.json()
      await loadMoonBuildings()
      await loadPlanet()
      setTimeout(() => { lastIronBehemothResult = null }, 5000)
    } catch (e) { error = e.message }
  }

  function formatFleetETA(arrivesAt) {
    const eta = new Date(arrivesAt)
    const now = new Date()
    const diff = eta - now
    if (diff <= 0) return 'Now'
    const h = Math.floor(diff / 3600000)
    const m = Math.floor((diff % 3600000) / 60000)
    const s = Math.floor((diff % 60000) / 1000)
    if (h > 0) return `${h}h ${m}m`
    if (m > 0) return `${m}m ${s}s`
    return `${s}s`
  }

  function estimateFuelCost() {
    let total = 0
    Object.entries(dispatchForm.shipQuantities || {}).forEach(([type, qty]) => {
      if (qty > 0 && fleetShips) {
        const ship = fleetShips.ships.find(s => s.type === type)
        if (ship) total += qty * (ship.fuel || 0)
      }
    })
    return total.toLocaleString()
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

  async function recallFleet(fleetId) {
    try {
      const res = await fetch(`/api/fleet/${fleetId}/recall`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) {
        const data = await res.json()
        error = data.error || 'Recall failed'
        return
      }
      await loadFleet()
    } catch (e) { error = e.message }
  }

  let splitTarget = null
  let mergeIds = []

  function toggleMerge(fleetId) {
    if (mergeIds.includes(fleetId)) {
      mergeIds = mergeIds.filter(id => id !== fleetId)
    } else {
      mergeIds = [...mergeIds, fleetId]
    }
  }

  async function doMerge() {
    if (mergeIds.length < 2) { error = 'Select at least 2 fleets'; return }
    try {
      const res = await fetch('/api/fleet/merge', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ fleet_ids: mergeIds })
      })
      if (!res.ok) {
        const data = await res.json()
        error = data.error || 'Merge failed'
        return
      }
      mergeIds = []
      await loadFleet()
    } catch (e) { error = e.message }
  }

  async function loadNotifications() {
    try {
      const res = await fetch('/api/notification/list?limit=20&offset=0', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        const data = await res.json()
        notifications = data.notifications || []
        notificationTotal = data.total || 0
      }
    } catch (e) { /* ignore */ }
  }

  async function loadNotificationUnread() {
    try {
      const res = await fetch('/api/notification/unread-count', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        const data = await res.json()
        notificationUnread = data.count || 0
      }
    } catch (e) { /* ignore */ }
  }

  function connectNotificationSSE() {
    if (notificationSocket) notificationSocket.close()
    const url = `/api/notification/stream?token=${token}`
    notificationSocket = new EventSource(url)
    notificationSocket.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data)
        if (event.type === 'notification') {
          notifications = [event.data, ...notifications]
          notificationTotal++
          notificationUnread++
        }
      } catch (err) { /* ignore */ }
    }
    notificationSocket.onerror = () => { /* ignore */ }
  }

  async function markNotificationRead(id) {
    try {
      const res = await fetch(`/api/notification/${id}/read`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        notifications = notifications.map(n => n.id === id ? { ...n, is_read: true } : n)
        notificationUnread = Math.max(0, notificationUnread - 1)
      }
    } catch (e) { /* ignore */ }
  }

  async function markAllNotificationsRead() {
    try {
      const res = await fetch('/api/notification/read-all', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        notifications = notifications.map(n => ({ ...n, is_read: true }))
        notificationUnread = 0
      }
    } catch (e) { /* ignore */ }
  }

  function toggleNotificationPanel() {
    notificationPanel = !notificationPanel
    if (notificationPanel) {
      loadNotifications()
      if (!notificationSocket) connectNotificationSSE()
    }
  }

  // ---- Quests ----
  let quests = []
  let questsTab = false
  let questTotalDM = 0

  async function loadQuests() {
    try {
      const res = await fetch('/api/quest/list', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ player_id: user?.id || 0 })
      })
      if (res.ok) {
        const data = await res.json()
        quests = data.quests || []
        questTotalDM = 0
        for (const q of quests) {
          if (q.progress.status === 'claimed') questTotalDM += q.definition.reward_dm
        }
      }
    } catch (e) { /* ignore */ }
  }

  async function claimQuestReward(questId) {
    try {
      const res = await fetch(`/api/quest/${questId}/claim`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ player_id: user?.id || 0, quest_id: questId })
      })
      if (res.ok) await loadQuests()
    } catch (e) { /* ignore */ }
  }

  // ---- Events ----
  let events = []
  let eventsTab = false
  let eventCountdownInterval

  async function loadEvents() {
    try {
      const res = await fetch('/api/event/all', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) events = await res.json()
    } catch (e) { /* ignore */ }
  }

  async function joinEvent(eventId) {
    try {
      const res = await fetch(`/api/event/${eventId}/join`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) await loadEvents()
      else { const d = await res.json(); error = d.error || 'Join failed' }
    } catch (e) { error = e.message }
  }

  async function claimEventReward(eventId) {
    try {
      const res = await fetch(`/api/event/${eventId}/claim`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) await loadEvents()
      else { const d = await res.json(); error = d.error || 'Claim failed' }
    } catch (e) { error = e.message }
  }

  function formatCountdown(target) {
    const diff = new Date(target) - new Date()
    if (diff <= 0) return 'Expired'
    const d = Math.floor(diff / 86400000)
    const h = Math.floor((diff % 86400000) / 3600000)
    const m = Math.floor((diff % 3600000) / 60000)
    const s = Math.floor((diff % 60000) / 1000)
    if (d > 0) return `${d}d ${h}h`
    if (h > 0) return `${h}h ${m}m`
    if (m > 0) return `${m}m ${s}s`
    return `${s}s`
  }

  function startEventCountdown() {
    if (eventCountdownInterval) clearInterval(eventCountdownInterval)
    eventCountdownInterval = setInterval(() => { events = events }, 1000)
  }

  function stopEventCountdown() {
    if (eventCountdownInterval) clearInterval(eventCountdownInterval)
  }

  function toggleEvents() {
    eventsTab = !eventsTab
    if (eventsTab) { loadEvents(); startEventCountdown() }
    else stopEventCountdown()
  }

  // ---- Daily Gift ----
  let dailyGift = null
  let showDailyGift = false
  let dailyGiftDismissed = false

  async function loadDailyGift() {
    try {
      const res = await fetch('/api/nebula/daily-gift/status', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        dailyGift = await res.json()
        if (dailyGift.can_claim && !dailyGiftDismissed) showDailyGift = true
      }
    } catch (e) { /* ignore */ }
  }

  async function claimDailyGift() {
    try {
      const res = await fetch('/api/nebula/daily-gift/claim', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        dailyGift = await res.json()
        showDailyGift = false
        await loadPlanet()
      }
    } catch (e) { /* ignore */ }
  }

  function dismissDailyGift() {
    showDailyGift = false
    dailyGiftDismissed = true
  }

  // ---- Store ----
  let storeItems = []
  let storeTab = false
  let storeMessage = ''

  async function loadStoreItems() {
    try {
      const res = await fetch('/api/nebula/store/items', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) storeItems = await res.json()
    } catch (e) { /* ignore */ }
  }

  async function buyStoreItem(itemId) {
    try {
      const res = await fetch(`/api/nebula/store/buy/${itemId}`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        const data = await res.json()
        storeMessage = data.message || 'Purchased!'
        await loadPlanet()
      } else {
        const d = await res.json()
        storeMessage = d.error || 'Purchase failed'
      }
    } catch (e) { storeMessage = e.message }
    setTimeout(() => { storeMessage = '' }, 4000)
  }

  // ---- Admin ----
  let adminTab = false
  let adminSearchQuery = ''
  let adminSearchResult = null
  let adminPlanetResult = null
  let adminResourceForm = { planet_id: '', metal: '', crystal: '', gas: '' }
  let adminDmForm = { player_id: '', amount: '', reason: '' }
  let adminCreditsForm = { player_id: '', amount: '', reason: '' }
  let adminBanForm = { player_id: '' }
  let adminGmForm = { player_id: '', subject: '', message: '' }
  let adminEventForm = { name: '', description: '', event_type: '', modifiers: '{}', starts_at: '', ends_at: '' }
  let adminMessage = ''

  async function adminSearchUsers() {
    try {
      const res = await fetch(`/api/admin/users?q=${encodeURIComponent(adminSearchQuery)}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) adminSearchResult = await res.json()
      else { const d = await res.json(); adminMessage = d.error || 'Search failed' }
    } catch (e) { adminMessage = e.message }
  }

  async function adminViewPlanets() {
    try {
      const res = await fetch(`/api/admin/planets?player_id=${adminResourceForm.planet_id}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) adminPlanetResult = await res.json()
      else { const d = await res.json(); adminMessage = d.error || 'View failed' }
    } catch (e) { adminMessage = e.message }
  }

  async function adminEditResources() {
    try {
      const res = await fetch(`/api/admin/planet/${adminResourceForm.planet_id}/resources`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ metal: parseInt(adminResourceForm.metal), crystal: parseInt(adminResourceForm.crystal), gas: parseInt(adminResourceForm.gas) })
      })
      if (res.ok) adminMessage = 'Resources updated'
      else { const d = await res.json(); adminMessage = d.error || 'Failed' }
    } catch (e) { adminMessage = e.message }
  }

  async function adminGrantDm() {
    try {
      const res = await fetch(`/api/admin/player/${adminDmForm.player_id}/dm`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ amount: parseInt(adminDmForm.amount), reason: adminDmForm.reason })
      })
      if (res.ok) adminMessage = 'DM granted'
      else { const d = await res.json(); adminMessage = d.error || 'Failed' }
    } catch (e) { adminMessage = e.message }
  }

  async function adminGrantCredits() {
    try {
      const res = await fetch(`/api/admin/player/${adminCreditsForm.player_id}/credits`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ amount: parseInt(adminCreditsForm.amount), reason: adminCreditsForm.reason })
      })
      if (res.ok) adminMessage = 'Credits granted'
      else { const d = await res.json(); adminMessage = d.error || 'Failed' }
    } catch (e) { adminMessage = e.message }
  }

  async function adminToggleBan() {
    try {
      const res = await fetch(`/api/admin/player/${adminBanForm.player_id}/ban`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) adminMessage = 'Ban toggled'
      else { const d = await res.json(); adminMessage = d.error || 'Failed' }
    } catch (e) { adminMessage = e.message }
  }

  async function adminSendGmMessage() {
    try {
      const res = await fetch('/api/admin/gm-message', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ player_id: parseInt(adminGmForm.player_id), subject: adminGmForm.subject, message: adminGmForm.message })
      })
      if (res.ok) adminMessage = 'GM message sent'
      else { const d = await res.json(); adminMessage = d.error || 'Failed' }
    } catch (e) { adminMessage = e.message }
  }

  async function adminCreateEvent() {
    let mods
    try { mods = JSON.parse(adminEventForm.modifiers) } catch { adminMessage = 'Invalid modifiers JSON'; return }
    try {
      const res = await fetch('/api/admin/event/create', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: adminEventForm.name, description: adminEventForm.description, event_type: adminEventForm.event_type, modifiers: mods, starts_at: adminEventForm.starts_at, ends_at: adminEventForm.ends_at })
      })
      if (res.ok) adminMessage = 'Event created'
      else { const d = await res.json(); adminMessage = d.error || 'Failed' }
    } catch (e) { adminMessage = e.message }
  }

  // ---- Tutorial ----
  let tutorial = null
  let showTutorial = false

  async function loadTutorial() {
    try {
      const res = await fetch('/api/tutorial/status', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        tutorial = await res.json()
        if (!tutorial.completed && tutorial.current_step > 0) showTutorial = true
      }
    } catch (e) { /* ignore */ }
  }

  async function claimTutorialStep(step) {
    try {
      const res = await fetch(`/api/tutorial/${step}/claim`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        tutorial = await res.json()
        if (tutorial.completed) showTutorial = false
        await loadPlanet()
      } else { const d = await res.json(); error = d.error || 'Claim failed' }
    } catch (e) { error = e.message }
  }

  async function skipTutorialStep() {
    try {
      const res = await fetch('/api/tutorial/skip', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) { showTutorial = false; tutorial = null }
    } catch (e) { /* ignore */ }
  }

  // ---- Gems ----
  let gemSlots = []
  let shardCounts = {}

  async function loadGems() {
    if (!planet) return
    try {
      const res = await fetch(`/api/planet/${planet.id}/gems`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        const data = await res.json()
        gemSlots = data.slots || []
        shardCounts = data.shards || {}
      }
    } catch (e) { /* ignore */ }
  }

  async function equipGem(slotIndex) {
    try {
      const res = await fetch(`/api/planet/${planet.id}/gems/equip`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ slot_index: slotIndex, gem_type: '', star_level: 1 })
      })
      if (res.ok) await loadGems()
      else { const d = await res.json(); error = d.error || 'Equip failed' }
    } catch (e) { error = e.message }
  }

  async function unequipGem(slotIndex) {
    try {
      const res = await fetch(`/api/planet/${planet.id}/gems/unequip`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ slot_index: slotIndex })
      })
      if (res.ok) await loadGems()
      else { const d = await res.json(); error = d.error || 'Unequip failed' }
    } catch (e) { error = e.message }
  }

  async function combineGem(slotIndex) {
    try {
      const res = await fetch(`/api/planet/${planet.id}/gems/combine`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ slot_index: slotIndex })
      })
      if (res.ok) await loadGems()
      else { const d = await res.json(); error = d.error || 'Combine failed' }
    } catch (e) { error = e.message }
  }

  // ---- Discoverer ----
  let discovererLevel = 0

  async function loadDiscoverer() {
    try {
      const res = await fetch('/api/nebula/discoverer', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        const data = await res.json()
        discovererLevel = data.level || 0
      }
    } catch (e) { /* ignore */ }
  }

  async function upgradeDiscoverer() {
    try {
      const res = await fetch('/api/nebula/discoverer/upgrade', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        const data = await res.json()
        discovererLevel = data.level || discovererLevel
        await loadPlanet()
      } else { const d = await res.json(); error = d.error || 'Upgrade failed' }
    } catch (e) { error = e.message }
  }

  // ---- Error Toast ----
  let errorToast = ''
  let errorToastTimer

  function showError(msg) {
    errorToast = msg
    if (errorToastTimer) clearTimeout(errorToastTimer)
    errorToastTimer = setTimeout(() => { errorToast = '' }, 5000)
  }

  // ---- UI Polish ----
  let loadingSkeleton = false
  let tooltipShip = null
  let tooltipBuilding = null

  function showShipTooltip(ship) { tooltipShip = ship }
  function hideShipTooltip() { tooltipShip = null }
  function showBuildingTooltip(building) { tooltipBuilding = building }
  function hideBuildingTooltip() { tooltipBuilding = null }
</script>

<div class="app">
  {#if token}
    <main class="dashboard">
      <header>
        <span class="user-email">{user ? user.email : ''}</span>
        <div class="header-actions">
          <button class="daily-gift-btn" on:click={() => { showDailyGift = true; loadDailyGift() }} title="Daily Gift">🎁</button>
          <button class="quests-toggle" on:click={() => { questsTab = !questsTab; if (questsTab) loadQuests() }}>
            📋
            {#if questTotalDM > 0}
              <span class="header-badge">{questTotalDM} DM</span>
            {/if}
          </button>
          <button class="events-toggle" on:click={toggleEvents}>⚡</button>
          <button class="store-toggle" on:click={() => { storeTab = !storeTab; if (storeTab) loadStoreItems() }}>🛒</button>
          <button class="notif-bell" on:click={toggleNotificationPanel} title="Notifications">
            🔔
            {#if notificationUnread > 0}
              <span class="notif-badge">{notificationUnread}</span>
            {/if}
          </button>
          <button class="logout" on:click={logout}>Logout</button>
          {#if user && user.id === 1}
            <button class="admin-toggle" on:click={() => adminTab = !adminTab}>🛡️ Admin</button>
          {/if}
        </div>
      </header>

      {#if notificationPanel}
        <div class="notif-panel">
          <div class="notif-panel-header">
            <h3>Notifications</h3>
            <div class="notif-panel-actions">
              {#if notificationUnread > 0}
                <button class="notif-mark-all" on:click={markAllNotificationsRead}>Mark all read</button>
              {/if}
              <button class="notif-close" on:click={() => { notificationPanel = false }}>✗</button>
            </div>
          </div>
          <div class="notif-list">
            {#each notifications as n}
              <div class="notif-item" class:unread={!n.is_read} on:click={() => { if (!n.is_read) markNotificationRead(n.id) }}>
                <div class="notif-item-header">
                  <span class="notif-category">{n.category}</span>
                  <span class="notif-time">{new Date(n.created_at).toLocaleString()}</span>
                </div>
                <div class="notif-title">{n.title}</div>
                <div class="notif-msg">{n.message}</div>
              </div>
            {:else}
              <p class="notif-empty">No notifications</p>
            {/each}
            {#if notificationTotal > notifications.length}
              <p class="notif-more">{notificationTotal - notifications.length} more</p>
            {/if}
          </div>
        </div>
      {/if}

      {#if showDailyGift && dailyGift}
        <div class="daily-gift-popup">
          <div class="daily-gift-content">
            <button class="daily-gift-close" on:click={dismissDailyGift}>✗</button>
            <h3>Daily Gift</h3>
            <p class="gift-streak">Day {dailyGift.streak_day} streak</p>
            <p class="gift-preview">{dailyGift.gift_preview}</p>
            <button class="btn-confirm" on:click={claimDailyGift} disabled={!dailyGift.can_claim}>
              {dailyGift.can_claim ? 'Claim' : 'Already Claimed'}
            </button>
          </div>
        </div>
      {/if}

      {#if storeTab}
        <div class="store-section">
          <button class="back-btn" on:click={() => storeTab = false}>← Back</button>
          <h3>Store</h3>
          {#if storeMessage}
            <div class="store-message">{storeMessage}</div>
          {/if}
          <div class="store-grid">
            {#each storeItems as item}
              <div class="store-card">
                <div class="store-name">{item.name}</div>
                <div class="store-desc">{item.description}</div>
                <div class="store-price">
                  {#if item.dm_price > 0}<span class="price-dm">💠 {item.dm_price} DM</span>{/if}
                  {#if item.credits_price > 0}<span class="price-credits">💎 {item.credits_price} Credits</span>{/if}
                </div>
                <button class="btn-store-buy" on:click={() => buyStoreItem(item.id)}>Buy</button>
              </div>
            {:else}
              <p class="empty-msg">No items available</p>
            {/each}
          </div>
        </div>
      {/if}

      {#if adminTab}
        <div class="admin-section">
          <button class="back-btn" on:click={() => adminTab = false}>← Back</button>
          <h3>Admin Panel</h3>
          {#if adminMessage}
            <div class="admin-message">{adminMessage}</div>
          {/if}
          <div class="admin-sections">
            <div class="admin-card">
              <h4>User Search</h4>
              <input bind:value={adminSearchQuery} placeholder="Email" class="form-input" />
              <button class="btn-admin" on:click={adminSearchUsers}>Search</button>
              {#if adminSearchResult}
                <pre class="admin-result">{JSON.stringify(adminSearchResult, null, 2)}</pre>
              {/if}
            </div>
            <div class="admin-card">
              <h4>Planet View</h4>
              <input type="number" bind:value={adminResourceForm.planet_id} placeholder="Player ID" class="form-input" />
              <button class="btn-admin" on:click={adminViewPlanets}>View Planets</button>
              {#if adminPlanetResult}
                <pre class="admin-result">{JSON.stringify(adminPlanetResult, null, 2)}</pre>
              {/if}
            </div>
            <div class="admin-card">
              <h4>Resource Edit</h4>
              <input type="number" bind:value={adminResourceForm.planet_id} placeholder="Planet ID" class="form-input" />
              <input type="number" bind:value={adminResourceForm.metal} placeholder="Metal" class="form-input" />
              <input type="number" bind:value={adminResourceForm.crystal} placeholder="Crystal" class="form-input" />
              <input type="number" bind:value={adminResourceForm.gas} placeholder="Gas" class="form-input" />
              <button class="btn-admin" on:click={adminEditResources}>Update Resources</button>
            </div>
            <div class="admin-card">
              <h4>DM Grant</h4>
              <input type="number" bind:value={adminDmForm.player_id} placeholder="Player ID" class="form-input" />
              <input type="number" bind:value={adminDmForm.amount} placeholder="Amount" class="form-input" />
              <input bind:value={adminDmForm.reason} placeholder="Reason" class="form-input" />
              <button class="btn-admin" on:click={adminGrantDm}>Grant DM</button>
            </div>
            <div class="admin-card">
              <h4>Credits Grant</h4>
              <input type="number" bind:value={adminCreditsForm.player_id} placeholder="Player ID" class="form-input" />
              <input type="number" bind:value={adminCreditsForm.amount} placeholder="Amount" class="form-input" />
              <input bind:value={adminCreditsForm.reason} placeholder="Reason" class="form-input" />
              <button class="btn-admin" on:click={adminGrantCredits}>Grant Credits</button>
            </div>
            <div class="admin-card">
              <h4>Ban/Unban</h4>
              <input type="number" bind:value={adminBanForm.player_id} placeholder="Player ID" class="form-input" />
              <button class="btn-admin" on:click={adminToggleBan}>Toggle Ban</button>
            </div>
            <div class="admin-card">
              <h4>GM Message</h4>
              <input type="number" bind:value={adminGmForm.player_id} placeholder="Player ID" class="form-input" />
              <input bind:value={adminGmForm.subject} placeholder="Subject" class="form-input" />
              <textarea bind:value={adminGmForm.message} placeholder="Message..." class="form-textarea"></textarea>
              <button class="btn-admin" on:click={adminSendGmMessage}>Send</button>
            </div>
            <div class="admin-card">
              <h4>Create Event</h4>
              <input bind:value={adminEventForm.name} placeholder="Name" class="form-input" />
              <input bind:value={adminEventForm.description} placeholder="Description" class="form-input" />
              <input bind:value={adminEventForm.event_type} placeholder="Event type" class="form-input" />
              <textarea bind:value={adminEventForm.modifiers} placeholder="Modifiers JSON e.g. bonus:1.5" class="form-textarea"></textarea>
              <input bind:value={adminEventForm.starts_at} placeholder="Starts at (ISO)" class="form-input" />
              <input bind:value={adminEventForm.ends_at} placeholder="Ends at (ISO)" class="form-input" />
              <button class="btn-admin" on:click={adminCreateEvent}>Create</button>
            </div>
          </div>
        </div>
      {/if}

      {#if showTutorial && tutorial}
        <div class="tutorial-overlay">
          <div class="tutorial-card">
            <h3>Tutorial</h3>
            <p class="tutorial-step">Step {tutorial.current_step}</p>
            <p class="tutorial-title">{tutorial.step_title}</p>
            <p class="tutorial-desc">{tutorial.step_description}</p>
            <div class="tutorial-actions">
              <button class="btn-tutorial-claim" on:click={() => claimTutorialStep(tutorial.current_step)}>Claim Reward</button>
              <button class="btn-tutorial-skip" on:click={skipTutorialStep}>Skip</button>
            </div>
          </div>
        </div>
      {/if}

      {#if errorToast}
        <div class="toast-error">{errorToast}</div>
      {/if}

      {#if planet}
        {#if renameMode}
          <div class="rename-row">
            <input class="rename-input" bind:value={renameName} placeholder="Planet name" maxlength="100" />
            <button class="btn-confirm" on:click={doRename}>✓</button>
            <button class="btn-cancel" on:click={() => { renameMode = false }}>✗</button>
          </div>
        {:else}
          <h1 class="name" on:click={startRename} title="Click to rename">{planet.name} ✏️</h1>
        {/if}
        <p class="coords">[{planet.galaxy}:{planet.system}:{planet.position}]</p>
        <p class="planet-meta">
          <span class="type-badge type-{planet.type}">{planetTypeIcon(planet.type)} {planetTypeLabel(planet.type)}</span>
          <span class="temperature">{planet.temperature}°C</span>
          <span class="temp-hint">{temperatureGasHint(planet.temperature)}{temperatureSolarHint(planet.temperature)}</span>
        </p>
        <p class="fields">Fields: {planet.fields_used}/{planet.max_fields}</p>

        <div class="player-stats">
          <span class="vip-badge">VIP {planet.vip_level}</span>
          <span class="rank-badge">Rank {planet.rank}</span>
          <span class="credits-badge">💎 {planet.credits || 0}</span>
          <span class="discoverer-badge">🔭 Lv.{discovererLevel}</span>
          <button class="btn-discoverer-upgrade" on:click={upgradeDiscoverer}>Upgrade</button>
        </div>

        <button class="chat-toggle" on:click={() => { chatView = true; loadChat() }}>Chat</button>
        <button class="msg-toggle" on:click={() => { messagingView = true; loadInbox(); loadUnreadCount() }}>
          Messages {unreadCount > 0 ? `(${unreadCount})` : ''}
        </button>
        <button class="alliance-toggle" on:click={() => { allianceView = true; loadAlliance() }}>Alliance</button>
        <button class="buddy-toggle" on:click={() => { buddyView = true; loadBuddyList() }}>Friends</button>
        <button class="galaxy-toggle" on:click={showGalaxyTab}>Galaxy</button>
        <button class="shipyard-toggle" on:click={() => { loadShipyard(); loadDefense() }}>Shipyard</button>
        <button class="fleet-toggle" on:click={loadFleet}>Fleet</button>
        <button class="moon-toggle" on:click={loadMoonBuildings}>Moon</button>

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

        {#if planet.buildings && planet.buildings.some(b => b.type === 'galactonite_research_center' && b.level > 0)}
          <div class="gem-section">
            <h2>Gems</h2>
            <div class="gem-slots">
              {#each gemSlots as slot, i}
                <div class="gem-slot" class:filled={slot.equipped}>
                  <span class="gem-index">Slot {i + 1}</span>
                  {#if slot.equipped}
                    <span class="gem-type">{slot.gem_type}</span>
                    <span class="gem-stars">{'★'.repeat(slot.star_level)}</span>
                    <div class="gem-actions">
                      <button class="btn-gem-unequip" on:click={() => unequipGem(i)}>Unequip</button>
                      <button class="btn-gem-combine" on:click={() => combineGem(i)}>Combine</button>
                    </div>
                  {:else}
                    <span class="gem-empty">Empty</span>
                    <button class="btn-gem-equip" on:click={() => equipGem(i)}>Equip</button>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/if}

        {#if positions}
          <div class="galaxy-section">
            <button class="back-btn" on:click={() => { positions = null; galaxyData = null; fleetView = null; currentSystemNum = 0 }}>← Back to Systems</button>
            <div class="position-grid">
              {#each positions.positions as pos}
                {@const fi = getFleetIndicators(pos)}
                <div class="position-card" class:occupied={pos.state === 'occupied'} title={fi.tooltip}>
                  <span class="pos-num">#{pos.position}</span>
                  {#if fi.count > 0}
                    <span class="fleet-badge">{fi.count}</span>
                  {/if}
                  {#if pos.state === 'occupied'}
                    <span class="pos-name">{pos.planet_name}</span>
                    <span class="pos-player">Player {pos.player_id}{#if pos.type === 'npc'} <span class="npc-badge">(NPC)</span>{/if}</span>
                    {#if fi.indicators.length > 0}
                      <span class="fleet-indicators">
                        {#each fi.indicators as ind}
                          <span class="fleet-icon">{ind}</span>
                        {/each}
                      </span>
                    {/if}
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
                <button class="system-row" on:click={() => selectSystem(sys.id, sys.system_num)}>
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
                    <div class="qty-buttons">
                      <button class="qty-btn" on:click={() => setQuantity(ship.type, 1)}>1</button>
                      <button class="qty-btn" on:click={() => setQuantity(ship.type, 10)}>10</button>
                      <button class="qty-btn" on:click={() => setQuantity(ship.type, 100)}>100</button>
                      <button class="qty-btn" on:click={() => setQuantity(ship.type, maxAfford(ship))}>Max</button>
                    </div>
                    <div class="qty-input-row">
                      <input type="number" min="1" bind:value={buildQuantities[ship.type]} class="qty-input" />
                      <button class="btn-build" on:click={() => buildShips(ship.type)}>Build</button>
                    </div>
                    {#if lastBuildResult && lastBuildResult.type === ship.type}
                      <div class="build-result">
                        Built {lastBuildResult.quantity} in {formatBuildTime(lastBuildResult.build_time_seconds)}
                      </div>
                    {/if}
                  </div>
                </div>
              {/each}
            </div>

            <h3>Defense</h3>
            {#if defenseData}
              <div class="defense-grid">
                {#each defenseData.defenses as def}
                  <div class="defense-card">
                    <div class="defense-header">
                      <span class="defense-name">{def.name}</span>
                      <span class="defense-qty">Owned: {def.quantity}</span>
                    </div>
                    <div class="defense-stats">
                      <span class="stat">🛡️{def.shield}</span>
                      <span class="stat">⚔️{def.attack}</span>
                      <span class="stat">🔋{def.strength}</span>
                    </div>
                    <div class="defense-cost">
                      <span class="cost metal">M {def.metal}</span>
                      <span class="cost crystal">C {def.crystal}</span>
                      {#if def.gas > 0}
                        <span class="cost gas">G {def.gas}</span>
                      {/if}
                    </div>
                    <div class="defense-build">
                      <div class="qty-buttons">
                        <button class="qty-btn" on:click={() => setDefenseQuantity(def.type, 1)}>1</button>
                        <button class="qty-btn" on:click={() => setDefenseQuantity(def.type, 10)}>10</button>
                        <button class="qty-btn" on:click={() => setDefenseQuantity(def.type, 100)}>100</button>
                        <button class="qty-btn" on:click={() => setDefenseQuantity(def.type, maxAffordDefense(def))}>Max</button>
                      </div>
                      <div class="qty-input-row">
                        <input type="number" min="1" bind:value={defenseBuildQuantities[def.type]} class="qty-input" />
                        <button class="btn-build" on:click={() => buildDefense(def.type)}>Build</button>
                      </div>
                      {#if lastDefenseBuildResult && lastDefenseBuildResult.type === def.type}
                        <div class="build-result">
                          Built {lastDefenseBuildResult.quantity} in {formatBuildTime(lastDefenseBuildResult.build_time_seconds)}
                        </div>
                      {/if}
                    </div>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
        {#if moonData}
          <div class="moon-section">
            <h3>Moon Base <span class="moon-coords">[{planet.galaxy}:{planet.system}:{planet.position}]</span></h3>
            <p class="moon-fields">Fields: {moonData.fields_used}/{moonData.max_fields}</p>

            <div class="moon-buildings">
              {#each moonData.buildings as building}
                {@const cost = buildingCost(building.type, building.level)}
                <div class="moon-building">
                  <div class="mb-info">
                    <span class="mb-name">{buildingLabel(building.type)}</span>
                    <span class="mb-level">Lv.{building.level}</span>
                  </div>
                  <div class="mb-actions">
                    <button class="btn-moon-upgrade" on:click={() => upgradeMoonBuilding(building.type)}
                      disabled={!canAfford(building.type, building.level)}>
                      Upgrade
                    </button>
                  </div>
                  <div class="mb-cost">
                    <span class="cost metal">M {cost.metal}</span>
                    {#if cost.crystal > 0}<span class="cost crystal">C {cost.crystal}</span>{/if}
                    {#if cost.gas > 0}<span class="cost gas">G {cost.gas}</span>{/if}
                  </div>
                </div>
              {/each}
            </div>

            {#if moonData.buildings.some(b => b.type === 'wormhole_generator' && b.level > 0)}
              <div class="wormhole-section">
                <h4>Wormhole Link</h4>
                <div class="wormhole-form">
                  <input type="number" placeholder="Galaxy" bind:value={wormholeLinkForm.targetGalaxy} />
                  <input type="number" placeholder="System" bind:value={wormholeLinkForm.targetSystem} />
                  <input type="number" placeholder="Position" bind:value={wormholeLinkForm.targetPosition} />
                  <button class="btn-wormhole-link" on:click={linkWormholes}>Link</button>
                </div>
              </div>
            {/if}

            {#if moonData.buildings.some(b => b.type === 'pioneer_lab' && b.level > 0)}
              <div class="iron-behemoth-section">
                <h4>Iron Behemoth</h4>
                <div class="behemoth-build">
                  <input type="number" min="1" bind:value={ironBehemothQty} class="behemoth-qty" />
                  <button class="btn-behemoth-build" on:click={buildIronBehemoth}>Build</button>
                </div>
                {#if lastIronBehemothResult}
                  <div class="build-result">
                    Built {lastIronBehemothResult.quantity} Iron Behemoth{lastIronBehemothResult.quantity > 1 ? 's' : ''} in {formatBuildTime(lastIronBehemothResult.build_time_seconds)}
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        {/if}

        {#if questsTab}
          <div class="quests-section">
            <button class="back-btn" on:click={() => { questsTab = false }}>← Back</button>
            <h3>Quests</h3>
            <p class="quests-dm">Total DM earned: {questTotalDM}</p>
            <div class="quest-list">
              {#each quests as q}
                <div class="quest-card" class:completed={q.progress.status === 'completed' || q.progress.status === 'claimed'} class:claimed={q.progress.status === 'claimed'}>
                  <div class="quest-header">
                    <span class="quest-name">{q.definition.name}</span>
                    <span class="quest-category">{q.definition.category}</span>
                  </div>
                  <div class="quest-desc">{q.definition.description}</div>
                  <div class="quest-progress">
                    <div class="quest-bar-bg">
                      <div class="quest-bar-fill" style="width: {Math.min(100, q.progress.progress_current / q.progress.progress_target * 100)}%"></div>
                    </div>
                    <span class="quest-progress-text">{q.progress.progress_current}/{q.progress.progress_target}</span>
                  </div>
                  <div class="quest-reward">
                    <span class="reward-dm">+{q.definition.reward_dm} DM</span>
                    {#if q.definition.reward_metal > 0}<span class="reward-metal">M {q.definition.reward_metal}</span>{/if}
                    {#if q.definition.reward_crystal > 0}<span class="reward-crystal">C {q.definition.reward_crystal}</span>{/if}
                    {#if q.definition.reward_gas > 0}<span class="reward-gas">G {q.definition.reward_gas}</span>{/if}
                  </div>
                  {#if q.progress.status === 'completed'}
                    <button class="btn-confirm btn-claim-quest" on:click={() => claimQuestReward(q.definition.id)}>Claim</button>
                  {:else if q.progress.status === 'claimed'}
                    <span class="claimed-badge">✓ Claimed</span>
                  {:else}
                    <span class="quest-status">{q.progress.status}</span>
                  {/if}
                </div>
              {:else}
                <p class="empty-msg">No quests available</p>
              {/each}
            </div>
          </div>
        {:else if eventsTab}
          <div class="events-section">
            <button class="back-btn" on:click={() => { eventsTab = false; stopEventCountdown() }}>← Back</button>
            <h3>Events</h3>
            <div class="event-list">
              {#each events as ev}
                <div class="event-card" class:active={ev.status === 'active'} class:ended={ev.status === 'ended'}>
                  <div class="event-header">
                    <span class="event-name">{ev.name}</span>
                    <span class="event-type-badge">{ev.event_type}</span>
                  </div>
                  <div class="event-desc">{ev.description}</div>
                  <div class="event-modifiers">
                    {#each Object.entries(ev.modifiers) as [key, val]}
                      <span class="modifier">{key}: {val}</span>
                    {/each}
                  </div>
                  <div class="event-timer">
                    {#if ev.status === 'active'}
                      <span class="timer-label">Ends in: </span>
                      <span class="timer-value">{formatCountdown(ev.ends_at)}</span>
                    {:else if ev.status === 'upcoming'}
                      <span class="timer-label">Starts in: </span>
                      <span class="timer-value">{formatCountdown(ev.starts_at)}</span>
                    {:else}
                      <span class="timer-label">Ended</span>
                    {/if}
                  </div>
                  <div class="event-actions">
                    {#if ev.status === 'active' && !ev.joined}
                      <button class="btn-confirm" on:click={() => joinEvent(ev.id)}>Join</button>
                    {:else if ev.joined && ev.completed && !ev.rewards_claimed}
                      <button class="btn-confirm" on:click={() => claimEventReward(ev.id)}>Claim Rewards</button>
                    {:else if ev.joined}
                      <span class="joined-badge">{ev.rewards_claimed ? '✓ Rewards Claimed' : '✓ Joined'}</span>
                    {/if}
                  </div>
                </div>
              {:else}
                <p class="empty-msg">No events available</p>
              {/each}
            </div>
          </div>
        {/if}
        {#if fleetData}
          <div class="fleet-section">
            <h3>My Fleets</h3>
            {#if mergeIds.length >= 2}
              <div class="merge-bar">
                <span>{mergeIds.length} fleets selected</span>
                <button class="btn-action btn-merge-do" on:click={doMerge}>Merge</button>
                <button class="btn-action btn-cancel" on:click={() => mergeIds = []}>Cancel</button>
              </div>
            {/if}
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
                    <div class="fleet-arrival">
                      {#if fleet.status === 'in_transit'}
                        Arrives {formatFleetETA(fleet.arrives_at)}
                      {:else}
                        Arrived: {new Date(fleet.arrives_at).toLocaleString()}
                      {/if}
                    </div>
                  {/if}
                  <div class="fleet-actions">
                    {#if fleet.status === 'stationed'}
                      <button class="btn-action btn-split" on:click={() => splitTarget = fleet.id}>Split</button>
                      <button class="btn-action btn-merge-toggle"
                        class:selected={mergeIds.includes(fleet.id)}
                        on:click={() => toggleMerge(fleet.id)}>
                        {mergeIds.includes(fleet.id) ? 'Selected' : 'Merge'}
                      </button>
                    {/if}
                    {#if fleet.status === 'in_transit'}
                      <button class="btn-action btn-recall" on:click={() => recallFleet(fleet.id)}>Recall</button>
                    {/if}
                  </div>
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
              <div class="form-row">
                <label>Est. Fuel</label>
                <span class="fuel-estimate">~{estimateFuelCost()} gas</span>
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
      {#if chatView}
        <div class="chat-section">
          <button class="back-btn" on:click={() => { chatView = null; if (chatSocket) { chatSocket.close(); chatSocket = null } }}>← Back</button>
          <h3>Chat</h3>
          <div class="channel-tabs">
            <button class="channel-tab" class:active={chatChannel === 'global'} on:click={() => { chatChannel = 'global'; loadChatMessages() }}>Global</button>
            <button class="channel-tab" class:active={chatChannel === 'alliance'} on:click={() => { chatChannel = 'alliance'; loadChatMessages() }}>Alliance</button>
          </div>
          <div class="chat-messages">
            {#each chatMessages as msg}
              <div class="chat-msg">
                <span class="chat-sender">{msg.sender_name}:</span>
                <span class="chat-text">{msg.content}</span>
                <span class="chat-time">{new Date(msg.created_at).toLocaleTimeString()}</span>
              </div>
            {/each}
          </div>
          <div class="chat-input-row">
            <input class="chat-input" bind:value={chatInput} placeholder="Type a message..." maxlength="500" on:keydown={(e) => { if (e.key === 'Enter') sendChat() }} />
            <button class="btn-chat-send" on:click={sendChat}>Send</button>
          </div>
        </div>
      {/if}

      {#if messagingView}
        <div class="msg-section">
          <button class="back-btn" on:click={() => { messagingView = null; messagingTab = 'inbox' }}>← Back</button>
          <h3>Messages</h3>
          <div class="channel-tabs">
            <button class="channel-tab" class:active={messagingTab === 'inbox'} on:click={() => { messagingTab = 'inbox'; loadInbox() }}>Inbox {unreadCount > 0 ? `(${unreadCount})` : ''}</button>
            <button class="channel-tab" class:active={messagingTab === 'outbox'} on:click={() => { messagingTab = 'outbox'; loadOutbox() }}>Sent</button>
            <button class="channel-tab" class:active={messagingTab === 'compose'} on:click={() => { messagingTab = 'compose' }}>Compose</button>
          </div>
          {#if messagingTab === 'compose'}
            <div class="compose-form">
              <input bind:value={composeMsg.receiver_id} type="number" placeholder="Player ID" class="form-input" />
              <textarea bind:value={composeMsg.content} placeholder="Message..." maxlength="500" class="form-textarea"></textarea>
              <button class="btn-chat-send" on:click={sendPrivateMsg}>Send</button>
            </div>
          {:else}
            <div class="msg-list">
              {#each messagingConv as msg}
                <div class="msg-card" on:click={() => markMsgRead(msg.id)}>
                  <span class="msg-sender">{messagingTab === 'inbox' ? `From: Player ${msg.sender_id}` : `To: Player ${msg.receiver_id}`}</span>
                  <span class="msg-text">{msg.content}</span>
                  <span class="msg-time">{new Date(msg.created_at).toLocaleString()}</span>
                  {#if msg.is_system}<span class="system-tag">System</span>{/if}
                </div>
              {:else}
                <p class="empty-msg">No messages</p>
              {/each}
            </div>
          {/if}
        </div>
      {/if}

      {#if allianceView}
        <div class="alliance-section">
          <button class="back-btn" on:click={() => { allianceView = null }}>← Back</button>
          <h3>Alliance: {allianceView.name}</h3>
          <p class="alliance-tag">[{allianceView.tag}] · {allianceView.role}</p>
          <div class="channel-tabs">
            <button class="channel-tab" class:active={allianceTab === 'info'} on:click={() => { allianceTab = 'info' }}>Info</button>
            <button class="channel-tab" class:active={allianceTab === 'members'} on:click={() => { allianceTab = 'members' }}>Members</button>
            <button class="channel-tab" class:active={allianceTab === 'bulletin'} on:click={() => { allianceTab = 'bulletin' }}>Bulletin</button>
          </div>
          {#if allianceTab === 'info'}
            <div class="alliance-info">
              <p>Members: {allianceView.members ? allianceView.members.length : 0}</p>
              {#if allianceView.role === 'founder' || allianceView.role === 'officer'}
                <button class="btn-danger" on:click={leaveAlliance}>Leave Alliance</button>
              {:else}
                <button class="btn-danger" on:click={leaveAlliance}>Leave Alliance</button>
              {/if}
            </div>
          {:else if allianceTab === 'members'}
            <div class="member-list">
              {#each allianceView.members as member}
                <div class="member-card">
                  <span class="member-id">Player {member.player_id}</span>
                  <span class="member-role">{member.role}</span>
                  <span class="member-online">{member.online ? '🟢 Online' : '⚪ Offline'}</span>
                  {#if allianceView.role === 'founder' && member.role !== 'founder'}
                    <button class="btn-action-small" on:click={() => transferFounder(member.player_id)}>Transfer</button>
                  {/if}
                </div>
              {/each}
            </div>
          {:else if allianceTab === 'bulletin'}
            <div class="bulletin-board">
              {#if allianceView.role === 'founder' || allianceView.role === 'officer'}
                <div class="bulletin-form">
                  <input bind:value={bulletinForm.title} placeholder="Title" class="form-input" />
                  <textarea bind:value={bulletinForm.content} placeholder="Content..." class="form-textarea"></textarea>
                  <button class="btn-chat-send" on:click={postBulletin}>Post</button>
                </div>
              {/if}
              {#each bulletins as post}
                <div class="bulletin-post">
                  <strong>{post.title}</strong>
                  <p>{post.content}</p>
                  <span class="msg-time">by Player {post.author_player_id} · {new Date(post.created_at).toLocaleString()}</span>
                  {#if allianceView.role === 'founder' || allianceView.role === 'officer'}
                    <button class="btn-action-small" on:click={() => deleteBulletin(post.id)}>Delete</button>
                  {/if}
                </div>
              {:else}
                <p class="empty-msg">No bulletins</p>
              {/each}
            </div>
          {/if}
        </div>
      {/if}

      {#if buddyView}
        <div class="buddy-section">
          <button class="back-btn" on:click={() => { buddyView = null }}>← Back</button>
          <h3>Friends</h3>
          <div class="buddy-form">
            <input bind:value={addFriendId} type="number" placeholder="Player ID to add" class="form-input" />
            <button class="btn-chat-send" on:click={addFriend}>Add</button>
          </div>
          <div class="buddy-list">
            {#each buddyView.friends as friend}
              <div class="buddy-card">
                <span class="buddy-id">Player {friend.player_id}</span>
                <span class="buddy-status">{friend.online ? '🟢 Online' : '⚪ Offline'}</span>
                <button class="btn-action-small btn-danger" on:click={() => removeFriend(friend.player_id)}>Remove</button>
              </div>
            {:else}
              <p class="empty-msg">No friends yet</p>
            {/each}
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
  .temp-hint { font-size: 0.7rem; color: #8a9ab5; }
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
  .fleet-badge {
    display: inline-flex; align-items: center; justify-content: center;
    width: 18px; height: 18px; border-radius: 50%;
    background: #d4744a; color: #0a0e1a; font-size: 0.6rem; font-weight: 700;
    position: absolute; top: 2px; right: 2px;
  }
  .position-card { position: relative; }
  .fleet-indicators {
    display: flex; justify-content: center; gap: 0.15rem; margin-top: 0.15rem;
  }
  .fleet-icon { font-size: 0.75rem; }

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
  .ship-build { display: flex; flex-direction: column; gap: 0.25rem; }
  .qty-buttons { display: flex; gap: 0.2rem; }
  .qty-btn {
    flex: 1; padding: 0.15rem 0; background: #0a0e1a; border: 1px solid #243050;
    border-radius: 3px; color: #5a7a9a; font-size: 0.65rem; cursor: pointer;
  }
  .qty-btn:hover { background: #1a2a4a; color: #8ab5d4; }
  .qty-input-row { display: flex; gap: 0.25rem; }
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
  .build-result { font-size: 0.65rem; color: #74d4a8; text-align: center; }

  .defense-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem; margin-bottom: 1rem; }
  .defense-card {
    padding: 0.5rem; background: #1a1a30; border: 1px solid #2a2040;
    border-radius: 6px; font-size: 0.8rem;
  }
  .defense-header { display: flex; justify-content: space-between; margin-bottom: 0.3rem; }
  .defense-name { font-weight: 600; color: #c8d6e5; }
  .defense-qty { font-size: 0.7rem; color: #5a7a9a; }
  .defense-stats { display: flex; gap: 0.5rem; font-size: 0.7rem; color: #b074d4; margin-bottom: 0.3rem; }
  .defense-cost { display: flex; gap: 0.5rem; font-size: 0.7rem; margin-bottom: 0.3rem; }
  .defense-build { display: flex; flex-direction: column; gap: 0.25rem; }

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
  .fuel-estimate { font-size: 0.75rem; color: #74d4a8; font-family: monospace; }
  .fleet-empty { font-size: 0.8rem; color: #5a5a6a; text-align: center; padding: 1rem; }
  .fleet-actions { display: flex; gap: 0.35rem; margin-top: 0.35rem; }
  .btn-action {
    padding: 0.2rem 0.5rem; border: none; border-radius: 3px;
    font-size: 0.65rem; cursor: pointer;
  }
  .btn-recall { background: #5a3a2a; color: #e8b87a; }
  .btn-recall:hover { background: #6a4a3a; }
  .btn-split { background: #2a3a5a; color: #7ab8e8; }
  .btn-split:hover { background: #3a4a6a; }
  .btn-merge-toggle { background: #2a4a3a; color: #7ae8a8; }
  .btn-merge-toggle:hover { background: #3a5a4a; }
  .btn-merge-toggle.selected { background: #4a6a3a; }
  .btn-merge-do { background: #4a6a3a; color: #a8e8a8; }
  .btn-cancel { background: #5a3a3a; color: #e8a8a8; }
  .merge-bar {
    display: flex; align-items: center; justify-content: center; gap: 0.5rem;
    padding: 0.4rem; margin-bottom: 0.5rem; background: #1a2a1a; border: 1px solid #2a4a2a;
    border-radius: 6px; font-size: 0.75rem; color: #8ab58a;
  }
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
  .moon-toggle {
    display: block; margin: 1rem auto; padding: 0.5rem 1rem;
    background: #1a2a3a; border: 1px solid #2a4a6a; border-radius: 6px;
    color: #74a8c8; font-size: 0.85rem; cursor: pointer;
  }
  .moon-toggle:hover { background: #2a3a5a; }
  .moon-section { margin-top: 1.5rem; }
  .moon-section h3 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.5rem; text-align: center; }
  .moon-section h4 { font-size: 0.8rem; color: #8a9ab5; margin-bottom: 0.4rem; text-align: center; }
  .moon-coords { font-size: 0.7rem; color: #5a7a9a; font-family: monospace; }
  .moon-fields { font-size: 0.75rem; color: #6a8a6a; margin-bottom: 0.75rem; text-align: center; }
  .moon-buildings { display: flex; flex-direction: column; gap: 0.4rem; }
  .moon-building {
    display: flex; justify-content: space-between; align-items: center;
    padding: 0.4rem 0.6rem; background: #1a2830; border: 1px solid #1a3a4a;
    border-radius: 6px; font-size: 0.8rem;
  }
  .mb-info { display: flex; flex-direction: column; align-items: flex-start; gap: 0.1rem; }
  .mb-name { font-size: 0.8rem; color: #c8d6e5; }
  .mb-level { font-size: 0.7rem; color: #5a7a9a; }
  .mb-actions { display: flex; gap: 0.3rem; }
  .btn-moon-upgrade {
    padding: 0.2rem 0.5rem; background: #2a4a3a; border: none;
    border-radius: 4px; color: #a8e8a8; font-size: 0.7rem; cursor: pointer;
  }
  .btn-moon-upgrade:disabled { opacity: 0.4; cursor: not-allowed; }
  .btn-moon-upgrade:hover:not(:disabled) { background: #3a5a4a; }
  .mb-cost { display: flex; gap: 0.3rem; font-size: 0.65rem; }
  .wormhole-section { margin-top: 1rem; text-align: center; }
  .wormhole-form { display: flex; gap: 0.3rem; justify-content: center; flex-wrap: wrap; }
  .wormhole-form input {
    width: 60px; padding: 0.3rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 4px; color: #c8d6e5; font-size: 0.75rem; text-align: center;
  }
  .btn-wormhole-link {
    padding: 0.3rem 0.6rem; background: #2a4a6a; border: none;
    border-radius: 4px; color: #8ab5d4; font-size: 0.7rem; cursor: pointer;
  }
  .btn-wormhole-link:hover { background: #3a5a7a; }
  .iron-behemoth-section { margin-top: 1rem; text-align: center; }
  .behemoth-build { display: flex; gap: 0.3rem; justify-content: center; }
  .behemoth-qty {
    width: 60px; padding: 0.3rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 4px; color: #c8d6e5; font-size: 0.75rem; text-align: center;
  }
  .btn-behemoth-build {
    padding: 0.3rem 0.6rem; background: #4a2a5a; border: none;
    border-radius: 4px; color: #c8a8e8; font-size: 0.7rem; cursor: pointer;
  }
  .btn-behemoth-build:hover { background: #5a3a6a; }

  .btn-dispatch {
    padding: 0.5rem; background: #4a2a6a; border: none;
    border-radius: 6px; color: #c8a8e8; font-size: 0.85rem; font-weight: 600; cursor: pointer;
  }
  .btn-dispatch:hover { background: #5a3a7a; }

  .chat-toggle { background: #1a3a3a; border: 1px solid #2a5a5a; color: #74d4d4; }
  .chat-toggle:hover { background: #2a4a4a; }
  .msg-toggle { background: #2a1a3a; border: 1px solid #4a2a5a; color: #b074d4; }
  .msg-toggle:hover { background: #3a2a4a; }
  .alliance-toggle { background: #1a2a3a; border: 1px solid #2a4a6a; color: #74a8d4; }
  .alliance-toggle:hover { background: #2a3a5a; }
  .buddy-toggle { background: #2a3a2a; border: 1px solid #3a5a3a; color: #8ad48a; }
  .buddy-toggle:hover { background: #3a4a3a; }

  .chat-toggle, .msg-toggle, .alliance-toggle, .buddy-toggle {
    display: block; margin: 0.5rem auto; padding: 0.5rem 1rem;
    border-radius: 6px; font-size: 0.85rem; cursor: pointer;
  }

  .chat-section, .msg-section, .alliance-section, .buddy-section {
    margin-top: 1.5rem; text-align: left;
  }
  .chat-section h3, .msg-section h3, .alliance-section h3, .buddy-section h3 {
    font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-align: center;
  }
  .channel-tabs { display: flex; gap: 0.25rem; margin-bottom: 0.75rem; justify-content: center; }
  .channel-tab {
    padding: 0.3rem 0.75rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 4px; color: #5a7a9a; font-size: 0.75rem; cursor: pointer;
  }
  .channel-tab.active { background: #243050; border-color: #3a6ab5; color: #8ab5d4; }
  .channel-tab:hover { background: #1e2a4a; }

  .chat-messages {
    height: 200px; overflow-y: auto; background: #0a0e1a; border: 1px solid #1e2a4a;
    border-radius: 6px; padding: 0.5rem; margin-bottom: 0.5rem;
  }
  .chat-msg { padding: 0.2rem 0; font-size: 0.75rem; border-bottom: 1px solid #1a2340; }
  .chat-sender { font-weight: 600; color: #8ab5d4; margin-right: 0.35rem; }
  .chat-text { color: #c8d6e5; }
  .chat-time { float: right; font-size: 0.6rem; color: #3a5a6a; }
  .chat-input-row { display: flex; gap: 0.35rem; }
  .chat-input {
    flex: 1; padding: 0.4rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 4px; color: #c8d6e5; font-size: 0.75rem;
  }
  .btn-chat-send {
    padding: 0.4rem 0.75rem; background: #2a4a6a; border: none;
    border-radius: 4px; color: #a8d4e8; font-size: 0.7rem; cursor: pointer;
  }
  .btn-chat-send:hover { background: #3a5a7a; }

  .msg-list { max-height: 300px; overflow-y: auto; }
  .msg-card {
    padding: 0.5rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 6px; margin-bottom: 0.4rem; cursor: pointer; font-size: 0.75rem;
  }
  .msg-card:hover { background: #1e2a4a; }
  .msg-sender { display: block; font-weight: 600; color: #8ab5d4; margin-bottom: 0.15rem; }
  .msg-text { display: block; color: #c8d6e5; margin-bottom: 0.15rem; }
  .msg-time { display: block; font-size: 0.6rem; color: #3a5a6a; }
  .system-tag {
    display: inline-block; font-size: 0.6rem; padding: 0.1rem 0.3rem;
    background: #4a2a4a; border-radius: 3px; color: #d4a8d4; margin-top: 0.15rem;
  }
  .empty-msg { font-size: 0.75rem; color: #5a5a6a; text-align: center; padding: 1rem; }

  .compose-form { display: flex; flex-direction: column; gap: 0.5rem; }
  .form-input {
    padding: 0.4rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 4px; color: #c8d6e5; font-size: 0.75rem;
  }
  .form-textarea {
    padding: 0.4rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 4px; color: #c8d6e5; font-size: 0.75rem; min-height: 60px; resize: vertical;
  }

  .alliance-info { text-align: center; font-size: 0.8rem; color: #8a9ab5; }
  .alliance-tag { text-align: center; font-size: 0.7rem; color: #3a5a8a; margin-bottom: 0.75rem; }
  .member-list { max-height: 300px; overflow-y: auto; }
  .member-card {
    display: flex; align-items: center; gap: 0.5rem;
    padding: 0.4rem 0.6rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 6px; margin-bottom: 0.4rem; font-size: 0.75rem;
  }
  .member-id { flex: 1; color: #c8d6e5; }
  .member-role { font-size: 0.65rem; color: #5a7a9a; padding: 0.1rem 0.3rem; background: #0a0e1a; border-radius: 3px; }
  .member-online { font-size: 0.65rem; }
  .btn-action-small {
    padding: 0.2rem 0.4rem; background: #2a3a5a; border: none;
    border-radius: 3px; color: #8ab5d4; font-size: 0.6rem; cursor: pointer;
  }
  .btn-danger { background: #4a2020; color: #d47474; padding: 0.3rem 0.6rem; border: none; border-radius: 4px; cursor: pointer; font-size: 0.7rem; }
  .btn-danger:hover { background: #5a3030; }

  .bulletin-board { max-height: 300px; overflow-y: auto; }
  .bulletin-form { display: flex; flex-direction: column; gap: 0.35rem; margin-bottom: 0.75rem; }
  .bulletin-post {
    padding: 0.5rem; background: #1a2830; border: 1px solid #1a3a4a;
    border-radius: 6px; margin-bottom: 0.4rem; font-size: 0.75rem;
  }
  .bulletin-post strong { color: #74c8d4; display: block; margin-bottom: 0.2rem; }
  .bulletin-post p { color: #c8d6e5; margin-bottom: 0.2rem; }

  .buddy-form { display: flex; gap: 0.35rem; margin-bottom: 0.75rem; }
  .buddy-list { max-height: 300px; overflow-y: auto; }
  .buddy-card {
    display: flex; align-items: center; gap: 0.5rem;
    padding: 0.4rem 0.6rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 6px; margin-bottom: 0.4rem; font-size: 0.75rem;
  }
  .buddy-id { flex: 1; color: #c8d6e5; }
  .buddy-status { font-size: 0.65rem; }

  .rename-row { display: flex; gap: 0.35rem; justify-content: center; margin-bottom: 0.5rem; }
  .rename-input {
    padding: 0.4rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 4px; color: #c8d6e5; font-size: 1rem; text-align: center; max-width: 200px;
  }
  .credits-badge {
    padding: 0.15rem 0.5rem; border-radius: 4px; font-size: 0.8rem;
    background: #2a2a1a; border: 1px solid #5a5a2a; color: #d4d474;
  }

  .header-actions { display: flex; align-items: center; gap: 0.5rem; }
  .notif-bell {
    position: relative; background: none; border: none; font-size: 1.25rem;
    cursor: pointer; padding: 0.25rem; line-height: 1;
  }
  .notif-badge {
    position: absolute; top: -2px; right: -6px; min-width: 16px; height: 16px;
    background: #d4744a; color: #0a0e1a; border-radius: 8px; font-size: 0.6rem;
    font-weight: 700; display: flex; align-items: center; justify-content: center;
    padding: 0 3px;
  }
  .notif-panel {
    background: #141b2d; border: 1px solid #1e2a4a; border-radius: 8px;
    margin-bottom: 1rem; max-height: 350px; overflow: hidden; display: flex; flex-direction: column;
  }
  .notif-panel-header {
    display: flex; justify-content: space-between; align-items: center;
    padding: 0.5rem 0.75rem; border-bottom: 1px solid #1e2a4a;
  }
  .notif-panel-header h3 { font-size: 0.85rem; color: #8a9ab5; margin: 0; }
  .notif-panel-actions { display: flex; align-items: center; gap: 0.5rem; }
  .notif-mark-all {
    padding: 0.2rem 0.5rem; background: #2a4a3a; border: none;
    border-radius: 4px; color: #a8e8a8; font-size: 0.65rem; cursor: pointer;
  }
  .notif-mark-all:hover { background: #3a5a4a; }
  .notif-close {
    background: none; border: none; color: #5a7a9a; font-size: 0.85rem; cursor: pointer;
  }
  .notif-close:hover { color: #c8d6e5; }
  .notif-list {
    overflow-y: auto; flex: 1; max-height: 280px;
  }
  .notif-item {
    padding: 0.5rem 0.75rem; border-bottom: 1px solid #1a2340; cursor: pointer;
    font-size: 0.75rem;
  }
  .notif-item:hover { background: #1a2340; }
  .notif-item.unread { border-left: 3px solid #3a6ab5; }
  .notif-item-header {
    display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.2rem;
  }
  .notif-category {
    font-size: 0.6rem; text-transform: uppercase; letter-spacing: 0.05em;
    padding: 0.1rem 0.3rem; background: #1a2a4a; border-radius: 3px; color: #8ab5d4;
  }
  .notif-time { font-size: 0.6rem; color: #3a5a6a; }
  .notif-title { font-weight: 600; color: #c8d6e5; margin-bottom: 0.15rem; }
  .notif-msg { color: #8a9ab5; }
  .notif-empty { font-size: 0.75rem; color: #5a5a6a; text-align: center; padding: 1rem; }
  .notif-more { font-size: 0.65rem; color: #5a7a9a; text-align: center; padding: 0.5rem; }

  .daily-gift-btn { background: none; border: none; font-size: 1.1rem; cursor: pointer; padding: 0.25rem; line-height: 1; }
  .quests-toggle { background: none; border: none; font-size: 1.1rem; cursor: pointer; padding: 0.25rem; line-height: 1; position: relative; }
  .events-toggle { background: none; border: none; font-size: 1.1rem; cursor: pointer; padding: 0.25rem; line-height: 1; }
  .header-badge {
    position: absolute; top: -2px; right: -8px; min-width: 16px; height: 14px;
    background: #4a6a3a; color: #a8e8a8; border-radius: 7px; font-size: 0.55rem;
    font-weight: 700; display: flex; align-items: center; justify-content: center; padding: 0 3px;
  }

  .daily-gift-popup {
    position: fixed; top: 0; left: 0; right: 0; bottom: 0; z-index: 100;
    background: rgba(0,0,0,0.6); display: flex; align-items: center; justify-content: center;
  }
  .daily-gift-content {
    background: #141b2d; border: 1px solid #2a4a6a; border-radius: 12px;
    padding: 2rem; text-align: center; position: relative; max-width: 300px; width: 90%;
    animation: popupIn 0.3s ease-out;
  }
  @keyframes popupIn { from { transform: scale(0.8); opacity: 0; } to { transform: scale(1); opacity: 1; } }
  .daily-gift-close {
    position: absolute; top: 0.5rem; right: 0.75rem; background: none; border: none;
    color: #5a7a9a; font-size: 1rem; cursor: pointer;
  }
  .daily-gift-close:hover { color: #c8d6e5; }
  .gift-streak { font-size: 1rem; color: #d4d474; margin: 0.5rem 0; }
  .gift-preview { font-size: 0.85rem; color: #8ab5d4; margin-bottom: 1rem; }

  .toast-error {
    position: fixed; top: 1rem; left: 50%; transform: translateX(-50%); z-index: 200;
    background: #4a2020; border: 1px solid #6a3030; border-radius: 8px;
    padding: 0.75rem 1.5rem; color: #d47474; font-size: 0.85rem;
    animation: toastIn 0.3s ease-out;
    max-width: 90%;
  }
  @keyframes toastIn { from { opacity: 0; transform: translateX(-50%) translateY(-20px); } to { opacity: 1; transform: translateX(-50%) translateY(0); } }

  .quests-section { margin-top: 1.5rem; text-align: left; }
  .quests-section h3 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.5rem; text-align: center; }
  .quests-dm { text-align: center; font-size: 0.8rem; color: #d4d474; margin-bottom: 0.75rem; }
  .quest-list { max-height: 400px; overflow-y: auto; display: flex; flex-direction: column; gap: 0.5rem; }
  .quest-card {
    padding: 0.5rem 0.75rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 6px; font-size: 0.8rem; transition: all 0.2s;
  }
  .quest-card.completed { border-color: #2a5a3a; background: #1a2a1a; }
  .quest-card.claimed { opacity: 0.6; }
  .quest-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.25rem; }
  .quest-name { font-weight: 600; color: #c8d6e5; }
  .quest-category { font-size: 0.6rem; padding: 0.1rem 0.3rem; background: #1a2a4a; border-radius: 3px; color: #8ab5d4; text-transform: uppercase; }
  .quest-desc { font-size: 0.75rem; color: #8a9ab5; margin-bottom: 0.5rem; }
  .quest-progress { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.5rem; }
  .quest-bar-bg { flex: 1; height: 6px; background: #0a0e1a; border-radius: 3px; overflow: hidden; }
  .quest-bar-fill { height: 100%; background: #3a6ab5; border-radius: 3px; transition: width 0.5s; }
  .quest-progress-text { font-size: 0.65rem; color: #5a7a9a; min-width: 50px; text-align: right; }
  .quest-reward { display: flex; gap: 0.5rem; font-size: 0.7rem; margin-bottom: 0.35rem; }
  .reward-dm { color: #d4d474; }
  .reward-metal { color: #d4a574; }
  .reward-crystal { color: #74a8d4; }
  .reward-gas { color: #74d4a8; }
  .btn-claim-quest { display: block; width: 100%; }
  .claimed-badge { display: block; text-align: center; font-size: 0.7rem; color: #5aaa5a; }
  .quest-status { display: block; text-align: center; font-size: 0.7rem; color: #5a7a9a; text-transform: capitalize; }

  .events-section { margin-top: 1.5rem; text-align: left; }
  .events-section h3 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-align: center; }
  .event-list { display: flex; flex-direction: column; gap: 0.5rem; max-height: 400px; overflow-y: auto; }
  .event-card {
    padding: 0.5rem 0.75rem; background: #1a2340; border: 1px solid #243050;
    border-radius: 6px; font-size: 0.8rem; transition: all 0.2s;
  }
  .event-card.active { border-color: #2a5a3a; background: #1a2a1a; }
  .event-card.ended { opacity: 0.5; }
  .event-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.25rem; }
  .event-name { font-weight: 600; color: #c8d6e5; }
  .event-type-badge { font-size: 0.6rem; padding: 0.1rem 0.3rem; background: #2a3a1a; border-radius: 3px; color: #8ad474; text-transform: uppercase; }
  .event-desc { font-size: 0.75rem; color: #8a9ab5; margin-bottom: 0.35rem; }
  .event-modifiers { display: flex; flex-wrap: wrap; gap: 0.3rem; margin-bottom: 0.35rem; }
  .modifier { font-size: 0.65rem; padding: 0.1rem 0.3rem; background: #1a2a3a; border-radius: 3px; color: #74a8c8; }
  .event-timer { font-size: 0.75rem; margin-bottom: 0.35rem; }
  .timer-label { color: #5a7a9a; }
  .timer-value { font-family: monospace; color: #d4d474; }
  .event-actions { display: flex; gap: 0.35rem; }
  .joined-badge { font-size: 0.7rem; color: #5aaa5a; }

  .storage-bar .fill { transition: width 0.5s ease; }

  .building { transition: opacity 0.3s, border-color 0.3s; }

  .notif-panel { animation: slideIn 0.2s ease-out; }
  @keyframes slideIn { from { opacity: 0; max-height: 0; } to { opacity: 1; max-height: 350px; } }
  .store-toggle { background: none; border: none; font-size: 1.1rem; cursor: pointer; padding: 0.25rem; line-height: 1; }
  .store-section { margin-top: 1.5rem; text-align: left; }
  .store-section h3 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-align: center; }
  .store-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem; }
  .store-card { padding: 0.5rem; background: #1a2340; border: 1px solid #243050; border-radius: 6px; font-size: 0.8rem; }
  .store-name { font-weight: 600; color: #c8d6e5; margin-bottom: 0.25rem; }
  .store-desc { font-size: 0.7rem; color: #8a9ab5; margin-bottom: 0.5rem; }
  .store-price { display: flex; gap: 0.5rem; margin-bottom: 0.5rem; }
  .price-dm { font-size: 0.75rem; color: #d4d474; }
  .price-credits { font-size: 0.75rem; color: #d4a574; }
  .btn-store-buy { width: 100%; padding: 0.3rem; background: #2a4a3a; border: none; border-radius: 4px; color: #a8e8a8; font-size: 0.75rem; cursor: pointer; }
  .btn-store-buy:hover { background: #3a5a4a; }
  .store-message { padding: 0.4rem; background: #1a2a1a; border: 1px solid #2a4a2a; border-radius: 4px; color: #8ad48a; font-size: 0.75rem; text-align: center; margin-bottom: 0.5rem; }

  .admin-toggle { padding: 0.3rem 0.6rem; background: #3a1a1a; border: 1px solid #5a2a2a; border-radius: 4px; color: #d47474; font-size: 0.7rem; cursor: pointer; }
  .admin-toggle:hover { background: #4a2a2a; }
  .admin-section { margin-top: 1.5rem; text-align: left; }
  .admin-section h3 { font-size: 0.9rem; color: #8a9ab5; margin-bottom: 0.75rem; text-align: center; }
  .admin-message { padding: 0.4rem; background: #1a2a1a; border: 1px solid #2a4a2a; border-radius: 4px; color: #8ad48a; font-size: 0.75rem; text-align: center; margin-bottom: 0.5rem; }
  .admin-sections { display: flex; flex-direction: column; gap: 0.75rem; max-height: 400px; overflow-y: auto; }
  .admin-card { padding: 0.75rem; background: #1a2340; border: 1px solid #243050; border-radius: 6px; font-size: 0.8rem; }
  .admin-card h4 { font-size: 0.85rem; color: #8ab5d4; margin-bottom: 0.5rem; }
  .admin-card input, .admin-card textarea { margin-bottom: 0.35rem; }
  .btn-admin { padding: 0.3rem 0.6rem; background: #2a3a5a; border: none; border-radius: 4px; color: #8ab5d4; font-size: 0.75rem; cursor: pointer; }
  .btn-admin:hover { background: #3a4a6a; }
  .admin-result { font-size: 0.65rem; color: #8a9ab5; background: #0a0e1a; padding: 0.5rem; border-radius: 4px; overflow-x: auto; margin-top: 0.5rem; }

  .tutorial-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; z-index: 100; background: rgba(0,0,0,0.7); display: flex; align-items: center; justify-content: center; }
  .tutorial-card { background: #141b2d; border: 1px solid #2a4a6a; border-radius: 12px; padding: 2rem; text-align: center; max-width: 320px; width: 90%; animation: popupIn 0.3s ease-out; }
  .tutorial-card h3 { font-size: 1.1rem; color: #e8eef5; margin-bottom: 0.5rem; }
  .tutorial-step { font-size: 0.75rem; color: #5a7a9a; margin-bottom: 0.75rem; }
  .tutorial-title { font-size: 1rem; color: #d4d474; margin-bottom: 0.5rem; }
  .tutorial-desc { font-size: 0.85rem; color: #8a9ab5; margin-bottom: 1rem; }
  .tutorial-actions { display: flex; gap: 0.5rem; justify-content: center; }
  .btn-tutorial-claim { padding: 0.5rem 1rem; background: #2a5a3a; border: none; border-radius: 6px; color: #a8e8a8; font-size: 0.85rem; cursor: pointer; }
  .btn-tutorial-claim:hover { background: #3a6a4a; }
  .btn-tutorial-skip { padding: 0.5rem 1rem; background: #4a2020; border: none; border-radius: 6px; color: #d47474; font-size: 0.85rem; cursor: pointer; }
  .btn-tutorial-skip:hover { background: #5a3030; }

  .gem-section { margin-top: 1.5rem; }
  .gem-section h2 { font-size: 0.9rem; color: #b074d4; margin-bottom: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; }
  .gem-slots { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.5rem; }
  .gem-slot { padding: 0.5rem; background: #1a1a30; border: 1px solid #2a2040; border-radius: 6px; text-align: center; font-size: 0.75rem; }
  .gem-slot.filled { border-color: #4a2a7a; background: #2a1a4a; }
  .gem-index { display: block; font-size: 0.65rem; color: #5a5a7a; margin-bottom: 0.25rem; }
  .gem-type { display: block; color: #c8a8e8; font-weight: 600; }
  .gem-stars { display: block; color: #d4d474; font-size: 0.7rem; margin-bottom: 0.25rem; }
  .gem-empty { display: block; color: #5a5a6a; font-style: italic; margin-bottom: 0.25rem; }
  .gem-actions { display: flex; gap: 0.25rem; justify-content: center; }
  .btn-gem-unequip { padding: 0.2rem 0.4rem; background: #4a2a2a; border: none; border-radius: 3px; color: #d47474; font-size: 0.65rem; cursor: pointer; }
  .btn-gem-unequip:hover { background: #5a3a3a; }
  .btn-gem-combine { padding: 0.2rem 0.4rem; background: #2a3a5a; border: none; border-radius: 3px; color: #8ab5d4; font-size: 0.65rem; cursor: pointer; }
  .btn-gem-combine:hover { background: #3a4a6a; }
  .btn-gem-equip { padding: 0.2rem 0.5rem; background: #2a4a3a; border: none; border-radius: 3px; color: #a8e8a8; font-size: 0.65rem; cursor: pointer; }
  .btn-gem-equip:hover { background: #3a5a4a; }

  .discoverer-badge { padding: 0.15rem 0.5rem; border-radius: 4px; font-size: 0.8rem; background: #1a2a3a; border: 1px solid #2a4a6a; color: #74a8d4; }
  .btn-discoverer-upgrade { padding: 0.15rem 0.4rem; background: #2a4a3a; border: none; border-radius: 4px; color: #a8e8a8; font-size: 0.7rem; cursor: pointer; }
  .btn-discoverer-upgrade:hover { background: #3a5a4a; }

  .npc-badge { font-size: 0.6rem; padding: 0.05rem 0.25rem; background: #4a2a2a; border-radius: 3px; color: #d47474; margin-left: 0.2rem; }
</style>
