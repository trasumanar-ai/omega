import { GridWorld } from './gridWorld'
import {
  chooseActionIndexFromGenome,
  cloneGenomeWithMutation,
  createRandomGenome,
  type AgentGenome,
  type GenomeGenerationConfig,
} from './genome'
import { Rng } from './rng'
import {
  ACTION_SPACE,
  emptyActionHistogram,
  encodeObservation,
  indexToAction,
  type ActionProvider,
  type ActionType,
  type AgentAction,
  type AgentObservation,
  type AgentSnapshot,
  type Coord,
  type FruitType,
  type SimulationConfig,
  type TickStats,
} from './types'

const NONE = 0
const APPLE = 1
const BANANA = 2
const ORANGE = 3

const CODE_TO_FRUIT: FruitType[] = ['none', 'apple', 'banana', 'orange']

type ActionEffect = {
  moved: boolean
  collectedFruit: number
  eatenFruit: number
  tradedFruit: number
}

export class Simulation {
  readonly config: SimulationConfig
  readonly grid: GridWorld
  readonly agentIds: number[]

  private readonly rng: Rng
  private readonly initialAgentCount: number
  private readonly maxAgentSlots: number
  private readonly alive: Uint8Array
  private readonly energy: Float32Array

  private readonly inventoryTotal: Float32Array
  private readonly inventoryApple: Float32Array
  private readonly inventoryBanana: Float32Array
  private readonly inventoryOrange: Float32Array

  private readonly vitaminA: Float32Array
  private readonly vitaminB: Float32Array
  private readonly vitaminC: Float32Array

  private readonly bornTick: Int32Array
  private readonly deathTick: Int32Array
  private readonly actionLog: ActionType[][]
  private readonly genomes: AgentGenome[]
  private readonly observationSize: number
  private readonly genomeGenerationConfig: GenomeGenerationConfig
  private readonly deadAgentSlots: number[] = []

  private readonly treeTypeByCell: Uint8Array
  private readonly groundFruitTypeByCell: Uint8Array
  private readonly treeIndices: number[] = []

  private readonly initialEnergy: number
  private readonly maxEnergy: number
  private readonly moveEnergyCost: number
  private readonly idleEnergyCost: number
  private readonly neuronStepEnergyCost: number
  private readonly cloneEnergyThreshold: number
  private readonly cloneEnergyCost: number
  private readonly cloneChancePerTick: number

  private readonly treeDensity: number
  private readonly fruitDropChancePerTree: number
  private readonly maxFruitAroundTree: number
  private readonly maxInventory: number
  private readonly fruitEnergyGain: number
  private readonly tradeAmount: number

  private readonly initialVitamin: number
  private readonly maxVitamin: number
  private readonly vitaminDecayPerTick: number
  private readonly vitaminGainPerFruit: number
  private readonly vitaminNeedThreshold: number
  private readonly vitaminDeficiencyEnergyPenalty: number

  private tickCount = 0

