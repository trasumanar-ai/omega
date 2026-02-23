import {
  BarChart, Bar, XAxis, YAxis, Tooltip,
  ResponsiveContainer, Cell,
} from 'recharts'
import type { ActionHistogram } from '../../sim/types'

const ACTION_COLORS: Record<string, string> = {
  stay: '#a3a3a3',
  move: '#737373',
  collect: '#d97706',
  eat: '#16a34a',
  trade: '#2563eb',
  clone: '#8b5cf6',
}

type Props = {
  histogram: ActionHistogram
  height?: number
}

export function ActionBarChart({ histogram, height = 160 }: Props) {
  const data = [
    { name: 'Stay', value: histogram.stay, color: ACTION_COLORS.stay },
    { name: 'Move', value: histogram.move_north + histogram.move_east + histogram.move_south + histogram.move_west, color: ACTION_COLORS.move },
    { name: 'Collect', value: histogram.collect_fruit, color: ACTION_COLORS.collect },
    { name: 'Eat', value: histogram.eat_fruit, color: ACTION_COLORS.eat },
    { name: 'Trade', value: histogram.trade_fruit, color: ACTION_COLORS.trade },
    { name: 'Clone', value: histogram.clone_self, color: ACTION_COLORS.clone },
  ]

  return (
    <div className="ts-chart">
      <span className="ts-chart-title">Actions (this tick)</span>
      <ResponsiveContainer width="100%" height={height}>
        <BarChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
          <XAxis
            dataKey="name"
            tick={{ fontSize: 10 }}
            tickLine={false}
            axisLine={{ stroke: '#e5e5e5' }}
          />
          <YAxis
            tick={{ fontSize: 10 }}
            tickLine={false}
            axisLine={false}
            width={36}
          />
          <Tooltip
            contentStyle={{
              background: '#fff',
              border: '1px solid #e5e5e5',
              borderRadius: 4,
              fontSize: 11,
              padding: '4px 8px',
            }}
          />
          <Bar dataKey="value" radius={[2, 2, 0, 0]} isAnimationActive={false}>
            {data.map((entry, i) => (
              <Cell key={i} fill={entry.color} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
