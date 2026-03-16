export type Coord = {
  x: number
  y: number
}

export const FRUIT_TYPES = ['none', 'apple', 'banana', 'orange'] as const
export type FruitType = (typeof FRUIT_TYPES)[number]

export const VITAMIN_KEYS = ['vitamin_a', 'vitamin_b', 'vitamin_c'] as const
export type VitaminKey = (typeof VITAMIN_KEYS)[number]

export type FruitInventory = {
  apple: number
  banana: number
  orange: number
}

export type VitaminLevels = {
  vitamin_a: number
  vitamin_b: number
  vitamin_c: number
}

export const ACTION_SPACE = [
  'stay',
  'move_north',
  'move_east',
  'move_south',
  'move_west',
  'collect_fruit',
  'eat_fruit',
  'trade_fruit',
  'clone_self',
] as const

export type ActionType = (typeof ACTION_SPACE)[number]

export type AgentAction = {
  type: ActionType
}

export type AgentSnapshot = {
  id: number
  x: number
  y: number
  energy: number
  inventory: number
  inventoryByFruit: FruitInventory
  vitamins: VitaminLevels
  alive: boolean
}

export type AgentObservation = {
  tick: number
  self: AgentSnapshot
  neighbors: {
    north: number
    east: number
    south: number
    west: number
  }
  resources: {
    hasTree: number
    groundFruit: number
    treeType: FruitType
    groundFruitType: FruitType
    neighborFruitNorth: number
    neighborFruitEast: number
    neighborFruitSouth: number
    neighborFruitWest: number
  }
  social: {
    adjacentAgents: number
  }
}

export type ActionProvider = (
  agentId: number,
  observation: AgentObservation,
  validActions: readonly AgentAction[],
) => AgentAction

export type ActionHistogram = Record<ActionType, number>

export type SimulationConfig = {
  width: number
  height: number
  agentCount: number
  maxAgentSlots?: number
  initialEnergy?: number
  maxEnergy?: number
  moveEnergyCost?: number
  idleEnergyCost?: number
  neuronStepEnergyCost?: number
  minNeuronCount?: number
  maxNeuronCount?: number
  mutationRateBase?: number
  mutationScaleBase?: number
  addNeuronChanceBase?: number
  removeNeuronChanceBase?: number
  cloneEnergyThreshold?: number
  cloneEnergyCost?: number
  cloneChancePerTick?: number
  treeDensity?: number
  fruitDropPerTree?: number
  maxFruitAroundTree?: number
  maxInventory?: number
  fruitEnergyGain?: number
  tradeAmount?: number
  initialVitamin?: number
  maxVitamin?: number
  vitaminDecayPerTick?: number
  vitaminGainPerFruit?: number
  vitaminNeedThreshold?: number
  vitaminDeficiencyEnergyPenalty?: number
}

export type TickStats = {
  tick: number
  movedAgents: number
  occupancy: number
  aliveAgents: number
  deathsThisTick: number
  avgEnergy: number
  avgInventory: number
  collectedFruit: number
  eatenFruit: number
  tradedFruit: number
  groundFruitTotal: number
  groundApple: number
  groundBanana: number
  groundOrange: number
  avgVitaminA: number
  avgVitaminB: number
  avgVitaminC: number
  deficientAgents: number
  actionHistogram: ActionHistogram
}

export function emptyActionHistogram(): ActionHistogram {
  return {
    stay: 0,
    move_north: 0,
    move_east: 0,
    move_south: 0,
    move_west: 0,
    collect_fruit: 0,
    eat_fruit: 0,
    trade_fruit: 0,
    clone_self: 0,
  }
}

export function actionToIndex(action: AgentAction): number {
  return ACTION_SPACE.indexOf(action.type)
}

export function indexToAction(index: number): AgentAction {
  if (index < 0 || index >= ACTION_SPACE.length) {
    return { type: 'stay' }
  }
  return { type: ACTION_SPACE[index] }
}

export function encodeObservation(observation: AgentObservation): number[] {
  return [
    observation.tick,
    observation.self.x,
    observation.self.y,
    observation.self.energy,
    observation.self.inventory,
    observation.self.inventoryByFruit.apple,
    observation.self.inventoryByFruit.banana,
    observation.self.inventoryByFruit.orange,
    observation.self.vitamins.vitamin_a,
    observation.self.vitamins.vitamin_b,
    observation.self.vitamins.vitamin_c,
    observation.self.alive ? 1 : 0,
    observation.neighbors.north,
    observation.neighbors.east,
    observation.neighbors.south,
    observation.neighbors.west,
    observation.resources.hasTree,
    observation.resources.groundFruit,
    fruitTypeToCode(observation.resources.treeType),
    fruitTypeToCode(observation.resources.groundFruitType),
    observation.resources.neighborFruitNorth,
    observation.resources.neighborFruitEast,
    observation.resources.neighborFruitSouth,
    observation.resources.neighborFruitWest,
    observation.social.adjacentAgents,
  ]
}

export function fruitTypeToCode(type: FruitType): number {
  return FRUIT_TYPES.indexOf(type)
}
