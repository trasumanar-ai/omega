import { emptyActionHistogram, type TickStats } from '../sim/types'
import type { SimulationConfig } from '../sim/types'

export type GameConfig = {
  width: number
  height: number
  agents: number
  maxAgentSlots: number
  speed: number
  initialEnergy: number
  maxEnergy: number
  moveEnergyCost: number
  idleEnergyCost: number
  neuronStepEnergyCost: number
  minNeuronCount: number
  maxNeuronCount: number
  cloneEnergyThreshold: number
  cloneEnergyCost: number
  cloneChancePerTick: number
  treeDensity: number
  fruitDropPerTree: number
  maxFruitAroundTree: number
  maxInventory: number
  fruitEnergyGain: number
  tradeAmount: number
}

export const DEFAULT_GAME_CONFIG: GameConfig = {
  width: 80,
  height: 50,
  agents: 240,
  maxAgentSlots: 420,
  speed: 60,
  initialEnergy: 80,
  maxEnergy: 220,
  moveEnergyCost: 0.75,
  idleEnergyCost: 0.15,
  neuronStepEnergyCost: 0.03,
  minNeuronCount: 4,
  maxNeuronCount: 14,
  cloneEnergyThreshold: 145,
  cloneEnergyCost: 56,
  cloneChancePerTick: 0.6,
  treeDensity: 0.12,
  fruitDropPerTree: 0.55,
  maxFruitAroundTree: 4,
  maxInventory: 50,
  fruitEnergyGain: 8,
  tradeAmount: 1,
}

export function normalizeGameConfig(input: GameConfig): GameConfig {
  const width = clamp(input.width, 8, 200)
  const height = clamp(input.height, 8, 200)
  const agents = Math.min(clamp(input.agents, 1, 30000), width * height)
  const maxAgentSlots = Math.min(clamp(input.maxAgentSlots, agents, 50000), width * height)
  const initialEnergy = clamp(input.initialEnergy, 1, 500)
  const maxEnergy = clamp(input.maxEnergy, initialEnergy, 1200)
  const minNeuronCount = clamp(input.minNeuronCount, 1, 64)
  const maxNeuronCount = clamp(input.maxNeuronCount, minNeuronCount, 128)
  return {
    width,
    height,
    agents,
    maxAgentSlots,
    speed: clamp(input.speed, 16, 500),
    initialEnergy,
    maxEnergy,
    moveEnergyCost: clamp(input.moveEnergyCost, 0.05, 20),
    idleEnergyCost: clamp(input.idleEnergyCost, 0.01, 10),
    neuronStepEnergyCost: clamp(input.neuronStepEnergyCost, 0, 3),
    minNeuronCount,
    maxNeuronCount,
    cloneEnergyThreshold: clamp(input.cloneEnergyThreshold, 1, 5000),
    cloneEnergyCost: clamp(input.cloneEnergyCost, 1, 2000),
    cloneChancePerTick: clamp(input.cloneChancePerTick, 0, 1),
    treeDensity: clamp(input.treeDensity, 0, 1),
    fruitDropPerTree: clamp(input.fruitDropPerTree, 0, 1),
    maxFruitAroundTree: clamp(input.maxFruitAroundTree, 1, 4),
    maxInventory: clamp(input.maxInventory, 1, 300),
    fruitEnergyGain: clamp(input.fruitEnergyGain, 0.1, 100),
    tradeAmount: clamp(input.tradeAmount, 1, 20),
  }
}

export function toSimulationConfig(config: GameConfig): SimulationConfig {
  return {
    width: config.width,
    height: config.height,
    agentCount: config.agents,
    maxAgentSlots: config.maxAgentSlots,
    initialEnergy: config.initialEnergy,
    maxEnergy: config.maxEnergy,
    moveEnergyCost: config.moveEnergyCost,
    idleEnergyCost: config.idleEnergyCost,
    neuronStepEnergyCost: config.neuronStepEnergyCost,
    minNeuronCount: config.minNeuronCount,
    maxNeuronCount: config.maxNeuronCount,
    cloneEnergyThreshold: config.cloneEnergyThreshold,
    cloneEnergyCost: config.cloneEnergyCost,
    cloneChancePerTick: config.cloneChancePerTick,
    treeDensity: config.treeDensity,
    fruitDropPerTree: config.fruitDropPerTree,
    maxFruitAroundTree: config.maxFruitAroundTree,
    maxInventory: config.maxInventory,
    fruitEnergyGain: config.fruitEnergyGain,
    tradeAmount: config.tradeAmount,
  }
}

export function createInitialStats(config: GameConfig): TickStats {
  return {
    tick: 0,
    movedAgents: 0,
    occupancy: config.agents / (config.width * config.height),
    aliveAgents: config.agents,
    deathsThisTick: 0,
    avgEnergy: config.initialEnergy,
    avgInventory: 0,
    collectedFruit: 0,
    eatenFruit: 0,
    tradedFruit: 0,
    groundFruitTotal: 0,
    groundApple: 0,
    groundBanana: 0,
    groundOrange: 0,
    avgVitaminA: 0,
    avgVitaminB: 0,
    avgVitaminC: 0,
    deficientAgents: 0,
    actionHistogram: emptyActionHistogram(),
  }
}

function clamp(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) {
    return min
  }
  return Math.max(min, Math.min(max, value))
}
