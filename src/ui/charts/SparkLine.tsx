import { LineChart, Line, ResponsiveContainer, Tooltip } from 'recharts'

type Series = {
  key: string
  color: string
  label?: string
}

type Props = {
  data: Record<string, unknown>[]
  series: Series[]
  height?: number
}

export function SparkLine({ data, series, height = 40 }: Props) {
  if (data.length < 2) {
    return <div className="spark-empty" style={{ height }} />
  }

  return (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={data} margin={{ top: 2, right: 2, bottom: 2, left: 2 }}>
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
        {series.map(s => (
          <Line
            key={s.key}
            type="monotone"
            dataKey={s.key}
            stroke={s.color}
            strokeWidth={1.5}
            dot={false}
            isAnimationActive={false}
            name={s.label ?? s.key}
          />
        ))}
      </LineChart>
    </ResponsiveContainer>
  )
}
