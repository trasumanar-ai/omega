export type Coord = {
  x: number
  y: number
}

export const ACTION_SPACE = [
  'stay',
  'move_north',
  'move_east',
  'move_south',
  'move_west',
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
  initialEnergy?: number
  moveEnergyCost?: number
  idleEnergyCost?: number
}

export type TickStats = {
  tick: number
  movedAgents: number
  occupancy: number
  aliveAgents: number
  deathsThisTick: number
  avgEnergy: number
  actionHistogram: ActionHistogram
}

export function emptyActionHistogram(): ActionHistogram {
  return {
    stay: 0,
    move_north: 0,
    move_east: 0,
    move_south: 0,
    move_west: 0,
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
    observation.self.x,
    observation.self.y,
    observation.self.energy,
    observation.self.inventory,
    observation.self.alive ? 1 : 0,
    observation.neighbors.north,
    observation.neighbors.east,
    observation.neighbors.south,
    observation.neighbors.west,
  ]
}
