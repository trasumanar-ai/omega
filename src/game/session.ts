import { Simulation } from '../sim/simulation'
import type { TickStats } from '../sim/types'
import {
  DEFAULT_GAME_CONFIG,
  createInitialStats,
  normalizeGameConfig,
  toSimulationConfig,
  type GameConfig,
} from './config'

export class GameSession {
  private config: GameConfig
  private sim: Simulation | null = null
  private stats: TickStats
  private dirty = true

  constructor(initialConfig: GameConfig = DEFAULT_GAME_CONFIG) {
    this.config = normalizeGameConfig(initialConfig)
    this.stats = createInitialStats(this.config)
  }

  getConfig(): GameConfig {
    return this.config
  }

  getStats(): TickStats {
    return this.stats
  }

  getSimulation(): Simulation {
    if (!this.sim || this.dirty) {
      this.rebuildSimulation()
    }
    return this.sim as Simulation
  }

  setConfig(update: Partial<GameConfig> | ((prev: GameConfig) => GameConfig)): GameConfig {
    const next = typeof update === 'function'
      ? update(this.config)
      : { ...this.config, ...update }
    this.config = normalizeGameConfig(next)
    this.dirty = true
    return this.config
  }

  step(): TickStats {
    const simulation = this.getSimulation()
    this.stats = simulation.step()
    return this.stats
  }

  reset(): TickStats {
    this.sim = null
    this.dirty = true
    this.stats = createInitialStats(this.config)
    return this.stats
  }

  private rebuildSimulation(): void {
    this.sim = new Simulation(toSimulationConfig(this.config), Date.now() & 0xffffffff)
    this.stats = createInitialStats(this.config)
    this.dirty = false
  }
}