  constructor(config: SimulationConfig, seed = 42) {
    if (config.width <= 0 || config.height <= 0) {
      throw new Error('Grid dimensions must be positive.')
    }
    if (config.agentCount <= 0) {
      throw new Error('agentCount must be positive.')
    }
    const cellCount = config.width * config.height
    if (config.agentCount > cellCount) {
      throw new Error('agentCount cannot exceed cell count.')
    }

    this.config = config
    this.rng = new Rng(seed)
    this.initialAgentCount = config.agentCount
    this.maxAgentSlots = Math.min(
      cellCount,
      Math.max(config.agentCount, Math.floor(positive(config.maxAgentSlots, Math.ceil(config.agentCount * 1.5)))),
    )

    this.initialEnergy = positive(config.initialEnergy, 40)
    this.maxEnergy = positive(config.maxEnergy, this.initialEnergy * 3)
    this.moveEnergyCost = positive(config.moveEnergyCost, 1)
    this.idleEnergyCost = positive(config.idleEnergyCost, 0.25)
    this.neuronStepEnergyCost = nonNegative(config.neuronStepEnergyCost, 0.05)
    this.cloneEnergyThreshold = positive(config.cloneEnergyThreshold, this.initialEnergy * 1.8)
    this.cloneEnergyCost = positive(config.cloneEnergyCost, this.initialEnergy * 0.7)
    this.cloneChancePerTick = ratio(config.cloneChancePerTick, 1)

    const minNeuronCount = Math.max(1, Math.floor(positive(config.minNeuronCount, 4)))
    const maxNeuronCount = Math.max(minNeuronCount, Math.floor(positive(config.maxNeuronCount, 14)))
    this.genomeGenerationConfig = {
      minNeuronCount,
      maxNeuronCount,
      mutationRateBase: ratio(config.mutationRateBase, 0.08),
      mutationScaleBase: positive(config.mutationScaleBase, 0.2),
      addNeuronChanceBase: ratio(config.addNeuronChanceBase, 0.06),
      removeNeuronChanceBase: ratio(config.removeNeuronChanceBase, 0.04),
    }

    this.treeDensity = ratio(config.treeDensity, 0.08)
    this.fruitDropChancePerTree = ratio(config.fruitDropPerTree, 0.35)
    this.maxFruitAroundTree = Math.max(1, Math.min(4, Math.floor(positive(config.maxFruitAroundTree, 4))))
    this.maxInventory = positive(config.maxInventory, 50)
    this.fruitEnergyGain = positive(config.fruitEnergyGain, 6)
    this.tradeAmount = Math.max(1, Math.floor(positive(config.tradeAmount, 1)))

    this.initialVitamin = positive(config.initialVitamin, 60)
    this.maxVitamin = positive(config.maxVitamin, 100)
    this.vitaminDecayPerTick = nonNegative(config.vitaminDecayPerTick, 0.45)
    this.vitaminGainPerFruit = positive(config.vitaminGainPerFruit, 26)
    this.vitaminNeedThreshold = positive(config.vitaminNeedThreshold, 25)
    this.vitaminDeficiencyEnergyPenalty = nonNegative(config.vitaminDeficiencyEnergyPenalty, 0.6)

    this.grid = new GridWorld(config.width, config.height, this.maxAgentSlots)
    this.agentIds = Array.from({ length: this.maxAgentSlots }, (_, idx) => idx)
    this.alive = new Uint8Array(this.maxAgentSlots)
    this.energy = new Float32Array(this.maxAgentSlots)
    this.inventoryTotal = new Float32Array(this.maxAgentSlots)
    this.inventoryApple = new Float32Array(this.maxAgentSlots)
    this.inventoryBanana = new Float32Array(this.maxAgentSlots)
    this.inventoryOrange = new Float32Array(this.maxAgentSlots)
    this.vitaminA = new Float32Array(this.maxAgentSlots)
    this.vitaminB = new Float32Array(this.maxAgentSlots)
    this.vitaminC = new Float32Array(this.maxAgentSlots)
    this.bornTick = new Int32Array(this.maxAgentSlots)
    this.deathTick = new Int32Array(this.maxAgentSlots).fill(-1)
    this.actionLog = Array.from({ length: this.maxAgentSlots }, () => [])

    this.treeTypeByCell = new Uint8Array(cellCount)
    this.groundFruitTypeByCell = new Uint8Array(cellCount)

    this.seedTrees()
    this.seedAgentState()
    this.seedAgents()
    this.observationSize = this.getObservationVector(0).length
    this.genomes = Array.from(
      { length: this.maxAgentSlots },
      () => createRandomGenome(this.rng, this.observationSize, this.genomeGenerationConfig),
    )
  }

  get tick(): number {
    return this.tickCount
  }

  getActionSpace(): readonly string[] {
    return ACTION_SPACE
  }

  getObservationVector(agentId: number): number[] {
    return encodeObservation(this.getObservation(agentId))
  }

  getValidActionIndices(agentId: number): number[] {
    return this.getValidActions(agentId).map(action => ACTION_SPACE.indexOf(action.type))
  }

  hasTreeAt(x: number, y: number): boolean {
    if (!this.inBounds(x, y)) {
      return false
    }
    return this.treeTypeByCell[this.indexXY(x, y)] !== NONE
  }

  getTreeKindAt(x: number, y: number): FruitType {
    if (!this.inBounds(x, y)) {
      return 'none'
    }
    return CODE_TO_FRUIT[this.treeTypeByCell[this.indexXY(x, y)]]
  }

  getGroundFruitAt(x: number, y: number): number {
    if (!this.inBounds(x, y)) {
      return 0
    }
    return this.groundFruitTypeByCell[this.indexXY(x, y)] === NONE ? 0 : 1
  }

