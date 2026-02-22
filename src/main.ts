import './style.css'
import { Simulation } from './sim/simulation'
import type { TickStats } from './sim/types'

const app = document.querySelector<HTMLDivElement>('#app')
if (!app) {
  throw new Error('#app root not found')
}

app.innerHTML = `
  <main class="layout">
    <header class="header">
      <h1>Grid Simulation MVP</h1>
      <p>Ilk adim: grid uzerinde hareket eden ajanlar.</p>
    </header>

    <section class="controls">
      <label>
        Width
        <input id="width-input" type="number" min="5" max="180" value="60" />
      </label>
      <label>
        Height
        <input id="height-input" type="number" min="5" max="120" value="40" />
      </label>
      <label>
        Agents
        <input id="agents-input" type="number" min="1" max="20000" value="650" />
      </label>
      <label>
        Tick (ms)
        <input id="speed-input" type="range" min="30" max="500" step="10" value="90" />
      </label>
      <div class="buttons">
        <button id="toggle-btn" type="button">Start</button>
        <button id="step-btn" type="button">Step</button>
        <button id="reset-btn" type="button">Reset</button>
      </div>
    </section>

    <section class="stats" aria-live="polite">
      <span>Tick: <strong id="tick-value">0</strong></span>
      <span>Moved: <strong id="moved-value">0</strong></span>
      <span>Occupancy: <strong id="occupancy-value">0%</strong></span>
      <span>Interval: <strong id="interval-value">90 ms</strong></span>
    </section>

    <section class="canvas-wrap">
      <canvas id="grid-canvas" aria-label="Grid simulation canvas"></canvas>
    </section>
  </main>
`

const widthInput = must<HTMLInputElement>('#width-input')
const heightInput = must<HTMLInputElement>('#height-input')
const agentsInput = must<HTMLInputElement>('#agents-input')
const speedInput = must<HTMLInputElement>('#speed-input')

const toggleBtn = must<HTMLButtonElement>('#toggle-btn')
const stepBtn = must<HTMLButtonElement>('#step-btn')
const resetBtn = must<HTMLButtonElement>('#reset-btn')

const tickValue = must<HTMLElement>('#tick-value')
const movedValue = must<HTMLElement>('#moved-value')
const occupancyValue = must<HTMLElement>('#occupancy-value')
const intervalValue = must<HTMLElement>('#interval-value')

const canvas = must<HTMLCanvasElement>('#grid-canvas')
const ctx = mustContext(canvas)

let simulation = createSimulation()
let timerId: number | null = null
let lastStats: TickStats = {
  tick: 0,
  movedAgents: 0,
  occupancy: simulation.config.agentCount / (simulation.config.width * simulation.config.height),
}

render(lastStats)
updateStats(lastStats)

toggleBtn.addEventListener('click', () => {
  if (timerId === null) {
    startLoop()
    return
  }
  stopLoop()
})

stepBtn.addEventListener('click', () => {
  if (timerId !== null) {
    return
  }
  advanceTick()
})

resetBtn.addEventListener('click', () => {
  stopLoop()
  simulation = createSimulation()
  lastStats = {
    tick: 0,
    movedAgents: 0,
    occupancy: simulation.config.agentCount / (simulation.config.width * simulation.config.height),
  }
  render(lastStats)
  updateStats(lastStats)
})

speedInput.addEventListener('input', () => {
  intervalValue.textContent = `${speedMs()} ms`
  if (timerId !== null) {
    stopLoop()
    startLoop()
  }
})

window.addEventListener('resize', () => {
  render(lastStats)
})

function startLoop(): void {
  if (timerId !== null) {
    return
  }
  timerId = window.setInterval(advanceTick, speedMs())
  toggleBtn.textContent = 'Pause'
  stepBtn.disabled = true
}

function stopLoop(): void {
  if (timerId === null) {
    return
  }
  window.clearInterval(timerId)
  timerId = null
  toggleBtn.textContent = 'Start'
  stepBtn.disabled = false
}

