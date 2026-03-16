import type { AgentAction, AgentObservation } from './types'

export type PolicyInput = {
  observation: AgentObservation
  validActions: readonly AgentAction[]
}

export interface AgentPolicy {
  selectAction(input: PolicyInput): AgentAction
}

export class RuleBasedPolicy implements AgentPolicy {
  private readonly random: () => number
  private readonly stayChance: number

  constructor(random: () => number, stayChance = 0.12) {
    this.random = random
    this.stayChance = stayChance
  }

  selectAction(input: PolicyInput): AgentAction {
    const { observation, validActions } = input
    const has = (type: AgentAction['type']) => validActions.some(action => action.type === type)

    if (!observation.self.alive) {
      return { type: 'stay' }
    }

    const vitaminNeed =
      Number(observation.self.vitamins.vitamin_a < 25) +
      Number(observation.self.vitamins.vitamin_b < 25) +
      Number(observation.self.vitamins.vitamin_c < 25)

    if (observation.resources.groundFruit > 0 && has('collect_fruit')) {
      return { type: 'collect_fruit' }
    }

    if (vitaminNeed > 0 && has('eat_fruit')) {
      return { type: 'eat_fruit' }
    }

    if (observation.resources.neighborFruitNorth > 0 && has('move_north')) {
      return { type: 'move_north' }
    }
    if (observation.resources.neighborFruitEast > 0 && has('move_east')) {
      return { type: 'move_east' }
    }
    if (observation.resources.neighborFruitSouth > 0 && has('move_south')) {
      return { type: 'move_south' }
    }
    if (observation.resources.neighborFruitWest > 0 && has('move_west')) {
      return { type: 'move_west' }
    }

    if (observation.self.energy <= 1 && has('eat_fruit')) {
      return { type: 'eat_fruit' }
    }
    if (observation.self.energy < 10 && has('eat_fruit')) {
      return { type: 'eat_fruit' }
    }
    if (observation.self.inventory > 3 && observation.social.adjacentAgents > 0 && has('trade_fruit') && this.random() < 0.35) {
      return { type: 'trade_fruit' }
    }
    if (observation.self.energy <= 1) {
      return { type: 'stay' }
    }

    const movementActions = validActions.filter(action => action.type.startsWith('move_'))
    if (movementActions.length === 0) {
      if (has('collect_fruit')) {
        return { type: 'collect_fruit' }
      }
      if (has('eat_fruit')) {
        return { type: 'eat_fruit' }
      }
      return { type: 'stay' }
    }
    if (this.random() < this.stayChance) {
      return { type: 'stay' }
    }
    return movementActions[Math.floor(this.random() * movementActions.length)]
  }
}