  getGroundFruitKindAt(x: number, y: number): FruitType {
    if (!this.inBounds(x, y)) {
      return 'none'
    }
    return CODE_TO_FRUIT[this.groundFruitTypeByCell[this.indexXY(x, y)]]
  }

  getTreeIndices(): readonly number[] {
    return this.treeIndices
  }

  step(actionProvider?: ActionProvider): TickStats {
    this.tickCount += 1
    this.dropFruitFromTrees()

    let movedAgents = 0
    let deathsThisTick = 0
    let collectedFruit = 0
    let eatenFruit = 0
    let tradedFruit = 0
    const actionHistogram = emptyActionHistogram()

    this.rng.shuffle(this.agentIds)
    for (const agentId of this.agentIds) {
      if (!this.isAgentAlive(agentId)) {
        continue
      }
      if (this.bornTick[agentId] === this.tickCount) {
        continue
      }

      this.decayVitamins(agentId)

      const observation = this.getObservation(agentId)
      const validActions = this.getValidActions(agentId)
      const chosenAction = actionProvider
        ? this.pickValidAction(actionProvider(agentId, observation, validActions), validActions)
        : this.selectActionFromGenome(agentId, observation, validActions)

      actionHistogram[chosenAction.type] += 1
      this.actionLog[agentId].push(chosenAction.type)

      const effect = this.executeAction(agentId, chosenAction)
      movedAgents += effect.moved ? 1 : 0
      collectedFruit += effect.collectedFruit
      eatenFruit += effect.eatenFruit
      tradedFruit += effect.tradedFruit

      const deficiencyCount = this.countVitaminDeficiencies(agentId)
      const cost =
        (effect.moved ? this.moveEnergyCost : this.idleEnergyCost) +
        deficiencyCount * this.vitaminDeficiencyEnergyPenalty +
        this.getNeuronEnergyCost(agentId)
      this.energy[agentId] = Math.max(0, this.energy[agentId] - cost)

      if (this.energy[agentId] <= 0) {
        this.markAgentDead(agentId)
        deathsThisTick += 1
      }
    }

    const living = this.summarizeLivingAgents()
    const groundCounts = this.countGroundFruit()
    return {
      tick: this.tickCount,
      movedAgents,
      occupancy: living.aliveAgents / (this.config.width * this.config.height),
      aliveAgents: living.aliveAgents,
      deathsThisTick,
      avgEnergy: living.aliveAgents > 0 ? living.totalEnergy / living.aliveAgents : 0,
      avgInventory: living.aliveAgents > 0 ? living.totalInventory / living.aliveAgents : 0,
      collectedFruit,
      eatenFruit,
      tradedFruit,
      groundFruitTotal: groundCounts.apple + groundCounts.banana + groundCounts.orange,
      groundApple: groundCounts.apple,
      groundBanana: groundCounts.banana,
      groundOrange: groundCounts.orange,
      avgVitaminA: living.aliveAgents > 0 ? living.totalVitaminA / living.aliveAgents : 0,
      avgVitaminB: living.aliveAgents > 0 ? living.totalVitaminB / living.aliveAgents : 0,
      avgVitaminC: living.aliveAgents > 0 ? living.totalVitaminC / living.aliveAgents : 0,
      deficientAgents: living.deficientAgents,
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
      inventory: this.inventoryTotal[agentId],
      inventoryByFruit: {
        apple: this.inventoryApple[agentId],
        banana: this.inventoryBanana[agentId],
        orange: this.inventoryOrange[agentId],
      },
      vitamins: {
        vitamin_a: this.vitaminA[agentId],
        vitamin_b: this.vitaminB[agentId],
        vitamin_c: this.vitaminC[agentId],
      },
      alive: this.isAgentAlive(agentId),
    }
  }

  isAgentAlive(agentId: number): boolean {
    return this.alive[agentId] === 1
  }

