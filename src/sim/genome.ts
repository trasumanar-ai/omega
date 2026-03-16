import { ACTION_SPACE } from './types'

export type NeuronGene = {
  featureIndex: number
  actionIndex: number
  weight: number
  bias: number
  gain: number
}

export type AgentGenome = {
  neurons: NeuronGene[]
  mutationRate: number
  mutationScale: number
  addNeuronChance: number
  removeNeuronChance: number
  fruitBiasApple: number
  fruitBiasBanana: number
  fruitBiasOrange: number
}

export type GenomeGenerationConfig = {
  minNeuronCount: number
  maxNeuronCount: number
  mutationRateBase: number
  mutationScaleBase: number
  addNeuronChanceBase: number
  removeNeuronChanceBase: number
}

type RngLike = {
  next(): number
  int(min: number, maxExclusive: number): number
}

export function createRandomGenome(
  rng: RngLike,
  observationSize: number,
  config: GenomeGenerationConfig,
): AgentGenome {
  const neuronCount = rng.int(config.minNeuronCount, config.maxNeuronCount + 1)
  const neurons: NeuronGene[] = []
  for (let i = 0; i < neuronCount; i += 1) {
    neurons.push(randomNeuron(rng, observationSize))
  }
  return {
    neurons,
    mutationRate: clamp(config.mutationRateBase + jitter(rng, 0.04), 0.005, 0.45),
    mutationScale: clamp(config.mutationScaleBase + jitter(rng, 0.15), 0.01, 1.5),
    addNeuronChance: clamp(config.addNeuronChanceBase + jitter(rng, 0.08), 0, 0.6),
    removeNeuronChance: clamp(config.removeNeuronChanceBase + jitter(rng, 0.08), 0, 0.6),
    fruitBiasApple: clamp(1 + jitter(rng, 0.35), 0.2, 2.5),
    fruitBiasBanana: clamp(1 + jitter(rng, 0.35), 0.2, 2.5),
    fruitBiasOrange: clamp(1 + jitter(rng, 0.35), 0.2, 2.5),
  }
}

export function cloneGenomeWithMutation(
  parent: AgentGenome,
  rng: RngLike,
  observationSize: number,
  config: GenomeGenerationConfig,
): AgentGenome {
  const child: AgentGenome = {
    neurons: parent.neurons.map(neuron => ({ ...neuron })),
    mutationRate: parent.mutationRate,
    mutationScale: parent.mutationScale,
    addNeuronChance: parent.addNeuronChance,
    removeNeuronChance: parent.removeNeuronChance,
    fruitBiasApple: parent.fruitBiasApple,
    fruitBiasBanana: parent.fruitBiasBanana,
    fruitBiasOrange: parent.fruitBiasOrange,
  }

  const mutateScalar = (value: number) =>
    clamp(value + jitter(rng, child.mutationScale), 0.001, 3)

  if (rng.next() < child.mutationRate) child.mutationRate = clamp(mutateScalar(child.mutationRate), 0.005, 0.45)
  if (rng.next() < child.mutationRate) child.mutationScale = clamp(mutateScalar(child.mutationScale), 0.01, 1.5)
  if (rng.next() < child.mutationRate) child.addNeuronChance = clamp(mutateScalar(child.addNeuronChance), 0, 0.6)
  if (rng.next() < child.mutationRate) child.removeNeuronChance = clamp(mutateScalar(child.removeNeuronChance), 0, 0.6)
  if (rng.next() < child.mutationRate) child.fruitBiasApple = clamp(mutateScalar(child.fruitBiasApple), 0.1, 3)
  if (rng.next() < child.mutationRate) child.fruitBiasBanana = clamp(mutateScalar(child.fruitBiasBanana), 0.1, 3)
  if (rng.next() < child.mutationRate) child.fruitBiasOrange = clamp(mutateScalar(child.fruitBiasOrange), 0.1, 3)

  for (const neuron of child.neurons) {
    if (rng.next() < child.mutationRate) neuron.featureIndex = rng.int(0, observationSize)
    if (rng.next() < child.mutationRate) neuron.actionIndex = rng.int(0, ACTION_SPACE.length)
    if (rng.next() < child.mutationRate) neuron.weight += jitter(rng, child.mutationScale)
    if (rng.next() < child.mutationRate) neuron.bias += jitter(rng, child.mutationScale * 0.6)
    if (rng.next() < child.mutationRate) neuron.gain = clamp(neuron.gain + jitter(rng, child.mutationScale * 0.4), 0.05, 4)
  }

  if (rng.next() < child.addNeuronChance && child.neurons.length < config.maxNeuronCount) {
    child.neurons.push(randomNeuron(rng, observationSize))
  }
  if (rng.next() < child.removeNeuronChance && child.neurons.length > config.minNeuronCount) {
    const removeIndex = rng.int(0, child.neurons.length)
    child.neurons.splice(removeIndex, 1)
  }

  return child
}

export function chooseActionIndexFromGenome(
  genome: AgentGenome,
  features: number[],
  validActionIndices: readonly number[],
  rng: RngLike,
): number {
  if (validActionIndices.length === 0) {
    return 0
  }
  if (genome.neurons.length === 0) {
    return validActionIndices[rng.int(0, validActionIndices.length)]
  }

  const scores = new Float32Array(ACTION_SPACE.length)
  for (const neuron of genome.neurons) {
    const feature = features[neuron.featureIndex] ?? 0
    const activation = Math.tanh(feature * neuron.gain + neuron.bias)
    scores[neuron.actionIndex] += activation * neuron.weight
  }

  let best = validActionIndices[0]
  let bestScore = scores[best]
  for (let i = 1; i < validActionIndices.length; i += 1) {
    const actionIndex = validActionIndices[i]
    const score = scores[actionIndex]
    if (score > bestScore + 1e-8) {
      best = actionIndex
      bestScore = score
      continue
    }
    if (Math.abs(score - bestScore) <= 1e-8 && rng.next() < 0.5) {
      best = actionIndex
      bestScore = score
    }
  }
  return best
}

function randomNeuron(rng: RngLike, observationSize: number): NeuronGene {
  return {
    featureIndex: rng.int(0, observationSize),
    actionIndex: rng.int(0, ACTION_SPACE.length),
    weight: jitter(rng, 1.2),
    bias: jitter(rng, 0.8),
    gain: clamp(1 + jitter(rng, 0.55), 0.05, 4),
  }
}

function jitter(rng: RngLike, amplitude: number): number {
  return (rng.next() * 2 - 1) * amplitude
}

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value))
}
