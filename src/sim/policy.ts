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
    if (!observation.self.alive) {
      return { type: 'stay' }
    }
    if (observation.self.energy <= 1) {
      return { type: 'stay' }
    }
    const movementActions = validActions.filter(action => action.type !== 'stay')
    if (movementActions.length === 0) {
      return { type: 'stay' }
    }
    if (this.random() < this.stayChance) {
      return { type: 'stay' }
    }
    return movementActions[Math.floor(this.random() * movementActions.length)]
  }
}