  getAgentDetail(agentId: number) {
    const state = this.getAgentState(agentId)
    const log = this.actionLog[agentId]
    const genome = this.genomes[agentId]
    const counts = emptyActionHistogram()
    for (const action of log) {
      counts[action] += 1
    }
    return {
      ...state,
      bornTick: this.bornTick[agentId],
      deathTick: this.deathTick[agentId],
      age: state.alive
        ? this.tickCount - this.bornTick[agentId]
        : this.deathTick[agentId] - this.bornTick[agentId],
      totalActions: log.length,
      actionCounts: counts,
      recentActions: log.slice(-20),
      neuronCount: genome.neurons.length,
      neuronEnergyCost: this.getNeuronEnergyCost(agentId),
      genome: {
        mutationRate: genome.mutationRate,
        mutationScale: genome.mutationScale,
        addNeuronChance: genome.addNeuronChance,
        removeNeuronChance: genome.removeNeuronChance,
        fruitBiasApple: genome.fruitBiasApple,
        fruitBiasBanana: genome.fruitBiasBanana,
        fruitBiasOrange: genome.fruitBiasOrange,
      },
    }
  }

  agentAt(x: number, y: number): number {
    if (!this.inBounds(x, y)) {
      return -1
    }
    return this.grid.occupancy[this.indexXY(x, y)]
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
        resources: {
          hasTree: 0,
          groundFruit: 0,
          treeType: 'none',
          groundFruitType: 'none',
          neighborFruitNorth: 0,
          neighborFruitEast: 0,
          neighborFruitSouth: 0,
          neighborFruitWest: 0,
        },
        social: {
          adjacentAgents: 0,
        },
      }
    }

    const pos = this.grid.getPosition(agentId)
    const idx = this.index(pos)
    const nFruit = this.groundFruitTypeByCell[this.indexXY(pos.x, pos.y - 1)] ?? NONE
    const eFruit = this.groundFruitTypeByCell[this.indexXY(pos.x + 1, pos.y)] ?? NONE
    const sFruit = this.groundFruitTypeByCell[this.indexXY(pos.x, pos.y + 1)] ?? NONE
    const wFruit = this.groundFruitTypeByCell[this.indexXY(pos.x - 1, pos.y)] ?? NONE
    return {
      tick: this.tickCount,
      self,
      neighbors: {
        north: this.sense(pos.x, pos.y - 1),
        east: this.sense(pos.x + 1, pos.y),
        south: this.sense(pos.x, pos.y + 1),
        west: this.sense(pos.x - 1, pos.y),
      },
      resources: {
        hasTree: this.treeTypeByCell[idx] === NONE ? 0 : 1,
        groundFruit: this.groundFruitTypeByCell[idx] === NONE ? 0 : 1,
        treeType: CODE_TO_FRUIT[this.treeTypeByCell[idx]],
        groundFruitType: CODE_TO_FRUIT[this.groundFruitTypeByCell[idx]],
        neighborFruitNorth: nFruit === NONE ? 0 : 1,
        neighborFruitEast: eFruit === NONE ? 0 : 1,
        neighborFruitSouth: sFruit === NONE ? 0 : 1,
        neighborFruitWest: wFruit === NONE ? 0 : 1,
      },
      social: {
        adjacentAgents: this.countAdjacentAliveAgents(pos),
      },
    }
  }

  getValidActions(agentId: number): readonly AgentAction[] {
    if (!this.isAgentAlive(agentId)) {
      return [{ type: 'stay' }]
    }

    const pos = this.grid.getPosition(agentId)
    const idx = this.index(pos)
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

    if (this.groundFruitTypeByCell[idx] !== NONE && this.inventoryTotal[agentId] < this.maxInventory) {
      valid.push({ type: 'collect_fruit' })
    }
    if (this.inventoryTotal[agentId] > 0) {
      valid.push({ type: 'eat_fruit' })
    }
    if (this.inventoryTotal[agentId] > 0 && this.countAdjacentAliveAgents(pos) > 0) {
      valid.push({ type: 'trade_fruit' })
    }
    if (this.canCloneSelf(agentId, pos)) {
      valid.push({ type: 'clone_self' })
    }

    return valid
  }

  private pickValidAction(chosen: AgentAction, validActions: readonly AgentAction[]): AgentAction {
    const valid = validActions.some(action => action.type === chosen.type)
    return valid ? chosen : { type: 'stay' }
  }

  private selectActionFromGenome(agentId: number, observation: AgentObservation, validActions: readonly AgentAction[]): AgentAction {
    const validActionIndices = validActions
      .map(action => ACTION_SPACE.indexOf(action.type))
      .filter(index => index >= 0)
    const featureVector = encodeObservation(observation)
    const chosenActionIndex = chooseActionIndexFromGenome(
      this.genomes[agentId],
      featureVector,
      validActionIndices,
      this.rng,
    )
    return this.pickValidAction(indexToAction(chosenActionIndex), validActions)
  }

  private executeAction(agentId: number, action: AgentAction): ActionEffect {
    const effect: ActionEffect = {
      moved: false,
      collectedFruit: 0,
      eatenFruit: 0,
      tradedFruit: 0,
    }
    if (!this.isAgentAlive(agentId)) {
      return effect
    }

    const current = this.grid.getPosition(agentId)
    const idx = this.index(current)
    switch (action.type) {
      case 'stay':
        return effect
      case 'move_north':
        effect.moved = this.grid.move(agentId, { x: current.x, y: current.y - 1 })
        return effect
      case 'move_east':
        effect.moved = this.grid.move(agentId, { x: current.x + 1, y: current.y })
        return effect
      case 'move_south':
        effect.moved = this.grid.move(agentId, { x: current.x, y: current.y + 1 })
        return effect
      case 'move_west':
        effect.moved = this.grid.move(agentId, { x: current.x - 1, y: current.y })
        return effect
      case 'collect_fruit': {
        const code = this.groundFruitTypeByCell[idx]
        if (code === NONE || this.inventoryTotal[agentId] >= this.maxInventory) {
          return effect
        }
        this.groundFruitTypeByCell[idx] = NONE
        this.addFruitToInventory(agentId, code, 1)
        effect.collectedFruit = 1
        return effect
      }
      case 'eat_fruit': {
        const code = this.pickFruitToEat(agentId)
        if (code === NONE) {
          return effect
        }
        this.removeFruitFromInventory(agentId, code, 1)
        this.energy[agentId] = Math.min(this.maxEnergy, this.energy[agentId] + this.fruitEnergyGain)
        this.gainVitaminFromFruit(agentId, code, 1)
        effect.eatenFruit = 1
        return effect
      }
      case 'trade_fruit':
        effect.tradedFruit = this.tradeFruitWithNeighbor(agentId, current)
        return effect
      case 'clone_self':
        this.cloneSelf(agentId, current)
        return effect
    }
  }

  private tradeFruitWithNeighbor(agentId: number, from: Coord): number {
    const neighbors = this.grid.neighbors4(from)
    const candidates: number[] = []
    for (const cell of neighbors) {
      const other = this.grid.occupancy[this.index(cell)]
      if (other >= 0 && this.isAgentAlive(other)) {
        candidates.push(other)
      }
    }
    if (candidates.length === 0) {
      return 0
    }

    const receiverId = candidates[this.rng.int(0, candidates.length)]
    const code = this.pickFruitToTrade(agentId)
    if (code === NONE) {
      return 0
    }

    const senderStock = this.getFruitInventory(agentId, code)
    const receiverCapacity = this.maxInventory - this.inventoryTotal[receiverId]
    const amount = Math.min(this.tradeAmount, senderStock, receiverCapacity)
    if (amount <= 0) {
      return 0
    }

    this.removeFruitFromInventory(agentId, code, amount)
    this.addFruitToInventory(receiverId, code, amount)
    return amount
  }

  private canCloneSelf(agentId: number, from: Coord): boolean {
    if (!this.isAgentAlive(agentId)) {
      return false
    }
    if (this.deadAgentSlots.length === 0) {
      return false
    }
    if (this.energy[agentId] < this.cloneEnergyThreshold) {
      return false
    }
    if (this.energy[agentId] <= this.cloneEnergyCost + this.getNeuronEnergyCost(agentId)) {
      return false
    }
    return this.getEmptyNeighborCells(from).length > 0
  }

  private cloneSelf(agentId: number, from: Coord): boolean {
    if (!this.canCloneSelf(agentId, from)) {
      return false
    }
    if (this.rng.next() > this.cloneChancePerTick) {
      return false
    }

    const spawnCandidates = this.getEmptyNeighborCells(from)
    if (spawnCandidates.length === 0) {
      return false
    }
    const childId = this.takeDeadAgentSlot()
    if (childId < 0) {
      return false
    }

    const spawn = spawnCandidates[this.rng.int(0, spawnCandidates.length)]
    if (!this.grid.isEmpty(spawn)) {
      this.deadAgentSlots.push(childId)
      return false
    }
    const transferredEnergy = Math.max(1, Math.min(this.cloneEnergyCost, this.energy[agentId] - 1))
    this.energy[agentId] = Math.max(1, this.energy[agentId] - transferredEnergy)

    this.alive[childId] = 1
    this.energy[childId] = transferredEnergy
    this.inventoryTotal[childId] = 0
    this.inventoryApple[childId] = 0
    this.inventoryBanana[childId] = 0
    this.inventoryOrange[childId] = 0
    this.vitaminA[childId] = Math.max(0, this.vitaminA[agentId] * 0.8)
    this.vitaminB[childId] = Math.max(0, this.vitaminB[agentId] * 0.8)
    this.vitaminC[childId] = Math.max(0, this.vitaminC[agentId] * 0.8)
    this.bornTick[childId] = this.tickCount
    this.deathTick[childId] = -1
    this.actionLog[childId].length = 0
    this.genomes[childId] = cloneGenomeWithMutation(
      this.genomes[agentId],
      this.rng,
      this.observationSize,
      this.genomeGenerationConfig,
    )
    this.grid.place(childId, spawn)
    return true
  }

  private getEmptyNeighborCells(from: Coord): Coord[] {
    return this.grid.neighbors4(from).filter(cell => this.grid.occupancy[this.index(cell)] === -1)
  }

  private takeDeadAgentSlot(): number {
    if (this.deadAgentSlots.length === 0) {
      return -1
    }
    const index = this.rng.int(0, this.deadAgentSlots.length)
    const [slot] = this.deadAgentSlots.splice(index, 1)
    return slot ?? -1
  }

  private markAgentDead(agentId: number): void {
    if (!this.isAgentAlive(agentId)) {
      return
    }
    this.alive[agentId] = 0
    this.deathTick[agentId] = this.tickCount
    this.grid.remove(agentId)
    this.deadAgentSlots.push(agentId)
  }

  private getNeuronEnergyCost(agentId: number): number {
    return this.genomes[agentId].neurons.length * this.neuronStepEnergyCost
  }

  private pickFruitToEat(agentId: number): number {
    if (this.inventoryTotal[agentId] <= 0) {
      return NONE
    }

    const candidates: Array<{ code: number; score: number }> = [
      {
        code: APPLE,
        score:
          (this.vitaminNeedThreshold - this.vitaminA[agentId]) * this.getFruitBias(agentId, APPLE) +
          this.inventoryApple[agentId] * 0.1,
      },
      {
        code: BANANA,
        score:
          (this.vitaminNeedThreshold - this.vitaminB[agentId]) * this.getFruitBias(agentId, BANANA) +
          this.inventoryBanana[agentId] * 0.1,
      },
      {
        code: ORANGE,
        score:
          (this.vitaminNeedThreshold - this.vitaminC[agentId]) * this.getFruitBias(agentId, ORANGE) +
          this.inventoryOrange[agentId] * 0.1,
      },
    ].filter(item => this.getFruitInventory(agentId, item.code) > 0)

    if (candidates.length === 0) {
      return NONE
    }

    candidates.sort((a, b) => b.score - a.score)
    return candidates[0].code
  }

  private pickFruitToTrade(agentId: number): number {
    const apple = this.inventoryApple[agentId]
    const banana = this.inventoryBanana[agentId]
    const orange = this.inventoryOrange[agentId]
    if (apple <= 0 && banana <= 0 && orange <= 0) {
      return NONE
    }

    const options: Array<{ code: number; pressure: number }> = [
      { code: APPLE, pressure: apple / Math.max(0.1, this.getFruitBias(agentId, APPLE)) },
      { code: BANANA, pressure: banana / Math.max(0.1, this.getFruitBias(agentId, BANANA)) },
      { code: ORANGE, pressure: orange / Math.max(0.1, this.getFruitBias(agentId, ORANGE)) },
    ].filter(item => this.getFruitInventory(agentId, item.code) > 0)

    if (options.length === 0) {
      return NONE
    }

    options.sort((a, b) => b.pressure - a.pressure)
    return options[0].code
  }

  private addFruitToInventory(agentId: number, code: number, amount: number): void {
    if (amount <= 0) {
      return
    }
    this.inventoryTotal[agentId] += amount
    if (code === APPLE) {
      this.inventoryApple[agentId] += amount
    } else if (code === BANANA) {
      this.inventoryBanana[agentId] += amount
    } else if (code === ORANGE) {
      this.inventoryOrange[agentId] += amount
    }
  }

  private removeFruitFromInventory(agentId: number, code: number, amount: number): void {
    if (amount <= 0) {
      return
    }
    this.inventoryTotal[agentId] = Math.max(0, this.inventoryTotal[agentId] - amount)
    if (code === APPLE) {
      this.inventoryApple[agentId] = Math.max(0, this.inventoryApple[agentId] - amount)
    } else if (code === BANANA) {
      this.inventoryBanana[agentId] = Math.max(0, this.inventoryBanana[agentId] - amount)
    } else if (code === ORANGE) {
      this.inventoryOrange[agentId] = Math.max(0, this.inventoryOrange[agentId] - amount)
    }
  }

  private getFruitInventory(agentId: number, code: number): number {
    if (code === APPLE) {
      return this.inventoryApple[agentId]
    }
    if (code === BANANA) {
      return this.inventoryBanana[agentId]
    }
    if (code === ORANGE) {
      return this.inventoryOrange[agentId]
    }
    return 0
  }

  private getFruitBias(agentId: number, code: number): number {
    const genome = this.genomes[agentId]
    if (code === APPLE) {
      return genome.fruitBiasApple
    }
    if (code === BANANA) {
      return genome.fruitBiasBanana
    }
    if (code === ORANGE) {
      return genome.fruitBiasOrange
    }
    return 1
  }

  private gainVitaminFromFruit(agentId: number, code: number, amount: number): void {
    if (code === APPLE) {
      this.vitaminA[agentId] = Math.min(this.maxVitamin, this.vitaminA[agentId] + this.vitaminGainPerFruit * amount)
      return
    }
    if (code === BANANA) {
      this.vitaminB[agentId] = Math.min(this.maxVitamin, this.vitaminB[agentId] + this.vitaminGainPerFruit * amount)
      return
    }
    if (code === ORANGE) {
      this.vitaminC[agentId] = Math.min(this.maxVitamin, this.vitaminC[agentId] + this.vitaminGainPerFruit * amount)
    }
  }

  private decayVitamins(agentId: number): void {
    this.vitaminA[agentId] = Math.max(0, this.vitaminA[agentId] - this.vitaminDecayPerTick)
    this.vitaminB[agentId] = Math.max(0, this.vitaminB[agentId] - this.vitaminDecayPerTick)
    this.vitaminC[agentId] = Math.max(0, this.vitaminC[agentId] - this.vitaminDecayPerTick)
  }

  private countVitaminDeficiencies(agentId: number): number {
    let count = 0
    if (this.vitaminA[agentId] < this.vitaminNeedThreshold) count += 1
    if (this.vitaminB[agentId] < this.vitaminNeedThreshold) count += 1
    if (this.vitaminC[agentId] < this.vitaminNeedThreshold) count += 1
    return count
  }

  private countAdjacentAliveAgents(from: Coord): number {
    let count = 0
    for (const cell of this.grid.neighbors4(from)) {
      const other = this.grid.occupancy[this.index(cell)]
      if (other >= 0 && this.isAgentAlive(other)) {
        count += 1
      }
    }
    return count
  }

  private dropFruitFromTrees(): void {
    if (this.fruitDropChancePerTree <= 0 || this.treeIndices.length === 0) {
      return
    }

    for (const treeIdx of this.treeIndices) {
      if (this.rng.next() > this.fruitDropChancePerTree) {
        continue
      }

      const coord = this.coordFromIndex(treeIdx)
      const slots = this.grid.neighbors4(coord)
      let fruitAround = 0
      const available: Coord[] = []

      for (const slot of slots) {
        const idx = this.index(slot)
        if (this.groundFruitTypeByCell[idx] !== NONE) {
          fruitAround += 1
        } else {
          available.push(slot)
        }
      }

      if (fruitAround >= this.maxFruitAroundTree || available.length === 0) {
        continue
      }

      const target = available[this.rng.int(0, available.length)]
      this.groundFruitTypeByCell[this.index(target)] = this.treeTypeByCell[treeIdx]
    }
  }

  private countGroundFruit(): { apple: number; banana: number; orange: number } {
    let apple = 0
    let banana = 0
    let orange = 0
    for (let idx = 0; idx < this.groundFruitTypeByCell.length; idx += 1) {
      const code = this.groundFruitTypeByCell[idx]
      if (code === APPLE) apple += 1
      else if (code === BANANA) banana += 1
      else if (code === ORANGE) orange += 1
    }
    return { apple, banana, orange }
  }

  private summarizeLivingAgents(): {
    aliveAgents: number
    totalEnergy: number
    totalInventory: number
    totalVitaminA: number
    totalVitaminB: number
    totalVitaminC: number
    deficientAgents: number
  } {
    let aliveAgents = 0
    let totalEnergy = 0
    let totalInventory = 0
    let totalVitaminA = 0
    let totalVitaminB = 0
    let totalVitaminC = 0
    let deficientAgents = 0

    for (const agentId of this.agentIds) {
      if (!this.isAgentAlive(agentId)) {
        continue
      }
      aliveAgents += 1
      totalEnergy += this.energy[agentId]
      totalInventory += this.inventoryTotal[agentId]
      totalVitaminA += this.vitaminA[agentId]
      totalVitaminB += this.vitaminB[agentId]
      totalVitaminC += this.vitaminC[agentId]
      if (this.countVitaminDeficiencies(agentId) > 0) {
        deficientAgents += 1
      }
    }

    return {
      aliveAgents,
      totalEnergy,
      totalInventory,
      totalVitaminA,
      totalVitaminB,
      totalVitaminC,
      deficientAgents,
    }
  }

  private sense(x: number, y: number): number {
    if (!this.inBounds(x, y)) {
      return -1
    }
    return this.grid.occupancy[this.indexXY(x, y)] === -1 ? 0 : 1
  }

  private seedTrees(): void {
    const cells = this.config.width * this.config.height
    for (let idx = 0; idx < cells; idx += 1) {
      if (this.rng.next() < this.treeDensity) {
        this.treeTypeByCell[idx] = this.randomFruitCode()
        this.treeIndices.push(idx)
      }
    }

    if (this.treeDensity > 0 && this.treeIndices.length === 0) {
      const idx = this.rng.int(0, cells)
      this.treeTypeByCell[idx] = this.randomFruitCode()
      this.treeIndices.push(idx)
    }
  }

  private seedAgentState(): void {
    this.deadAgentSlots.length = 0
    for (const agentId of this.agentIds) {
      const active = agentId < this.initialAgentCount
      this.alive[agentId] = active ? 1 : 0
      this.energy[agentId] = active ? this.initialEnergy : 0
      this.inventoryTotal[agentId] = 0
      this.inventoryApple[agentId] = 0
      this.inventoryBanana[agentId] = 0
      this.inventoryOrange[agentId] = 0
      this.vitaminA[agentId] = active ? this.initialVitamin : 0
      this.vitaminB[agentId] = active ? this.initialVitamin : 0
      this.vitaminC[agentId] = active ? this.initialVitamin : 0
      this.bornTick[agentId] = 0
      this.deathTick[agentId] = -1
      this.actionLog[agentId].length = 0
      if (!active) {
        this.deadAgentSlots.push(agentId)
      }
    }
  }

  private seedAgents(): void {
    for (let agentId = 0; agentId < this.initialAgentCount; agentId += 1) {
      this.grid.place(agentId, this.randomEmptyCell())
    }
  }

  private randomEmptyCell(): Coord {
    while (true) {
      const candidate = {
        x: this.rng.int(0, this.config.width),
        y: this.rng.int(0, this.config.height),
      }
      if (this.grid.isEmpty(candidate)) {
        return candidate
      }
    }
  }

  private randomFruitCode(): number {
    return this.rng.int(APPLE, ORANGE + 1)
  }

  private inBounds(x: number, y: number): boolean {
    return x >= 0 && x < this.config.width && y >= 0 && y < this.config.height
  }

  private index(coord: Coord): number {
    return this.indexXY(coord.x, coord.y)
  }

  private indexXY(x: number, y: number): number {
    return y * this.config.width + x
  }

  private coordFromIndex(index: number): Coord {
    const x = index % this.config.width
    const y = Math.floor(index / this.config.width)
    return { x, y }
  }
}

function positive(value: number | undefined, fallback: number): number {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    return fallback
  }
  return value
}

function nonNegative(value: number | undefined, fallback: number): number {
  if (typeof value !== 'number' || !Number.isFinite(value) || value < 0) {
    return fallback
  }
  return value
}

function ratio(value: number | undefined, fallback: number): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return fallback
  }
  return Math.max(0, Math.min(1, value))
}
