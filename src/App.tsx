import { useCallback, useEffect, useRef, useState } from 'react'
import { Simulation } from './sim/simulation'
import type { TickStats } from './sim/types'

const DEFAULTS = { width: 80, height: 50, agents: 800, speed: 60 }

export function App() {
  const [config, setConfig] = useState(DEFAULTS)
  const [stats, setStats] = useState<TickStats>({ tick: 0, movedAgents: 0, occupancy: 0 })
  const [running, setRunning] = useState(false)

  const simRef = useRef<Simulation | null>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const miniRef = useRef<HTMLCanvasElement>(null)
  const timerRef = useRef<number | null>(null)

  const getSim = useCallback(() => {
    if (!simRef.current) {
      const w = clamp(config.width, 8, 200)
      const h = clamp(config.height, 8, 200)
      const a = Math.min(clamp(config.agents, 1, 30000), w * h)
      simRef.current = new Simulation({ width: w, height: h, agentCount: a }, Date.now() & 0xffffffff)
    }
    return simRef.current
  }, [config])

  const render = useCallback(() => {
    const sim = getSim()
    const canvas = canvasRef.current
    const mini = miniRef.current
    if (!canvas || !mini) return

    const ctx = canvas.getContext('2d')!
    const mctx = mini.getContext('2d')!
    const { width: gw, height: gh } = sim.config
    const parent = canvas.parentElement!
    const cell = Math.max(4, Math.floor(Math.min((parent.clientWidth - 16) / gw, (parent.clientHeight - 16) / gh)))

    const cw = gw * cell
    const ch = gh * cell
    const dpr = devicePixelRatio || 1

    canvas.style.width = `${cw}px`
    canvas.style.height = `${ch}px`
    canvas.width = Math.floor(cw * dpr)
    canvas.height = Math.floor(ch * dpr)
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)

    ctx.fillStyle = '#ede8de'
    ctx.fillRect(0, 0, cw, ch)

    ctx.strokeStyle = 'rgba(26, 39, 68, 0.06)'
    ctx.lineWidth = 1
    for (let x = 0; x <= gw; x++) {
      const px = x * cell + 0.5
      ctx.beginPath(); ctx.moveTo(px, 0); ctx.lineTo(px, ch); ctx.stroke()
    }
    for (let y = 0; y <= gh; y++) {
      const py = y * cell + 0.5
      ctx.beginPath(); ctx.moveTo(0, py); ctx.lineTo(cw, py); ctx.stroke()
    }

    const pad = Math.max(1, Math.floor(cell * 0.12))
    const size = cell - pad * 2
    ctx.fillStyle = '#1a2744'
    for (const id of sim.agentIds) {
      const p = sim.getAgentPosition(id)
      ctx.fillRect(p.x * cell + pad, p.y * cell + pad, size, size)
    }

    // Minimap
    const mw = mini.width
    const mh = mini.height
    mctx.fillStyle = '#ede8de'
    mctx.fillRect(0, 0, mw, mh)
    const sx = mw / gw
    const sy = mh / gh
    mctx.fillStyle = 'rgba(26, 39, 68, 0.6)'
    for (const id of sim.agentIds) {
      const p = sim.getAgentPosition(id)
      mctx.fillRect(Math.floor(p.x * sx), Math.floor(p.y * sy), Math.max(1, Math.ceil(sx)), Math.max(1, Math.ceil(sy)))
    }
  }, [getSim])

  const tick = useCallback(() => {
    const sim = getSim()
    const s = sim.step()
    setStats(s)
    render()
  }, [getSim, render])

  // Start/stop loop
  useEffect(() => {
    if (running) {
      timerRef.current = window.setInterval(tick, config.speed)
      return () => { if (timerRef.current) clearInterval(timerRef.current) }
    }
    if (timerRef.current) { clearInterval(timerRef.current); timerRef.current = null }
  }, [running, config.speed, tick])

  // Initial render + resize
  useEffect(() => { render(); window.addEventListener('resize', render); return () => window.removeEventListener('resize', render) }, [render])

  const reset = () => {
    setRunning(false)
    simRef.current = null
    setStats({ tick: 0, movedAgents: 0, occupancy: 0 })
    requestAnimationFrame(render)
  }

  const step = () => { if (!running) tick() }

  return (
    <div className="layout">
      <aside className="sidebar">
        <h1 className="logo">Omega</h1>

        <Section label="Grid">
          <div className="field-row">
            <NumInput value={config.width} onChange={v => { setConfig(c => ({ ...c, width: v })); simRef.current = null }} min={8} max={200} />
            <span className="sep">&times;</span>
            <NumInput value={config.height} onChange={v => { setConfig(c => ({ ...c, height: v })); simRef.current = null }} min={8} max={200} />
          </div>
        </Section>

        <Section label="Agents">
          <NumInput value={config.agents} onChange={v => { setConfig(c => ({ ...c, agents: v })); simRef.current = null }} min={1} max={30000} />
        </Section>

        <Section label="Speed">
          <input
            type="range" min={16} max={500} step={1} value={config.speed}
            onChange={e => setConfig(c => ({ ...c, speed: Number(e.target.value) }))}
          />
          <span className="field-hint">{config.speed} ms</span>
        </Section>

        <div className="sidebar-buttons">
          <button className="btn btn-primary" onClick={() => setRunning(r => !r)}>
            {running ? 'Pause' : 'Start'}
          </button>
          <button className="btn" onClick={step} disabled={running}>Step</button>
          <button className="btn" onClick={reset}>Reset</button>
        </div>

        <div className="sidebar-stats">
          <Stat label="tick" value={stats.tick} accent />
          <Stat label="agents" value={getSim().config.agentCount} />
          <Stat label="moved" value={stats.movedAgents} />
          <Stat label="density" value={`${(stats.occupancy * 100).toFixed(1)}%`} />
        </div>

        <div className="minimap-wrap">
          <span className="section-label">Map</span>
          <canvas ref={miniRef} width={160} height={100} className="minimap-canvas" />
        </div>
      </aside>

      <main className="viewport">
        <canvas ref={canvasRef} />
      </main>
    </div>
  )
}

// ── Small components ──

function Section({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="section">
      <span className="section-label">{label}</span>
      {children}
    </div>
  )
}

function NumInput({ value, onChange, min, max }: { value: number; onChange: (v: number) => void; min: number; max: number }) {
  return (
    <input
      type="number" min={min} max={max} value={value}
      onChange={e => onChange(clamp(Number(e.target.value), min, max))}
    />
  )
}

function Stat({ label, value, accent }: { label: string; value: string | number; accent?: boolean }) {
  return (
    <div className="stat">
      <span className="stat-label">{label}</span>
      <span className={`stat-value${accent ? ' stat-accent' : ''}`}>{value}</span>
    </div>
  )
}

function clamp(v: number, lo: number, hi: number) {
  if (!Number.isFinite(v)) return lo
  return Math.max(lo, Math.min(hi, v))
}
