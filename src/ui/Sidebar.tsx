import { type RefObject, useState } from 'react'
import type { SimConfig } from '../hooks/useSimulation'
import type { TickStats } from '../sim/types'
import { Accordion } from './Accordion'
import { SparkLine } from './charts/SparkLine'

type Props = {
  config: SimConfig
  onConfig: (update: Partial<SimConfig>) => void
  stats: TickStats
  statsHistory: TickStats[]
  running: boolean
  onToggle: () => void
  onStep: () => void
  onReset: () => void
  miniRef: RefObject<HTMLCanvasElement | null>
}

type Tab = 'config' | 'charts'

export function Sidebar({ config, onConfig, stats, statsHistory, running, onToggle, onStep, onReset, miniRef }: Props) {
  const [tab, setTab] = useState<Tab>('config')

  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <h1>Omega</h1>
      </div>

      <div className="tab-bar">
        <button className="tab-btn" data-active={tab === 'config'} onClick={() => setTab('config')}>Config</button>
        <button className="tab-btn" data-active={tab === 'charts'} onClick={() => setTab('charts')}>Charts</button>
      </div>

      {tab === 'config' ? (
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
            <Field label="Max energy">
              <input
                type="number" min={1} max={1200} value={config.maxEnergy}
                onChange={e => onConfig({ maxEnergy: int(e.target.value, 1, 1200) })}
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

          <Accordion title="Trees & Fruit" defaultOpen>
            <Field label="Tree density (0-1)">
              <input
                type="number" min={0} max={1} step={0.01} value={config.treeDensity}
                onChange={e => onConfig({ treeDensity: float(e.target.value, 0, 1) })}
              />
            </Field>
            <Field label="Fruit drop chance / tree / tick">
              <input
                type="number" min={0} max={1} step={0.01} value={config.fruitDropPerTree}
                onChange={e => onConfig({ fruitDropPerTree: float(e.target.value, 0, 1) })}
              />
            </Field>
            <Field label="Max fruit around each tree">
              <input
                type="number" min={1} max={4} step={1} value={config.maxFruitAroundTree}
                onChange={e => onConfig({ maxFruitAroundTree: int(e.target.value, 1, 4) })}
              />
            </Field>
            <Field label="Inventory cap">
              <input
                type="number" min={1} max={300} step={1} value={config.maxInventory}
                onChange={e => onConfig({ maxInventory: int(e.target.value, 1, 300) })}
              />
            </Field>
            <Field label="Fruit energy gain">
              <input
                type="number" min={0.1} max={100} step={0.1} value={config.fruitEnergyGain}
                onChange={e => onConfig({ fruitEnergyGain: float(e.target.value, 0.1, 100) })}
              />
            </Field>
            <Field label="Trade amount">
              <input
                type="number" min={1} max={20} step={1} value={config.tradeAmount}
                onChange={e => onConfig({ tradeAmount: int(e.target.value, 1, 20) })}
              />
            </Field>
          </Accordion>
        </div>
      ) : (
        <div className="sidebar-charts">
          <SparkSection label={`Population — ${stats.aliveAgents}`}>
            <SparkLine
              data={statsHistory}
              series={[{ key: 'aliveAgents', color: '#171717' }]}
              height={36}
            />
          </SparkSection>

          <SparkSection label={`Avg Energy — ${stats.avgEnergy.toFixed(1)}`}>
            <SparkLine
              data={statsHistory}
              series={[{ key: 'avgEnergy', color: '#f59e0b' }]}
              height={36}
            />
          </SparkSection>

          <SparkSection label={`Deaths / tick — ${stats.deathsThisTick}`}>
            <SparkLine
              data={statsHistory}
              series={[{ key: 'deathsThisTick', color: '#e11d48' }]}
              height={36}
            />
          </SparkSection>

          <SparkSection label="Fruit Economy">
            <SparkLine
              data={statsHistory}
              series={[
                { key: 'collectedFruit', color: '#d97706', label: 'Collect' },
                { key: 'eatenFruit', color: '#16a34a', label: 'Eat' },
                { key: 'tradedFruit', color: '#2563eb', label: 'Trade' },
              ]}
              height={36}
            />
          </SparkSection>

          <SparkSection label="Vitamins">
            <SparkLine
              data={statsHistory}
              series={[
                { key: 'avgVitaminA', color: '#e11d48', label: 'A' },
                { key: 'avgVitaminB', color: '#f59e0b', label: 'B' },
                { key: 'avgVitaminC', color: '#2563eb', label: 'C' },
              ]}
              height={36}
            />
          </SparkSection>

          <SparkSection label={`Ground Fruit — ${stats.groundFruitTotal.toFixed(0)}`}>
            <SparkLine
              data={statsHistory}
              series={[{ key: 'groundFruitTotal', color: '#16a34a' }]}
              height={36}
            />
          </SparkSection>

          <SparkSection label={`Deficient — ${stats.deficientAgents}`}>
            <SparkLine
              data={statsHistory}
              series={[{ key: 'deficientAgents', color: '#dc2626' }]}
              height={36}
            />
          </SparkSection>
        </div>
      )}

      <div className="minimap-section">
        <span className="section-label">Map</span>
        <canvas ref={miniRef} width={180} height={110} className="minimap-canvas" />
      </div>

      <div className="sidebar-footer">
        <div className="stats-bar">
          <span>tick <strong className="hl">{stats.tick}</strong></span>
          <span>alive <strong>{stats.aliveAgents}</strong></span>
          <span>avg E <strong>{stats.avgEnergy.toFixed(1)}</strong></span>
          <span>density <strong>{(stats.occupancy * 100).toFixed(1)}%</strong></span>
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

// ── Small helpers ──

function SparkSection({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="spark-section">
      <span className="spark-label">{label}</span>
      {children}
    </div>
  )
}

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
