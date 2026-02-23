import type { SimConfig } from '../hooks/useSimulation'
import type { TickStats } from '../sim/types'
import { SparkLine } from './charts/SparkLine'
import { TimeSeriesChart } from './charts/TimeSeriesChart'
import { ActionBarChart } from './charts/ActionBarChart'

type Props = {
  section: string
  config: SimConfig
  onConfig: (update: Partial<SimConfig>) => void
  stats: TickStats
  statsHistory: TickStats[]
  running: boolean
  onToggle: () => void
  onStep: () => void
  onReset: () => void
}

export function ContentPanel({ section, config, onConfig, stats, statsHistory, running, onToggle, onStep, onReset }: Props) {
  return (
    <div className="content-panel">
      <div className="cp-body">
        {section === 'overview' && (
          <OverviewContent
            config={config}
            onConfig={onConfig}
            stats={stats}
            statsHistory={statsHistory}
            running={running}
            onToggle={onToggle}
            onStep={onStep}
            onReset={onReset}
          />
        )}
        {section === 'population' && (
          <PopulationContent stats={stats} history={statsHistory} />
        )}
        {section === 'economy' && (
          <EconomyContent stats={stats} history={statsHistory} />
        )}
        {section === 'config' && (
          <ConfigContent config={config} onConfig={onConfig} />
        )}
      </div>
    </div>
  )
}

// ── Overview ──

function OverviewContent({ config, onConfig, stats, statsHistory, running, onToggle, onStep, onReset }: {
  config: SimConfig
  onConfig: (u: Partial<SimConfig>) => void
  stats: TickStats
  statsHistory: TickStats[]
  running: boolean
  onToggle: () => void
  onStep: () => void
  onReset: () => void
}) {
  return (
    <>
      <div className="btn-row">
        <button className="btn btn-primary" onClick={onToggle}>
          {running ? 'Pause' : 'Start'}
        </button>
        <button className="btn" onClick={onStep} disabled={running}>Step</button>
        <button className="btn" onClick={onReset}>Reset</button>
      </div>

      <div className="inline-field">
        <span className="inline-label">Speed</span>
        <div className="inline-slider">
          <input type="range" min={16} max={500} step={1} value={config.speed}
            onChange={e => onConfig({ speed: Number(e.target.value) })} />
          <span className="inline-hint">{config.speed}ms</span>
        </div>
      </div>

      <div className="cp-stats-grid">
        <StatCard label="Tick" value={String(stats.tick)} />
        <StatCard label="Alive" value={String(stats.aliveAgents)} />
        <StatCard label="Deaths" value={String(stats.deathsThisTick)} />
        <StatCard label="Avg Energy" value={stats.avgEnergy.toFixed(1)} />
        <StatCard label="Density" value={`${(stats.occupancy * 100).toFixed(1)}%`} />
        <StatCard label="Deficient" value={String(stats.deficientAgents)} />
      </div>

      <SparkSection label={`Population — ${stats.aliveAgents}`}>
        <SparkLine data={statsHistory} series={[{ key: 'aliveAgents', color: '#171717' }]} height={48} />
      </SparkSection>

      <SparkSection label={`Avg Energy — ${stats.avgEnergy.toFixed(1)}`}>
        <SparkLine data={statsHistory} series={[{ key: 'avgEnergy', color: '#f59e0b' }]} height={48} />
      </SparkSection>
    </>
  )
}

// ── Population ──

function PopulationContent({ stats, history }: { stats: TickStats; history: TickStats[] }) {
  return (
    <>
      <div className="cp-stats-grid">
        <StatCard label="Alive" value={String(stats.aliveAgents)} />
        <StatCard label="Deaths/tick" value={String(stats.deathsThisTick)} />
        <StatCard label="Density" value={`${(stats.occupancy * 100).toFixed(1)}%`} />
        <StatCard label="Deficient" value={String(stats.deficientAgents)} />
      </div>

      <TimeSeriesChart title="Population" data={history}
        series={[{ key: 'aliveAgents', color: '#171717', label: 'Alive' }]} height={180} />

      <TimeSeriesChart title="Deaths per tick" data={history}
        series={[{ key: 'deathsThisTick', color: '#e11d48', label: 'Deaths', type: 'area' }]} height={140} />

      <TimeSeriesChart title="Vitamins" data={history}
        series={[
          { key: 'avgVitaminA', color: '#e11d48', label: 'Vit A' },
          { key: 'avgVitaminB', color: '#f59e0b', label: 'Vit B' },
          { key: 'avgVitaminC', color: '#2563eb', label: 'Vit C' },
        ]} height={160} />

      <TimeSeriesChart title="Energy" data={history}
        series={[{ key: 'avgEnergy', color: '#f59e0b', label: 'Avg Energy' }]} height={140} />

      <ActionBarChart histogram={stats.actionHistogram} height={140} />
    </>
  )
}

// ── Economy ──

