import { useState } from 'react'
import type { TickStats } from '../sim/types'
import { TimeSeriesChart } from './charts/TimeSeriesChart'
import { ActionBarChart } from './charts/ActionBarChart'

type Props = {
  stats: TickStats
  history: TickStats[]
}

export function BottomPanel({ stats, history }: Props) {
  const [open, setOpen] = useState(false)

  return (
    <div className="bottom-panel" data-open={open}>
      <button className="bp-toggle" onClick={() => setOpen(o => !o)}>
        <span className="bp-chevron">{open ? '\u25BC' : '\u25B2'}</span>
        <span>Charts</span>
      </button>
      {open && (
        <div className="bp-charts">
          <TimeSeriesChart
            title="Population"
            data={history}
            series={[
              { key: 'aliveAgents', color: '#171717', label: 'Alive' },
              { key: 'deathsThisTick', color: '#e11d48', label: 'Deaths/tick', type: 'area' },
            ]}
          />
          <TimeSeriesChart
            title="Energy"
            data={history}
            series={[
              { key: 'avgEnergy', color: '#f59e0b', label: 'Avg Energy' },
            ]}
          />
          <TimeSeriesChart
            title="Fruit Economy"
            data={history}
            series={[
              { key: 'collectedFruit', color: '#d97706', label: 'Collected' },
              { key: 'eatenFruit', color: '#16a34a', label: 'Eaten' },
              { key: 'tradedFruit', color: '#2563eb', label: 'Traded' },
            ]}
            stacked
          />
          <TimeSeriesChart
            title="Vitamins"
            data={history}
            series={[
              { key: 'avgVitaminA', color: '#e11d48', label: 'Vit A' },
              { key: 'avgVitaminB', color: '#f59e0b', label: 'Vit B' },
              { key: 'avgVitaminC', color: '#2563eb', label: 'Vit C' },
            ]}
          />
          <ActionBarChart histogram={stats.actionHistogram} />
        </div>
      )}
    </div>
  )
}
