import { GridWorld } from './gridWorld'
import { Rng } from './rng'
import type { Coord, SimulationConfig, TickStats } from './types'

export class Simulation {
  readonly config: SimulationConfig
  readonly grid: GridWorld
  readonly agentIds: number[]

  private readonly rng: Rng
  private tickCount = 0

  constructor(config: SimulationConfig, seed = 42) {
    if (config.width <= 0 || config.height <= 0) {
      throw new Error('Grid dimensions must be positive.')
    }
    if (config.agentCount <= 0) {
      throw new Error('agentCount must be positive.')
    }
    if (config.agentCount > config.width * config.height) {
      throw new Error('agentCount cannot exceed cell count.')
    }

    this.config = config
    this.rng = new Rng(seed)
    this.grid = new GridWorld(config.width, config.height, config.agentCount)
    this.agentIds = Array.from({ length: config.agentCount }, (_, idx) => idx)

    this.seedAgents()
  }

  get tick(): number {
    return this.tickCount
  }

  step(): TickStats {
    this.tickCount += 1
    let movedAgents = 0

    this.rng.shuffle(this.agentIds)
    for (const agentId of this.agentIds) {
      const current = this.grid.getPosition(agentId)
      const neighbors = this.grid.neighbors4(current)
      this.rng.shuffle(neighbors)

      for (const next of neighbors) {
        if (this.grid.move(agentId, next)) {
          movedAgents += 1
          break
        }
      }
    }

    return {
      tick: this.tickCount,
      movedAgents,
      occupancy: this.config.agentCount / (this.config.width * this.config.height),
    }
  }

  getAgentPosition(agentId: number): Coord {
    return this.grid.getPosition(agentId)
  }

  private seedAgents(): void {
    for (const agentId of this.agentIds) {
      const cell = this.randomEmptyCell()
      this.grid.place(agentId, cell)
    }
  }

  private randomEmptyCell(): Coord {
    while (true) {
      const cell = {
        x: this.rng.int(0, this.config.width),
        y: this.rng.int(0, this.config.height),
      }
      if (this.grid.isEmpty(cell)) {
        return cell
      }
    }
  }
}
