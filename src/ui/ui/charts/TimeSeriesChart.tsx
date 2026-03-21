import {
  AreaChart, Area, LineChart, Line,
  XAxis, YAxis, Tooltip, Legend,
  ResponsiveContainer,
} from 'recharts'

type Series = {
  key: string
  color: string
  label: string
  type?: 'line' | 'area'
}

type Props = {
  title: string
  data: Record<string, unknown>[]
  series: Series[]
  height?: number
  stacked?: boolean
}

export function TimeSeriesChart({ title, data, series, height = 160, stacked = false }: Props) {
  if (data.length < 2) {
    return (
      <div className="ts-chart">
        <span className="ts-chart-title">{title}</span>
        <div className="ts-chart-empty" style={{ height }}>waiting for data...</div>
      </div>
    )
  }

  const useArea = stacked || series.some(s => s.type === 'area')

  return (
    <div className="ts-chart">
      <span className="ts-chart-title">{title}</span>
      <ResponsiveContainer width="100%" height={height}>
        {useArea ? (
          <AreaChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <XAxis
              dataKey="tick"
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
              labelFormatter={(v) => `tick ${v}`}
            />
            <Legend
              iconSize={8}
              wrapperStyle={{ fontSize: 10, paddingTop: 2 }}
            />
            {series.map(s => (
              <Area
                key={s.key}
                type="monotone"
                dataKey={s.key}
                stroke={s.color}
                fill={s.color}
                fillOpacity={0.15}
                strokeWidth={1.5}
                stackId={stacked ? 'stack' : undefined}
                isAnimationActive={false}
                name={s.label}
              />
            ))}
          </AreaChart>
        ) : (
          <LineChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <XAxis
              dataKey="tick"
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
              labelFormatter={(v) => `tick ${v}`}
            />
            <Legend
              iconSize={8}
              wrapperStyle={{ fontSize: 10, paddingTop: 2 }}
            />
            {series.map(s => (
              <Line
                key={s.key}
                type="monotone"
                dataKey={s.key}
                stroke={s.color}
                strokeWidth={1.5}
                dot={false}
                isAnimationActive={false}
                name={s.label}
              />
            ))}
          </LineChart>
        )}
      </ResponsiveContainer>
    </div>
  )
}
