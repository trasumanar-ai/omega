import { GridWorld } from './gridWorld'
import { RuleBasedPolicy, type AgentPolicy } from './policy'
import { Rng } from './rng'
import {
  ACTION_SPACE,
  emptyActionHistogram,
  type ActionProvider,
  type AgentAction,
  type AgentObservation,
  type AgentSnapshot,
  type Coord,
  type SimulationConfig,
  type TickStats,
} from './types'

export class Simulation {
  readonly config: SimulationConfig
  readonly grid: GridWorld
  readonly agentIds: number[]

  private readonly rng: Rng
  private readonly policy: AgentPolicy
  private readonly energy: Float32Array
  private readonly inventory: Float32Array
  private readonly alive: Uint8Array
  private readonly initialEnergy: number
  private readonly moveEnergyCost: number
  private readonly idleEnergyCost: number
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
    this.initialEnergy = positive(config.initialEnergy, 40)
    this.moveEnergyCost = positive(config.moveEnergyCost, 1)
    this.idleEnergyCost = positive(config.idleEnergyCost, 0.25)
    this.grid = new GridWorld(config.width, config.height, config.agentCount)
    this.agentIds = Array.from({ length: config.agentCount }, (_, idx) => idx)
    this.energy = new Float32Array(config.agentCount)
    this.inventory = new Float32Array(config.agentCount)
    this.alive = new Uint8Array(config.agentCount)
    this.policy = new RuleBasedPolicy(() => this.rng.next())

    this.seedAgentState()
    this.seedAgents()
  }

  get tick(): number {
    return this.tickCount
  }

  step(actionProvider?: ActionProvider): TickStats {
    this.tickCount += 1
    let movedAgents = 0
    let deathsThisTick = 0
    let aliveAgents = 0
    let totalEnergy = 0
    const actionHistogram = emptyActionHistogram()

    this.rng.shuffle(this.agentIds)
    for (const agentId of this.agentIds) {
      if (!this.isAgentAlive(agentId)) {
        continue
      }
      const observation = this.getObservation(agentId)
      const validActions = this.getValidActions(agentId)
      const chosenAction = actionProvider
        ? this.pickValidAction(actionProvider(agentId, observation, validActions), validActions)
        : this.pickValidAction(this.policy.selectAction({ observation, validActions }), validActions)
      actionHistogram[chosenAction.type] += 1

      const moved = this.executeAction(agentId, chosenAction)
      movedAgents += moved ? 1 : 0

      const cost = moved ? this.moveEnergyCost : this.idleEnergyCost
      this.energy[agentId] = Math.max(0, this.energy[agentId] - cost)
      if (this.energy[agentId] <= 0) {
        this.alive[agentId] = 0
        this.grid.remove(agentId)
        deathsThisTick += 1
      } else {
        aliveAgents += 1
        totalEnergy += this.energy[agentId]
      }
    }

    return {
      tick: this.tickCount,
      movedAgents,
      occupancy: this.config.agentCount / (this.config.width * this.config.height),
      aliveAgents,
      deathsThisTick,
      avgEnergy: aliveAgents > 0 ? totalEnergy / aliveAgents : 0,
      actionHistogram,
    }
  }

  getAgentPosition(agentId: number): Coord {
    return this.grid.getPosition(agentId)
  }

  getAgentState(agentId: number): AgentSnapshot {
    const pos = this.grid.getPosition(agentId)
    return {
      id: agentId,
      x: pos.x,
      y: pos.y,
      energy: this.energy[agentId],
      inventory: this.inventory[agentId],
      alive: this.isAgentAlive(agentId),
    }
  }

  isAgentAlive(agentId: number): boolean {
    return this.alive[agentId] === 1
  }

  getObservation(agentId: number): AgentObservation {
    const self = this.getAgentState(agentId)
    if (!self.alive) {
      return {
        tick: this.tickCount,
        self,
        neighbors: {
          north: -1,
          east: -1,
          south: -1,
          west: -1,
        },
      }
    }
    const position = this.grid.getPosition(agentId)
    return {
      tick: this.tickCount,
      self,
      neighbors: {
        north: this.sense(position.x, position.y - 1),
        east: this.sense(position.x + 1, position.y),
        south: this.sense(position.x, position.y + 1),
        west: this.sense(position.x - 1, position.y),
      },
    }
  }

  getValidActions(agentId: number): readonly AgentAction[] {
    if (!this.isAgentAlive(agentId)) {
      return [{ type: 'stay' }]
    }
    const pos = this.grid.getPosition(agentId)
    const valid: AgentAction[] = [{ type: 'stay' }]
    if (this.sense(pos.x, pos.y - 1) === 0) {
      valid.push({ type: 'move_north' })
    }
    if (this.sense(pos.x + 1, pos.y) === 0) {
      valid.push({ type: 'move_east' })
    }
    if (this.sense(pos.x, pos.y + 1) === 0) {
      valid.push({ type: 'move_south' })
    }
    if (this.sense(pos.x - 1, pos.y) === 0) {
      valid.push({ type: 'move_west' })
    }
    return valid
  }

  getActionSpace(): readonly string[] {
    return ACTION_SPACE
  }

  private pickValidAction(chosen: AgentAction, validActions: readonly AgentAction[]): AgentAction {
    const ok = validActions.some(item => item.type === chosen.type)
    return ok ? chosen : { type: 'stay' }
  }

  private executeAction(agentId: number, action: AgentAction): boolean {
    if (!this.isAgentAlive(agentId)) {
      return false
    }
    const current = this.grid.getPosition(agentId)
    switch (action.type) {
      case 'stay':
        return false
      case 'move_north':
        return this.grid.move(agentId, { x: current.x, y: current.y - 1 })
      case 'move_east':
        return this.grid.move(agentId, { x: current.x + 1, y: current.y })
      case 'move_south':
        return this.grid.move(agentId, { x: current.x, y: current.y + 1 })
      case 'move_west':
        return this.grid.move(agentId, { x: current.x - 1, y: current.y })
    }
  }

  private sense(x: number, y: number): number {
    if (x < 0 || x >= this.config.width || y < 0 || y >= this.config.height) {
      return -1
    }
    return this.grid.occupancy[y * this.config.width + x] === -1 ? 0 : 1
  }

  private seedAgentState(): void {
    for (const agentId of this.agentIds) {
      this.energy[agentId] = this.initialEnergy
      this.inventory[agentId] = 0
      this.alive[agentId] = 1
    }
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

function positive(value: number | undefined, fallback: number): number {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    return fallback
  }
  return value
}
