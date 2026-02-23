import type { TickStats } from '../sim/types'
import type { GameConfig } from './config'

const RUN_HISTORY_KEY = 'omega.runHistory.v1'
const ACTIVE_RUN_KEY = 'omega.activeRun.v1'
const RUN_HISTORY_LIMIT = 300
const RUN_TRACE_LIMIT = 1200
const RUN_TRACE_INTERVAL = 5

export type RunEndReason =
  | 'manual_reset'
  | 'config_changed'
  | 'session_rebuild'
  | 'session_dispose'
  | 'interrupted_reload'

export type RunTracePoint = {
  tick: number
  aliveAgents: number
  avgEnergy: number
  avgInventory: number
  deathsThisTick: number
  collectedFruit: number
  eatenFruit: number
  tradedFruit: number
  groundFruitTotal: number
  deficientAgents: number
}

export type RunRecord = {
  id: string
  seed: number
  startedAt: string
  endedAt: string
  reason: RunEndReason
  config: GameConfig
  stepCount: number
  maxTick: number
  finalStats: TickStats
  trace: RunTracePoint[]
}

export type ActiveRunRecord = {
  id: string
  seed: number
  startedAt: string
  updatedAt: string
  config: GameConfig
  stepCount: number
  maxTick: number
  lastStats: TickStats
  trace: RunTracePoint[]
}

export function loadRunHistory(): RunRecord[] {
  const parsed = readJson<unknown>(RUN_HISTORY_KEY)
  if (!Array.isArray(parsed)) {
    return []
  }
  return parsed.filter((item): item is RunRecord => isObject(item)) as RunRecord[]
}

export function saveRunHistory(history: readonly RunRecord[]): void {
  writeJson(RUN_HISTORY_KEY, history.slice(-RUN_HISTORY_LIMIT))
}

export function loadActiveRun(): ActiveRunRecord | null {
  const parsed = readJson<unknown>(ACTIVE_RUN_KEY)
  return isObject(parsed) ? (parsed as ActiveRunRecord) : null
}

export function saveActiveRun(run: ActiveRunRecord | null): void {
  if (!canUseStorage()) {
    return
  }
  if (run === null) {
    window.localStorage.removeItem(ACTIVE_RUN_KEY)
    return
  }
  writeJson(ACTIVE_RUN_KEY, run)
}

export function appendRun(history: readonly RunRecord[], run: RunRecord): RunRecord[] {
  return [...history, run].slice(-RUN_HISTORY_LIMIT)
}

export function createActiveRun(config: GameConfig, seed: number, initialStats: TickStats): ActiveRunRecord {
  const now = new Date().toISOString()
  return {
    id: createRunId(),
    seed,
    startedAt: now,
    updatedAt: now,
    config: cloneConfig(config),
    stepCount: 0,
    maxTick: initialStats.tick,
    lastStats: cloneTickStats(initialStats),
    trace: [toTracePoint(initialStats)],
  }
}

export function recordRunStep(run: ActiveRunRecord, stats: TickStats): ActiveRunRecord {
  const nextStats = cloneTickStats(stats)
  run.stepCount += 1
  run.maxTick = Math.max(run.maxTick, nextStats.tick)
  run.lastStats = nextStats
  run.updatedAt = new Date().toISOString()

  const shouldRecordTrace =
    run.stepCount <= 120 ||
    run.stepCount % RUN_TRACE_INTERVAL === 0 ||
    nextStats.deathsThisTick > 0 ||
    nextStats.tradedFruit > 0

  if (shouldRecordTrace) {
    run.trace.push(toTracePoint(nextStats))
    if (run.trace.length > RUN_TRACE_LIMIT) {
      run.trace.shift()
    }
  }

  return run
}

export function finishRun(run: ActiveRunRecord, reason: RunEndReason): RunRecord {
  return {
    id: run.id,
    seed: run.seed,
    startedAt: run.startedAt,
    endedAt: new Date().toISOString(),
    reason,
    config: cloneConfig(run.config),
    stepCount: run.stepCount,
    maxTick: run.maxTick,
    finalStats: cloneTickStats(run.lastStats),
    trace: run.trace.map(point => ({ ...point })),
  }
}

export function exposeRunDebug(history: readonly RunRecord[], activeRun: ActiveRunRecord | null): void {
  if (typeof window === 'undefined') {
    return
  }
  const globalWindow = window as Window & {
    __omegaRuns?: readonly RunRecord[]
    __omegaActiveRun?: ActiveRunRecord | null
  }
  globalWindow.__omegaRuns = history
  globalWindow.__omegaActiveRun = activeRun
}

function createRunId(): string {
  const now = Date.now().toString(36)
  const rand = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER).toString(36).slice(0, 8)
  return `run_${now}_${rand}`
}

function cloneConfig(config: GameConfig): GameConfig {
  return { ...config }
}

function cloneTickStats(stats: TickStats): TickStats {
  return {
    ...stats,
    actionHistogram: { ...stats.actionHistogram },
  }
}

function toTracePoint(stats: TickStats): RunTracePoint {
  return {
    tick: stats.tick,
    aliveAgents: stats.aliveAgents,
    avgEnergy: stats.avgEnergy,
    avgInventory: stats.avgInventory,
    deathsThisTick: stats.deathsThisTick,
    collectedFruit: stats.collectedFruit,
    eatenFruit: stats.eatenFruit,
    tradedFruit: stats.tradedFruit,
    groundFruitTotal: stats.groundFruitTotal,
    deficientAgents: stats.deficientAgents,
  }
}

function canUseStorage(): boolean {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined'
}

function readJson<T>(key: string): T | null {
  if (!canUseStorage()) {
    return null
  }
  const raw = window.localStorage.getItem(key)
  if (!raw) {
    return null
  }
  try {
    return JSON.parse(raw) as T
  } catch {
    return null
  }
}

function writeJson(key: string, value: unknown): void {
  if (!canUseStorage()) {
    return
  }
  try {
    window.localStorage.setItem(key, JSON.stringify(value))
  } catch {
    // Ignore storage quota/availability errors for best-effort logging.
  }
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
