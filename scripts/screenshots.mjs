import { chromium } from 'playwright'
import { fileURLToPath } from 'url'
import { dirname, join } from 'path'
import { mkdirSync, existsSync } from 'fs'

// Walk up from this file until we find the repo root (docker-compose.yml),
// so screenshots always land in <repo>/docs/screenshots no matter where the
// script is run/copied to.
const findRepoRoot = (start) => {
  let dir = start
  while (dir !== dirname(dir)) {
    if (existsSync(join(dir, 'docker-compose.yml'))) return dir
    dir = dirname(dir)
  }
  return start
}
const repoRoot = findRepoRoot(dirname(fileURLToPath(import.meta.url)))
const outDir = join(repoRoot, 'docs', 'screenshots')
mkdirSync(outDir, { recursive: true })

const BASE = process.env.BASE_URL || 'http://localhost'
const email = `commander_${Date.now()}@galaxy.test`
const password = 'galaxy12345'

const shot = async (page, name, opts = {}) => {
  await page.screenshot({ path: join(outDir, name), ...opts })
  console.log('saved', name)
}

// Click a nav tile, then screenshot just the section it reveals.
const feature = async (page, navSelector, sectionSelector, file) => {
  const btn = page.locator(navSelector).first()
  if (!(await btn.count())) { console.log('  no nav button:', navSelector); return }
  await btn.click()
  const section = page.locator(sectionSelector).first()
  try {
    await section.waitFor({ state: 'visible', timeout: 8000 })
    await page.waitForTimeout(1200)
    await section.screenshot({ path: join(outDir, file) })
    console.log('saved', file)
  } catch { console.log('  section never appeared:', sectionSelector) }
}

const run = async () => {
  const browser = await chromium.launch()
  const page = await browser.newPage({ viewport: { width: 460, height: 950 }, deviceScaleFactor: 2 })

  // 1. Auth / login screen
  await page.goto(BASE, { waitUntil: 'networkidle' })
  await page.waitForTimeout(600)
  await shot(page, '01-login.png')

  // Register a fresh commander
  await page.getByText('Create Account', { exact: false }).click()
  await page.waitForTimeout(300)
  await page.locator('input[type="email"]').fill(email)
  await page.locator('input[type="password"]').fill(password)
  await page.getByRole('button', { name: 'Register' }).click()

  // 2. Planet dashboard — wait for resources to render
  await page.waitForSelector('text=METAL', { timeout: 15000 })
  await page.waitForTimeout(1500)
  await page.evaluate(() => window.scrollTo(0, 0))
  await shot(page, '02-dashboard.png')                 // top: resources + buildings
  await shot(page, '03-overview-full.png', { fullPage: true })

  // 3. Feature panels via nav tiles (element screenshots of each section)
  await feature(page, '.galaxy-toggle', '.galaxy-section', '04-galaxy-map.png')
  await feature(page, '.shipyard-toggle', '.shipyard-section', '05-shipyard.png')
  await feature(page, '.fleet-toggle', '.fleet-section', '06-fleet.png')

  await browser.close()
  console.log('done ->', outDir)
}

run().catch(e => { console.error(e); process.exit(1) })
