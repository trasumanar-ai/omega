package sim

import "math"

type NeuronGene struct {
	FeatureIndex int     `json:"featureIndex"`
	ActionIndex  int     `json:"actionIndex"`
	Weight       float64 `json:"weight"`
	Bias         float64 `json:"bias"`
	Gain         float64 `json:"gain"`
}

type AgentGenome struct {
	Neurons            []NeuronGene `json:"neurons"`
	MutationRate       float64      `json:"mutationRate"`
	MutationScale      float64      `json:"mutationScale"`
	AddNeuronChance    float64      `json:"addNeuronChance"`
	RemoveNeuronChance float64      `json:"removeNeuronChance"`
	FruitBiasApple     float64      `json:"fruitBiasApple"`
	FruitBiasBanana    float64      `json:"fruitBiasBanana"`
	FruitBiasOrange    float64      `json:"fruitBiasOrange"`
}

type GenomeGenerationConfig struct {
	MinNeuronCount         int
	MaxNeuronCount         int
	MutationRateBase       float64
	MutationScaleBase      float64
	AddNeuronChanceBase    float64
	RemoveNeuronChanceBase float64
}

func CreateRandomGenome(rng *RNG, observationSize int, cfg GenomeGenerationConfig) AgentGenome {
	neuronCount := rng.Int(cfg.MinNeuronCount, cfg.MaxNeuronCount+1)
	neurons := make([]NeuronGene, 0, neuronCount)
	for i := 0; i < neuronCount; i++ {
		neurons = append(neurons, randomNeuron(rng, observationSize))
	}
	return AgentGenome{
		Neurons:            neurons,
		MutationRate:       clamp(cfg.MutationRateBase+jitter(rng, 0.04), 0.005, 0.45),
		MutationScale:      clamp(cfg.MutationScaleBase+jitter(rng, 0.15), 0.01, 1.5),
		AddNeuronChance:    clamp(cfg.AddNeuronChanceBase+jitter(rng, 0.08), 0, 0.6),
		RemoveNeuronChance: clamp(cfg.RemoveNeuronChanceBase+jitter(rng, 0.08), 0, 0.6),
		FruitBiasApple:     clamp(1+jitter(rng, 0.35), 0.2, 2.5),
		FruitBiasBanana:    clamp(1+jitter(rng, 0.35), 0.2, 2.5),
		FruitBiasOrange:    clamp(1+jitter(rng, 0.35), 0.2, 2.5),
	}
}

func CloneGenomeWithMutation(parent AgentGenome, rng *RNG, observationSize int, cfg GenomeGenerationConfig) AgentGenome {
	child := AgentGenome{
		Neurons:            make([]NeuronGene, len(parent.Neurons)),
		MutationRate:       parent.MutationRate,
		MutationScale:      parent.MutationScale,
		AddNeuronChance:    parent.AddNeuronChance,
		RemoveNeuronChance: parent.RemoveNeuronChance,
		FruitBiasApple:     parent.FruitBiasApple,
		FruitBiasBanana:    parent.FruitBiasBanana,
		FruitBiasOrange:    parent.FruitBiasOrange,
	}
	copy(child.Neurons, parent.Neurons)

	mutateScalar := func(value float64) float64 {
		return clamp(value+jitter(rng, child.MutationScale), 0.001, 3)
	}

	if rng.Next() < child.MutationRate {
		child.MutationRate = clamp(mutateScalar(child.MutationRate), 0.005, 0.45)
	}
	if rng.Next() < child.MutationRate {
		child.MutationScale = clamp(mutateScalar(child.MutationScale), 0.01, 1.5)
	}
	if rng.Next() < child.MutationRate {
		child.AddNeuronChance = clamp(mutateScalar(child.AddNeuronChance), 0, 0.6)
	}
	if rng.Next() < child.MutationRate {
		child.RemoveNeuronChance = clamp(mutateScalar(child.RemoveNeuronChance), 0, 0.6)
	}
	if rng.Next() < child.MutationRate {
		child.FruitBiasApple = clamp(mutateScalar(child.FruitBiasApple), 0.1, 3)
	}
	if rng.Next() < child.MutationRate {
		child.FruitBiasBanana = clamp(mutateScalar(child.FruitBiasBanana), 0.1, 3)
	}
	if rng.Next() < child.MutationRate {
		child.FruitBiasOrange = clamp(mutateScalar(child.FruitBiasOrange), 0.1, 3)
	}

	actionCount := len(AllActions())
	for i := range child.Neurons {
		neuron := &child.Neurons[i]
		if rng.Next() < child.MutationRate {
			neuron.FeatureIndex = rng.Int(0, observationSize)
		}
		if rng.Next() < child.MutationRate {
			neuron.ActionIndex = rng.Int(0, actionCount)
		}
		if rng.Next() < child.MutationRate {
			neuron.Weight += jitter(rng, child.MutationScale)
		}
		if rng.Next() < child.MutationRate {
			neuron.Bias += jitter(rng, child.MutationScale*0.6)
		}
		if rng.Next() < child.MutationRate {
			neuron.Gain = clamp(neuron.Gain+jitter(rng, child.MutationScale*0.4), 0.05, 4)
		}
	}

	if rng.Next() < child.AddNeuronChance && len(child.Neurons) < cfg.MaxNeuronCount {
		child.Neurons = append(child.Neurons, randomNeuron(rng, observationSize))
	}
	if rng.Next() < child.RemoveNeuronChance && len(child.Neurons) > cfg.MinNeuronCount {
		removeIndex := rng.Int(0, len(child.Neurons))
		child.Neurons = append(child.Neurons[:removeIndex], child.Neurons[removeIndex+1:]...)
	}

	return child
}

func ChooseActionIndexFromGenome(genome AgentGenome, features []float64, validActionIndices []int, rng *RNG) int {
	if len(validActionIndices) == 0 {
		return 0
	}
	if len(genome.Neurons) == 0 {
		return validActionIndices[rng.Int(0, len(validActionIndices))]
	}

	scores := make([]float64, len(AllActions()))
	for _, neuron := range genome.Neurons {
		feature := 0.0
		if neuron.FeatureIndex >= 0 && neuron.FeatureIndex < len(features) {
			feature = features[neuron.FeatureIndex]
		}
		activation := math.Tanh(feature*neuron.Gain + neuron.Bias)
		if neuron.ActionIndex >= 0 && neuron.ActionIndex < len(scores) {
			scores[neuron.ActionIndex] += activation * neuron.Weight
		}
	}

	best := validActionIndices[0]
	bestScore := scores[best]
	for i := 1; i < len(validActionIndices); i++ {
		actionIndex := validActionIndices[i]
		score := scores[actionIndex]
		if score > bestScore+1e-8 {
			best = actionIndex
			bestScore = score
			continue
		}
		if math.Abs(score-bestScore) <= 1e-8 && rng.Next() < 0.5 {
			best = actionIndex
			bestScore = score
		}
	}
	return best
}

func randomNeuron(rng *RNG, observationSize int) NeuronGene {
	return NeuronGene{
		FeatureIndex: rng.Int(0, observationSize),
		ActionIndex:  rng.Int(0, len(AllActions())),
		Weight:       jitter(rng, 1.2),
		Bias:         jitter(rng, 0.8),
		Gain:         clamp(1+jitter(rng, 0.55), 0.05, 4),
	}
}

func jitter(rng *RNG, amplitude float64) float64 {
	return (rng.Next()*2 - 1) * amplitude
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