function advanceTick(): void {
  lastStats = simulation.step()
  render(lastStats)
  updateStats(lastStats)
}

function createSimulation(): Simulation {
  const width = parseIntSafe(widthInput.value, 5, 180, 60)
  const height = parseIntSafe(heightInput.value, 5, 120, 40)
  const maxAgents = width * height
  const requestedAgents = parseIntSafe(agentsInput.value, 1, 20000, 650)
  const agentCount = Math.min(requestedAgents, maxAgents)

  widthInput.value = String(width)
  heightInput.value = String(height)
  agentsInput.value = String(agentCount)

  return new Simulation(
    {
      width,
      height,
      agentCount,
    },
    Date.now() & 0xffffffff,
  )
}

function render(stats: TickStats): void {
  const { width, height } = simulation.config
  const { cellSize, canvasWidth, canvasHeight } = fitCanvas(width, height)

  ctx.fillStyle = '#f3f0e6'
  ctx.fillRect(0, 0, canvasWidth, canvasHeight)

  ctx.strokeStyle = 'rgba(30, 23, 17, 0.12)'
  ctx.lineWidth = 1
  for (let x = 0; x <= width; x += 1) {
    const px = x * cellSize + 0.5
    ctx.beginPath()
    ctx.moveTo(px, 0)
    ctx.lineTo(px, canvasHeight)
    ctx.stroke()
  }
  for (let y = 0; y <= height; y += 1) {
    const py = y * cellSize + 0.5
    ctx.beginPath()
    ctx.moveTo(0, py)
    ctx.lineTo(canvasWidth, py)
    ctx.stroke()
  }

  ctx.fillStyle = '#14532d'
  for (const agentId of simulation.agentIds) {
    const pos = simulation.getAgentPosition(agentId)
    ctx.fillRect(pos.x * cellSize + 1, pos.y * cellSize + 1, cellSize - 2, cellSize - 2)
  }

  ctx.fillStyle = '#1e1711'
  ctx.font = 'bold 13px ui-monospace, SFMono-Regular, Menlo, monospace'
  ctx.fillText(`tick ${stats.tick}`, 8, 16)
}

function fitCanvas(gridWidth: number, gridHeight: number): {
  cellSize: number
  canvasWidth: number
  canvasHeight: number
} {
  const wrapper = must<HTMLElement>('.canvas-wrap')
  const maxWidth = Math.max(320, wrapper.clientWidth - 16)
  const maxHeight = Math.max(220, window.innerHeight - 320)
  const cellSize = Math.max(6, Math.floor(Math.min(maxWidth / gridWidth, maxHeight / gridHeight)))

  const canvasWidth = gridWidth * cellSize
  const canvasHeight = gridHeight * cellSize
  const dpr = window.devicePixelRatio || 1

  canvas.style.width = `${canvasWidth}px`
  canvas.style.height = `${canvasHeight}px`
  canvas.width = Math.floor(canvasWidth * dpr)
  canvas.height = Math.floor(canvasHeight * dpr)
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  return { cellSize, canvasWidth, canvasHeight }
}

function updateStats(stats: TickStats): void {
  tickValue.textContent = String(stats.tick)
  movedValue.textContent = String(stats.movedAgents)
  occupancyValue.textContent = `${(stats.occupancy * 100).toFixed(1)}%`
  intervalValue.textContent = `${speedMs()} ms`
}

function speedMs(): number {
  return parseIntSafe(speedInput.value, 30, 500, 90)
}

function parseIntSafe(value: string, min: number, max: number, fallback: number): number {
  const parsed = Number.parseInt(value, 10)
  if (!Number.isFinite(parsed)) {
    return fallback
  }
  return Math.max(min, Math.min(max, parsed))
}

function must<T extends Element>(selector: string): T {
  const element = document.querySelector<T>(selector)
  if (!element) {
    throw new Error(`Element not found: ${selector}`)
  }
  return element
}

function mustContext(node: HTMLCanvasElement): CanvasRenderingContext2D {
  const context = node.getContext('2d')
  if (!context) {
    throw new Error('2D context not available')
  }
  return context
}
