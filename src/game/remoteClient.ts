import type { Coord, FruitType } from '../sim/types'
import type { SimulationConfig, TickStats, ActionType } from '../sim/types'

const FRUIT_CODES: readonly FruitType[] = ['none', 'apple', 'banana', 'orange']

export type WorldSnapshotResponse = {
  width: number
  height: number
  treeTypeByCell: number[]
  groundFruitTypeByCell: number[]
  agentX: number[]
  agentY: number[]
  alive: boolean[]
  treeIndices: number[]
}

export type SimulationStateResponse = {
  config: SimulationConfig
  stats: TickStats
  world: WorldSnapshotResponse
}

export type AgentDetailResponse = {
  id: number
  x: number
  y: number
  energy: number
  inventory: number
  inventoryByFruit: {
    apple: number
    banana: number
    orange: number
  }
  vitamins: {
    vitamin_a: number
    vitamin_b: number
    vitamin_c: number
  }
  alive: boolean
  bornTick: number
  deathTick: number
  age: number
  totalActions: number
  actionCounts: Record<ActionType, number>
  recentActions: string[]
  neuronCount: number
  neuronEnergyCost: number
  genome: {
    mutationRate: number
    mutationScale: number
    addNeuronChance: number
    removeNeuronChance: number
    fruitBiasApple: number
    fruitBiasBanana: number
    fruitBiasOrange: number
  }
}

export type AgentDetail = Omit<AgentDetailResponse, 'recentActions'> & {
  recentActions: ActionType[]
}

export class RemoteSimulation {
  config: { width: number; height: number }
  agentIds: number[]

  private treeTypeByCell: Uint8Array
  private groundFruitTypeByCell: Uint8Array
  private agentX: Int32Array
  private agentY: Int32Array
  private alive: Uint8Array
  private occupancy: Int32Array
  private treeIndices: number[]

  constructor(width: number, height: number, maxAgents: number) {
    this.config = { width, height }
    this.agentIds = Array.from({ length: maxAgents }, (_, idx) => idx)
    const cellCount = Math.max(1, width * height)
    this.treeTypeByCell = new Uint8Array(cellCount)
    this.groundFruitTypeByCell = new Uint8Array(cellCount)
    this.agentX = new Int32Array(maxAgents)
    this.agentY = new Int32Array(maxAgents)
    this.alive = new Uint8Array(maxAgents)
    this.occupancy = new Int32Array(cellCount)
    this.occupancy.fill(-1)
    this.treeIndices = []
  }

  applyWorld(world: WorldSnapshotResponse): void {
    this.config = { width: world.width, height: world.height }

    this.treeTypeByCell = Uint8Array.from(world.treeTypeByCell)
    this.groundFruitTypeByCell = Uint8Array.from(world.groundFruitTypeByCell)
    this.agentX = Int32Array.from(world.agentX)
    this.agentY = Int32Array.from(world.agentY)
    this.alive = Uint8Array.from(world.alive.map(v => (v ? 1 : 0)))
    this.agentIds = Array.from({ length: world.agentX.length }, (_, idx) => idx)
    this.treeIndices = [...world.treeIndices]

    const cellCount = Math.max(1, world.width * world.height)
    this.occupancy = new Int32Array(cellCount)
    this.occupancy.fill(-1)
    for (let id = 0; id < this.agentX.length; id += 1) {
      if (this.alive[id] !== 1) continue
      const x = this.agentX[id]
      const y = this.agentY[id]
      if (!this.inBounds(x, y)) continue
      this.occupancy[this.index(x, y)] = id
    }
  }

  getTreeKindAt(x: number, y: number): FruitType {
    if (!this.inBounds(x, y)) return 'none'
    return codeToFruit(this.treeTypeByCell[this.index(x, y)] ?? 0)
  }

  getGroundFruitKindAt(x: number, y: number): FruitType {
    if (!this.inBounds(x, y)) return 'none'
    return codeToFruit(this.groundFruitTypeByCell[this.index(x, y)] ?? 0)
  }

  getTreeIndices(): readonly number[] {
    return this.treeIndices
  }

  isAgentAlive(agentId: number): boolean {
    return this.alive[agentId] === 1
  }

  getAgentPosition(agentId: number): Coord {
    return {
      x: this.agentX[agentId] ?? -1,
      y: this.agentY[agentId] ?? -1,
    }
  }

  agentAt(x: number, y: number): number {
    if (!this.inBounds(x, y)) return -1
    return this.occupancy[this.index(x, y)] ?? -1
  }

  private inBounds(x: number, y: number): boolean {
    return x >= 0 && x < this.config.width && y >= 0 && y < this.config.height
  }

  private index(x: number, y: number): number {
    return y * this.config.width + x
  }
}

export function normalizeAgentDetail(raw: AgentDetailResponse): AgentDetail {
  const recentActions = raw.recentActions
    .map(toActionType)
    .filter((value): value is ActionType => value !== null)

  return {
    ...raw,
    recentActions,
  }
}

function toActionType(value: string): ActionType | null {
  switch (value) {
    case 'stay':
    case 'move_north':
    case 'move_east':
    case 'move_south':
    case 'move_west':
    case 'collect_fruit':
    case 'eat_fruit':
    case 'trade_fruit':
    case 'clone_self':
      return value
    default:
      return null
  }
}

function codeToFruit(code: number): FruitType {
  if (code < 0 || code >= FRUIT_CODES.length) {
    return 'none'
  }
  return FRUIT_CODES[code]
}
