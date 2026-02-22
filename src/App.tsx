import { useCallback, useEffect, useRef, useState } from 'react'
import { Sidebar } from './ui/Sidebar'
import { useSimulation } from './hooks/useSimulation'

const MIN_CELL = 2
const MAX_CELL = 64
const MIN_SIDEBAR = 180
const MAX_SIDEBAR = 480

export function App() {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const miniRef = useRef<HTMLCanvasElement>(null)
  const vpRef = useRef<HTMLDivElement>(null)

  const [cellSize, setCellSize] = useState(10)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [sidebarW, setSidebarW] = useState(232)
  const dragging = useRef(false)
  const dragStart = useRef({ x: 0, y: 0, px: 0, py: 0 })
  const resizingSidebar = useRef(false)

  const {
    sim, stats, running,
    toggle, step, reset,
    config, setConfig,
  } = useSimulation()

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

    // bg
    ctx.fillStyle = '#fafafa'
    ctx.fillRect(0, 0, vpW, vpH)

    // grid lines (only visible range)
    if (cell >= 6) {
      ctx.strokeStyle = 'rgba(0,0,0,0.06)'
      ctx.lineWidth = 1
      const x0 = Math.max(0, Math.floor(-ox / cell))
      const x1 = Math.min(gw, Math.ceil((vpW - ox) / cell))
      const y0 = Math.max(0, Math.floor(-oy / cell))
      const y1 = Math.min(gh, Math.ceil((vpH - oy) / cell))
      for (let x = x0; x <= x1; x++) {
        const px = ox + x * cell + 0.5
        ctx.beginPath(); ctx.moveTo(px, 0); ctx.lineTo(px, vpH); ctx.stroke()
      }
      for (let y = y0; y <= y1; y++) {
        const py = oy + y * cell + 0.5
        ctx.beginPath(); ctx.moveTo(0, py); ctx.lineTo(vpW, py); ctx.stroke()
      }
    }

    // agents (cull off-screen)
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

    // minimap
    const mw = mini.width
    const mh = mini.height
    mctx.fillStyle = '#fafafa'
    mctx.fillRect(0, 0, mw, mh)
    const sx = mw / gw
    const sy = mh / gh
    mctx.fillStyle = 'rgba(23,23,23,0.5)'
    for (const id of sim.agentIds) {
      if (!sim.isAgentAlive(id)) continue
      const p = sim.getAgentPosition(id)
      mctx.fillRect(
        Math.floor(p.x * sx), Math.floor(p.y * sy),
        Math.max(1, Math.ceil(sx)), Math.max(1, Math.ceil(sy)),
      )
    }

    // viewport rect on minimap
    const vx = (-pan.x - (vpW - totalW) / 2) / totalW
    const vy = (-pan.y - (vpH - totalH) / 2) / totalH
    const vw = vpW / totalW
    const vh = vpH / totalH
    mctx.strokeStyle = 'rgba(23,23,23,0.6)'
    mctx.lineWidth = 1.5
    mctx.strokeRect(
      Math.floor(vx * mw), Math.floor(vy * mh),
      Math.ceil(vw * mw), Math.ceil(vh * mh),
    )
  }, [sim, cellSize, pan])

  useEffect(() => { render() }, [stats, render])
  useEffect(() => {
    const h = () => render()
    window.addEventListener('resize', h)
    return () => window.removeEventListener('resize', h)
  }, [render])

  // wheel zoom (anchored to cursor)
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

  // drag pan
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
      if (resizingSidebar.current) {
        setSidebarW(Math.max(MIN_SIDEBAR, Math.min(MAX_SIDEBAR, e.clientX)))
      }
    }
    const onUp = () => {
      dragging.current = false
      if (resizingSidebar.current) {
        resizingSidebar.current = false
        document.body.style.cursor = ''
      }
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    return () => { window.removeEventListener('mousemove', onMove); window.removeEventListener('mouseup', onUp) }
  }, [])

  const onResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    resizingSidebar.current = true
    document.body.style.cursor = 'col-resize'
  }, [])

  return (
    <div className="shell" style={{ gridTemplateColumns: `${sidebarW}px 4px 1fr` }}>
      <Sidebar
        config={config}
        onConfig={setConfig}
        stats={stats}
        running={running}
        onToggle={toggle}
        onStep={step}
        onReset={reset}
        miniRef={miniRef}
      />
      <div className="resize-handle" onMouseDown={onResizeStart} />
      <main
        className="viewport"
        ref={vpRef}
        onWheel={onWheel}
        onMouseDown={onMouseDown}
        style={{ cursor: dragging.current ? 'grabbing' : 'grab' }}
      >
        <canvas ref={canvasRef} />
        <div className="vp-hud">
          tick <strong>{stats.tick}</strong> &middot; alive <strong>{stats.aliveAgents}</strong>
        </div>
      </main>
    </div>
  )
}
