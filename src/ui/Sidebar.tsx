import { type RefObject } from 'react'
import type { SimConfig } from '../hooks/useSimulation'
import type { TickStats } from '../sim/types'
import { Accordion } from './Accordion'

type Props = {
  config: SimConfig
  onConfig: (update: Partial<SimConfig>) => void
  stats: TickStats
  running: boolean
  onToggle: () => void
  onStep: () => void
  onReset: () => void
  miniRef: RefObject<HTMLCanvasElement | null>
}

export function Sidebar({ config, onConfig, stats, running, onToggle, onStep, onReset, miniRef }: Props) {
  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <h1>Omega</h1>
      </div>

      <div className="accordion">
        <Accordion title="Simulation" defaultOpen>
          <Field label="Grid size">
            <div className="field-row">
              <input
                type="number" min={8} max={200} value={config.width}
                onChange={e => onConfig({ width: int(e.target.value, 8, 200) })}
              />
              <span className="field-sep">&times;</span>
              <input
                type="number" min={8} max={200} value={config.height}
                onChange={e => onConfig({ height: int(e.target.value, 8, 200) })}
              />
            </div>
          </Field>
          <Field label="Agent count">
            <input
              type="number" min={1} max={30000} value={config.agents}
              onChange={e => onConfig({ agents: int(e.target.value, 1, 30000) })}
            />
          </Field>
          <Field label={`Tick interval — ${config.speed}ms`}>
            <input
              type="range" min={16} max={500} step={1} value={config.speed}
              onChange={e => onConfig({ speed: Number(e.target.value) })}
            />
          </Field>
        </Accordion>

        <Accordion title="Agent State" defaultOpen>
          <Field label="Initial energy">
            <input
              type="number" min={1} max={500} value={config.initialEnergy}
              onChange={e => onConfig({ initialEnergy: int(e.target.value, 1, 500) })}
            />
          </Field>
          <Field label="Move energy cost">
            <input
              type="number" min={0.05} max={20} step={0.05} value={config.moveEnergyCost}
              onChange={e => onConfig({ moveEnergyCost: float(e.target.value, 0.05, 20) })}
            />
          </Field>
          <Field label="Idle energy cost">
            <input
              type="number" min={0.01} max={10} step={0.01} value={config.idleEnergyCost}
              onChange={e => onConfig({ idleEnergyCost: float(e.target.value, 0.01, 10) })}
            />
          </Field>
        </Accordion>
      </div>

      <div className="minimap-section">
        <span className="section-label">Map</span>
        <canvas ref={miniRef} width={180} height={110} className="minimap-canvas" />
      </div>

      <div className="sidebar-footer">
        <div className="stats-bar">
          <span>tick <strong className="hl">{stats.tick}</strong></span>
          <span>moved <strong>{stats.movedAgents}</strong></span>
          <span>alive <strong>{stats.aliveAgents}</strong></span>
          <span>deaths <strong>{stats.deathsThisTick}</strong></span>
          <span>avg E <strong>{stats.avgEnergy.toFixed(1)}</strong></span>
          <span>density <strong>{(stats.occupancy * 100).toFixed(1)}%</strong></span>
          <span>stay <strong>{stats.actionHistogram.stay}</strong></span>
        </div>
        <div className="btn-row">
          <button className="btn btn-primary" onClick={onToggle}>
            {running ? 'Pause' : 'Start'}
          </button>
          <button className="btn" onClick={onStep} disabled={running}>Step</button>
          <button className="btn" onClick={onReset}>Reset</button>
        </div>
      </div>
    </aside>
  )
}

// ── Reusable field wrapper ──

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="field">
      <span className="field-label">{label}</span>
      {children}
    </div>
  )
}

function int(v: string, min: number, max: number): number {
  const n = Number.parseInt(v, 10)
  if (!Number.isFinite(n)) return min
  return Math.max(min, Math.min(max, n))
}

function float(v: string, min: number, max: number): number {
  const n = Number.parseFloat(v)
  if (!Number.isFinite(n)) return min
  return Math.max(min, Math.min(max, n))
}
