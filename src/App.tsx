import { useCallback, useEffect, useRef, useState } from 'react'
import { NavBar } from './ui/NavBar'
import { ContentPanel } from './ui/ContentPanel'
import { AgentPanel } from './ui/AgentPanel'
import { useSimulation } from './hooks/useSimulation'
import type { AgentDetail } from './hooks/useSimulation'
import type { FruitType } from './sim/types'

const MIN_CELL = 2
const MAX_CELL = 64
const CLICK_THRESHOLD = 4

const TREE_CELL_COLORS: Record<FruitType, { bed: string; canopy: string }> = {
  none: { bed: 'transparent', canopy: 'transparent' },
  apple: { bed: '#e6f4df', canopy: '#2f7d32' },
  banana: { bed: '#fff4cc', canopy: '#a78b1b' },
  orange: { bed: '#ffe9d6', canopy: '#c25a15' },
}

const FRUIT_BLOCK_COLORS: Record<FruitType, string> = {
  none: 'transparent',
  apple: '#e11d48',
  banana: '#facc15',
  orange: '#f97316',
}

export function App() {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const miniRef = useRef<HTMLCanvasElement>(null)
  const vpRef = useRef<HTMLDivElement>(null)

  const [cellSize, setCellSize] = useState(10)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [activeSection, setActiveSection] = useState<string | null>('overview')
  const [selectedAgent, setSelectedAgent] = useState<number | null>(null)
  const [selectedAgentDetail, setSelectedAgentDetail] = useState<AgentDetail | null>(null)
  const dragging = useRef(false)
  const dragStart = useRef({ x: 0, y: 0, px: 0, py: 0 })

  const {
    sim, stats, statsHistory, running,
    toggle, step, reset,
    config, setConfig,
    getAgentDetail,
  } = useSimulation()

  const handleReset = useCallback(() => {
    setSelectedAgent(null)
    setSelectedAgentDetail(null)
    reset()
  }, [reset])

  useEffect(() => {
    if (selectedAgent === null) {
      setSelectedAgentDetail(null)
      return
    }

    let cancelled = false
    const load = async () => {
      const detail = await getAgentDetail(selectedAgent)
      if (!cancelled) {
        setSelectedAgentDetail(detail)
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [selectedAgent, stats.tick, getAgentDetail])

  const render = useCallback(() => {
    const canvas = canvasRef.current
    const mini = miniRef.current
    const vp = vpRef.current
    if (!canvas || !mini || !vp || !sim) return

    const ctx = canvas.getContext('2d')!
    const mctx = mini.getContext('2d')!
    const { width: gw, height: gh } = sim.config
    const cell = cellSize

    const vpW = vp.clientWidth
    const vpH = vp.clientHeight
    const dpr = devicePixelRatio || 1

    canvas.style.width = `${vpW}px`
    canvas.style.height = `${vpH}px`
    canvas.width = Math.floor(vpW * dpr)
    canvas.height = Math.floor(vpH * dpr)
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)

    const totalW = gw * cell
    const totalH = gh * cell
    const ox = Math.floor((vpW - totalW) / 2 + pan.x)
    const oy = Math.floor((vpH - totalH) / 2 + pan.y)
    const x0 = Math.max(0, Math.floor(-ox / cell) - 1)
    const x1 = Math.min(gw - 1, Math.ceil((vpW - ox) / cell) + 1)
    const y0 = Math.max(0, Math.floor(-oy / cell) - 1)
    const y1 = Math.min(gh - 1, Math.ceil((vpH - oy) / cell) + 1)

    ctx.fillStyle = '#fafafa'
    ctx.fillRect(0, 0, vpW, vpH)

    if (cell >= 3) {
      const trunk = Math.max(1, Math.floor(cell * 0.15))
      for (let y = y0; y <= y1; y += 1) {
        for (let x = x0; x <= x1; x += 1) {
          const treeKind = sim.getTreeKindAt(x, y)
          if (treeKind === 'none') continue
          const treeColor = TREE_CELL_COLORS[treeKind]
          const px = ox + x * cell
          const py = oy + y * cell
          ctx.fillStyle = treeColor.bed
          ctx.fillRect(px, py, cell, cell)
          ctx.fillStyle = treeColor.canopy
          ctx.fillRect(px + trunk, py + trunk, cell - trunk * 2, cell - trunk * 2)
        }
      }
    }

    for (let y = y0; y <= y1; y += 1) {
      for (let x = x0; x <= x1; x += 1) {
        const fruitType = sim.getGroundFruitKindAt(x, y)
        if (fruitType === 'none') continue
        const px = ox + x * cell
        const py = oy + y * cell
        const fruitSize = Math.max(2, Math.floor(cell * 0.35))
        const fx = px + Math.floor((cell - fruitSize) / 2)
        const fy = py + Math.floor((cell - fruitSize) / 2)
        ctx.fillStyle = FRUIT_BLOCK_COLORS[fruitType]
        ctx.fillRect(fx, fy, fruitSize, fruitSize)
      }
    }

    if (cell >= 6) {
      ctx.strokeStyle = 'rgba(0,0,0,0.06)'
      ctx.lineWidth = 1
      for (let x = x0; x <= x1 + 1; x += 1) {
        const px = ox + x * cell + 0.5
        ctx.beginPath(); ctx.moveTo(px, 0); ctx.lineTo(px, vpH); ctx.stroke()
      }
      for (let y = y0; y <= y1 + 1; y += 1) {
        const py = oy + y * cell + 0.5
        ctx.beginPath(); ctx.moveTo(0, py); ctx.lineTo(vpW, py); ctx.stroke()
      }
    }

    const pad = Math.max(1, Math.floor(cell * 0.1))
    const sz = cell - pad * 2
    ctx.fillStyle = '#171717'
    for (const id of sim.agentIds) {
      if (!sim.isAgentAlive(id)) continue
      const p = sim.getAgentPosition(id)
      const px = ox + p.x * cell + pad
      const py = oy + p.y * cell + pad
      if (px + sz < 0 || px > vpW || py + sz < 0 || py > vpH) continue
      ctx.fillRect(px, py, sz, sz)
    }

    if (selectedAgent !== null) {
      const p = sim.getAgentPosition(selectedAgent)
      if (p.x >= 0 && p.y >= 0) {
        const px = ox + p.x * cell
        const py = oy + p.y * cell
        ctx.strokeStyle = '#e11d48'
        ctx.lineWidth = 2
        ctx.strokeRect(px + 0.5, py + 0.5, cell - 1, cell - 1)
      }
    }

    // minimap
    const mw = mini.width
    const mh = mini.height
    mctx.fillStyle = '#fafafa'
    mctx.fillRect(0, 0, mw, mh)
    const sx = mw / gw
    const sy = mh / gh

    mctx.fillStyle = 'rgba(47, 125, 50, 0.2)'
    for (const treeIndex of sim.getTreeIndices()) {
      const tx = treeIndex % gw
      const ty = Math.floor(treeIndex / gw)
      mctx.fillRect(Math.floor(tx * sx), Math.floor(ty * sy), Math.max(1, Math.ceil(sx)), Math.max(1, Math.ceil(sy)))
    }

    for (let gy = 0; gy < gh; gy += 1) {
      for (let gx = 0; gx < gw; gx += 1) {
        const fruitType = sim.getGroundFruitKindAt(gx, gy)
        if (fruitType === 'none') continue
        mctx.fillStyle = FRUIT_BLOCK_COLORS[fruitType]
        mctx.fillRect(Math.floor(gx * sx), Math.floor(gy * sy), Math.max(1, Math.ceil(sx)), Math.max(1, Math.ceil(sy)))
      }
    }

    mctx.fillStyle = 'rgba(23,23,23,0.5)'
    for (const id of sim.agentIds) {
      if (!sim.isAgentAlive(id)) continue
      const p = sim.getAgentPosition(id)
      mctx.fillRect(Math.floor(p.x * sx), Math.floor(p.y * sy), Math.max(1, Math.ceil(sx)), Math.max(1, Math.ceil(sy)))
    }

    const vx = (-pan.x - (vpW - totalW) / 2) / totalW
    const vy = (-pan.y - (vpH - totalH) / 2) / totalH
    const vw = vpW / totalW
    const vh = vpH / totalH
    mctx.strokeStyle = 'rgba(23,23,23,0.6)'
    mctx.lineWidth = 1.5
    mctx.strokeRect(Math.floor(vx * mw), Math.floor(vy * mh), Math.ceil(vw * mw), Math.ceil(vh * mh))
  }, [sim, cellSize, pan, selectedAgent])

  useEffect(() => { render() }, [stats, render])
  useEffect(() => {
    const h = () => render()
    window.addEventListener('resize', h)
    return () => window.removeEventListener('resize', h)
  }, [render])

  const onWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault()
    const vp = vpRef.current
    if (!vp || !sim) return

    const rect = vp.getBoundingClientRect()
    const mx = e.clientX - rect.left
    const my = e.clientY - rect.top

    const { width: gw, height: gh } = sim.config
    const totalW = gw * cellSize
    const totalH = gh * cellSize
    const curOx = (vp.clientWidth - totalW) / 2 + pan.x
    const curOy = (vp.clientHeight - totalH) / 2 + pan.y

    const wx = (mx - curOx) / cellSize
    const wy = (my - curOy) / cellSize

    const factor = e.deltaY < 0 ? 1.15 : 1 / 1.15
    const next = Math.max(MIN_CELL, Math.min(MAX_CELL, cellSize * factor))

    const newTotalW = gw * next
    const newTotalH = gh * next
    const newPanX = mx - wx * next - (vp.clientWidth - newTotalW) / 2
    const newPanY = my - wy * next - (vp.clientHeight - newTotalH) / 2

    setCellSize(next)
    setPan({ x: newPanX, y: newPanY })
  }, [cellSize, pan, sim])

  const onMouseDown = useCallback((e: React.MouseEvent) => {
    if (e.button !== 0) return
    dragging.current = true
    dragStart.current = { x: e.clientX, y: e.clientY, px: pan.x, py: pan.y }
  }, [pan])

  useEffect(() => {
    const onMove = (e: MouseEvent) => {
      if (dragging.current) {
        setPan({
          x: dragStart.current.px + (e.clientX - dragStart.current.x),
          y: dragStart.current.py + (e.clientY - dragStart.current.y),
        })
      }
    }
    const onUp = (e: MouseEvent) => {
      if (dragging.current) {
        const dx = Math.abs(e.clientX - dragStart.current.x)
        const dy = Math.abs(e.clientY - dragStart.current.y)
        if (dx < CLICK_THRESHOLD && dy < CLICK_THRESHOLD) {
          handleClick(e)
        }
        dragging.current = false
      }
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    return () => { window.removeEventListener('mousemove', onMove); window.removeEventListener('mouseup', onUp) }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sim, cellSize, pan])

  const handleClick = useCallback((e: MouseEvent) => {
    const vp = vpRef.current
    if (!vp || !sim) return

    const rect = vp.getBoundingClientRect()
    const mx = e.clientX - rect.left
    const my = e.clientY - rect.top

    const { width: gw, height: gh } = sim.config
    const totalW = gw * cellSize
    const totalH = gh * cellSize
    const ox = (vp.clientWidth - totalW) / 2 + pan.x
    const oy = (vp.clientHeight - totalH) / 2 + pan.y

    const gx = Math.floor((mx - ox) / cellSize)
    const gy = Math.floor((my - oy) / cellSize)

    const agentId = sim.agentAt(gx, gy)
    setSelectedAgent(agentId >= 0 ? agentId : null)
  }, [sim, cellSize, pan])

  return (
    <div className="shell">
      <NavBar active={activeSection} onSelect={setActiveSection} />
      {activeSection && (
        <ContentPanel
          section={activeSection}
          config={config}
          onConfig={setConfig}
          stats={stats}
          statsHistory={statsHistory}
          running={running}
          onToggle={toggle}
          onStep={step}
          onReset={handleReset}
        />
      )}
      <main
        className="viewport"
        ref={vpRef}
        onWheel={onWheel}
        onMouseDown={onMouseDown}
        style={{ cursor: 'grab' }}
      >
        <canvas ref={canvasRef} />
        <div className="vp-hud">
          tick <strong>{stats.tick}</strong> &middot; alive <strong>{stats.aliveAgents}</strong>
        </div>
        <div className="minimap-float">
          <canvas ref={miniRef} width={180} height={110} />
        </div>
        {selectedAgent !== null && (
          <AgentPanel
            agentId={selectedAgent}
            detail={selectedAgentDetail}
            onClose={() => setSelectedAgent(null)}
          />
        )}
      </main>
    </div>
  )
}
