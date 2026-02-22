export type Coord = {
  x: number
  y: number
}

export type SimulationConfig = {
  width: number
  height: number
  agentCount: number
}

export type TickStats = {
  tick: number
  movedAgents: number
  occupancy: number
}