function EconomyContent({ stats, history }: { stats: TickStats; history: TickStats[] }) {
  return (
    <>
      <div className="cp-stats-grid">
        <StatCard label="Collected" value={stats.collectedFruit.toFixed(0)} />
        <StatCard label="Eaten" value={stats.eatenFruit.toFixed(0)} />
        <StatCard label="Traded" value={stats.tradedFruit.toFixed(0)} />
        <StatCard label="Ground" value={stats.groundFruitTotal.toFixed(0)} />
        <StatCard label="Avg Inv" value={stats.avgInventory.toFixed(1)} />
      </div>

      <TimeSeriesChart title="Fruit Economy" data={history} stacked height={180}
        series={[
          { key: 'collectedFruit', color: '#d97706', label: 'Collected' },
          { key: 'eatenFruit', color: '#16a34a', label: 'Eaten' },
          { key: 'tradedFruit', color: '#2563eb', label: 'Traded' },
        ]} />

      <TimeSeriesChart title="Ground Fruit" data={history} stacked height={160}
        series={[
          { key: 'groundApple', color: '#e11d48', label: 'Apple' },
          { key: 'groundBanana', color: '#facc15', label: 'Banana' },
          { key: 'groundOrange', color: '#f97316', label: 'Orange' },
        ]} />

      <TimeSeriesChart title="Avg Inventory" data={history}
        series={[{ key: 'avgInventory', color: '#8b5cf6', label: 'Avg Inventory' }]} height={140} />
    </>
  )
}

// ── Config ──

function ConfigContent({ config, onConfig }: { config: SimConfig; onConfig: (u: Partial<SimConfig>) => void }) {
  return (
    <>
      <Group title="Simulation">
        <InlineField label="Grid W" value={config.width} onChange={v => onConfig({ width: v })} min={8} max={200} />
        <InlineField label="Grid H" value={config.height} onChange={v => onConfig({ height: v })} min={8} max={200} />
        <InlineField label="Agents" value={config.agents} onChange={v => onConfig({ agents: v })} min={1} max={30000} />
        <InlineField label="Max slots" value={config.maxAgentSlots} onChange={v => onConfig({ maxAgentSlots: v })} min={1} max={50000} />
        <div className="inline-field">
          <span className="inline-label">Speed</span>
          <div className="inline-slider">
            <input type="range" min={16} max={500} step={1} value={config.speed}
              onChange={e => onConfig({ speed: Number(e.target.value) })} />
            <span className="inline-hint">{config.speed}ms</span>
          </div>
        </div>
      </Group>

      <Group title="Energy">
        <InlineField label="Initial" value={config.initialEnergy} onChange={v => onConfig({ initialEnergy: v })} min={1} max={500} />
        <InlineField label="Max" value={config.maxEnergy} onChange={v => onConfig({ maxEnergy: v })} min={1} max={1200} />
        <InlineField label="Move cost" value={config.moveEnergyCost} onChange={v => onConfig({ moveEnergyCost: v })} min={0.05} max={20} step={0.05} />
        <InlineField label="Idle cost" value={config.idleEnergyCost} onChange={v => onConfig({ idleEnergyCost: v })} min={0.01} max={10} step={0.01} />
        <InlineField label="Neuron cost" value={config.neuronStepEnergyCost} onChange={v => onConfig({ neuronStepEnergyCost: v })} min={0} max={3} step={0.01} />
      </Group>

      <Group title="Genome & Clone">
        <InlineField label="Min neurons" value={config.minNeuronCount} onChange={v => onConfig({ minNeuronCount: v })} min={1} max={64} />
        <InlineField label="Max neurons" value={config.maxNeuronCount} onChange={v => onConfig({ maxNeuronCount: v })} min={1} max={128} />
        <InlineField label="Clone threshold" value={config.cloneEnergyThreshold} onChange={v => onConfig({ cloneEnergyThreshold: v })} min={1} max={5000} />
        <InlineField label="Clone transfer" value={config.cloneEnergyCost} onChange={v => onConfig({ cloneEnergyCost: v })} min={1} max={2000} />
        <InlineField label="Clone chance" value={config.cloneChancePerTick} onChange={v => onConfig({ cloneChancePerTick: v })} min={0} max={1} step={0.01} />
      </Group>

      <Group title="Trees & Fruit">
        <InlineField label="Tree density" value={config.treeDensity} onChange={v => onConfig({ treeDensity: v })} min={0} max={1} step={0.01} />
        <InlineField label="Fruit drop" value={config.fruitDropPerTree} onChange={v => onConfig({ fruitDropPerTree: v })} min={0} max={1} step={0.01} />
        <InlineField label="Max/tree" value={config.maxFruitAroundTree} onChange={v => onConfig({ maxFruitAroundTree: v })} min={1} max={4} />
        <InlineField label="Inv cap" value={config.maxInventory} onChange={v => onConfig({ maxInventory: v })} min={1} max={300} />
        <InlineField label="E gain" value={config.fruitEnergyGain} onChange={v => onConfig({ fruitEnergyGain: v })} min={0.1} max={100} step={0.1} />
        <InlineField label="Trade amt" value={config.tradeAmount} onChange={v => onConfig({ tradeAmount: v })} min={1} max={20} />
      </Group>
    </>
  )
}

// ── Reusable pieces ──

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="stat-card">
      <span className="stat-card-value">{value}</span>
      <span className="stat-card-label">{label}</span>
    </div>
  )
}

function SparkSection({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="spark-section">
      <span className="spark-label">{label}</span>
      {children}
    </div>
  )
}

function Group({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="cfg-group">
      <span className="cfg-group-title">{title}</span>
      <div className="cfg-group-body">{children}</div>
    </div>
  )
}

function InlineField({ label, value, onChange, min, max, step }: {
  label: string; value: number; onChange: (v: number) => void
  min: number; max: number; step?: number
}) {
  return (
    <div className="inline-field">
      <span className="inline-label">{label}</span>
      <input
        type="number" min={min} max={max} step={step ?? 1} value={value}
        onChange={e => {
          const n = step != null && step < 1 ? Number.parseFloat(e.target.value) : Number.parseInt(e.target.value, 10)
          if (Number.isFinite(n)) onChange(Math.max(min, Math.min(max, n)))
        }}
      />
    </div>
  )
}
