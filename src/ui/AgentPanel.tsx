import type { AgentDetail } from '../hooks/useSimulation'
import type { ActionType } from '../sim/types'

type Props = {
  agentId: number
  detail: AgentDetail | null
  onClose: () => void
}

const ACTION_LABELS: Record<ActionType, string> = {
  stay: 'Stay',
  move_north: 'N',
  move_east: 'E',
  move_south: 'S',
  move_west: 'W',
  collect_fruit: 'Collect',
  eat_fruit: 'Eat',
  trade_fruit: 'Trade',
  clone_self: 'Clone',
}

export function AgentPanel({ agentId, detail, onClose }: Props) {
  if (!detail || detail.id !== agentId) {
    return (
      <aside className="agent-panel">
        <div className="ap-header">
          <span className="ap-title">Agent #{agentId}</span>
          <button className="ap-close" onClick={onClose}>&times;</button>
        </div>
        <div className="ap-body">
          <span className="ap-empty">loading...</span>
        </div>
      </aside>
    )
  }

  const d = detail

  return (
    <aside className="agent-panel">
      <div className="ap-header">
        <span className="ap-title">Agent #{d.id}</span>
        <button className="ap-close" onClick={onClose}>&times;</button>
      </div>

      <div className="ap-body">
        <div className="ap-status" data-alive={d.alive}>
          {d.alive ? 'Alive' : 'Dead'}
        </div>

        <Section label="State">
          <Row k="Position" v={`${d.x}, ${d.y}`} />
          <Row k="Energy" v={d.energy.toFixed(1)} />
          <Row k="Inventory total" v={d.inventory.toFixed(1)} />
          <Row k="Apple" v={d.inventoryByFruit.apple.toFixed(1)} />
          <Row k="Banana" v={d.inventoryByFruit.banana.toFixed(1)} />
          <Row k="Orange" v={d.inventoryByFruit.orange.toFixed(1)} />
          <Row k="Vitamin A" v={d.vitamins.vitamin_a.toFixed(1)} />
          <Row k="Vitamin B" v={d.vitamins.vitamin_b.toFixed(1)} />
          <Row k="Vitamin C" v={d.vitamins.vitamin_c.toFixed(1)} />
          <Row k="Neurons" v={String(d.neuronCount)} />
          <Row k="Neuron step cost" v={d.neuronEnergyCost.toFixed(2)} />
        </Section>

        <Section label="Lifetime">
          <Row k="Born" v={`tick ${d.bornTick}`} />
          {!d.alive && <Row k="Died" v={`tick ${d.deathTick}`} />}
          <Row k="Age" v={`${d.age} ticks`} />
          <Row k="Total actions" v={String(d.totalActions)} />
        </Section>

        <Section label="Action counts">
          {Object.entries(d.actionCounts).map(([action, count]) => (
            <Row key={action} k={ACTION_LABELS[action as ActionType]} v={String(count)} />
          ))}
        </Section>

        <Section label="Genome">
          <Row k="Mutation rate" v={d.genome.mutationRate.toFixed(3)} />
          <Row k="Mutation scale" v={d.genome.mutationScale.toFixed(3)} />
          <Row k="Add neuron chance" v={d.genome.addNeuronChance.toFixed(3)} />
          <Row k="Remove neuron chance" v={d.genome.removeNeuronChance.toFixed(3)} />
          <Row k="Bias apple" v={d.genome.fruitBiasApple.toFixed(2)} />
          <Row k="Bias banana" v={d.genome.fruitBiasBanana.toFixed(2)} />
          <Row k="Bias orange" v={d.genome.fruitBiasOrange.toFixed(2)} />
        </Section>

        <Section label="Recent actions">
          <div className="ap-recent">
            {d.recentActions.map((a: ActionType, i: number) => (
              <span key={i} className="ap-action-tag" data-action={a}>
                {ACTION_LABELS[a]}
              </span>
            ))}
            {d.recentActions.length === 0 && <span className="ap-empty">none</span>}
          </div>
        </Section>
      </div>
    </aside>
  )
}

function Section({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="ap-section">
      <span className="ap-section-label">{label}</span>
      {children}
    </div>
  )
}

function Row({ k, v }: { k: string; v: string }) {
  return (
    <div className="ap-row">
      <span className="ap-key">{k}</span>
      <span className="ap-val">{v}</span>
    </div>
  )
}
