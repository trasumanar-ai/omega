package sim

import (
	"errors"
	"fmt"
	"math"
)

type actionEffect struct {
	Moved          bool
	CollectedFruit int
	EatenFruit     int
	TradedFruit    int
}

type livingSummary struct {
	AliveAgents     int
	TotalEnergy     float64
	TotalInventory  float64
	TotalVitaminA   float64
	TotalVitaminB   float64
	TotalVitaminC   float64
	DeficientAgents int
}

type groundCounts struct {
	Apple  int
	Banana int
	Orange int
}

type Simulation struct {
	Config   SimulationConfig
	Grid     *GridWorld
	AgentIDs []int

	rng               *RNG
	initialAgentCount int
	maxAgentSlots     int
	alive             []bool
	energy            []float64

	inventoryTotal  []float64
	inventoryApple  []float64
	inventoryBanana []float64
	inventoryOrange []float64

	vitaminA []float64
	vitaminB []float64
	vitaminC []float64

	bornTick               []int
	deathTick              []int
	actionLog              [][]ActionType
	genomes                []AgentGenome
	observationSize        int
	genomeGenerationConfig GenomeGenerationConfig
	deadAgentSlots         []int

	treeTypeByCell        []FruitType
	groundFruitTypeByCell []FruitType
	treeIndices           []int

	initialEnergy                  float64
	maxEnergy                      float64
	moveEnergyCost                 float64
	idleEnergyCost                 float64
	neuronStepEnergyCost           float64
	cloneEnergyThreshold           float64
	cloneEnergyCost                float64
	cloneChancePerTick             float64
	treeDensity                    float64
	fruitDropChancePerTree         float64
	maxFruitAroundTree             int
	maxInventory                   float64
	fruitEnergyGain                float64
	tradeAmount                    int
	initialVitamin                 float64
	maxVitamin                     float64
	vitaminDecayPerTick            float64
	vitaminGainPerFruit            float64
	vitaminNeedThreshold           float64
	vitaminDeficiencyEnergyPenalty float64
	tickCount                      int
}

func NewSimulation(cfg SimulationConfig, seed uint32) (*Simulation, error) {
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, errors.New("grid dimensions must be positive")
	}
	if cfg.AgentCount <= 0 {
		return nil, errors.New("agentCount must be positive")
	}
	cellCount := cfg.Width * cfg.Height
	if cfg.AgentCount > cellCount {
		return nil, errors.New("agentCount cannot exceed cell count")
	}

	base := DefaultConfig()
	config := cfg
	if config.MaxAgentSlots <= 0 {
		config.MaxAgentSlots = int(math.Ceil(float64(cfg.AgentCount) * 1.5))
	}

	sim := &Simulation{
		Config:    config,
		rng:       NewRNG(seed),
		tickCount: 0,
	}

	sim.initialAgentCount = config.AgentCount
	sim.maxAgentSlots = int(minFloat(float64(cellCount), maxFloat(float64(config.AgentCount), float64(config.MaxAgentSlots))))

	sim.initialEnergy = positive(config.InitialEnergy, base.InitialEnergy)
	sim.maxEnergy = positive(config.MaxEnergy, sim.initialEnergy*3)
	sim.moveEnergyCost = positive(config.MoveEnergyCost, base.MoveEnergyCost)
	sim.idleEnergyCost = positive(config.IdleEnergyCost, base.IdleEnergyCost)
	sim.neuronStepEnergyCost = nonNegative(config.NeuronStepEnergyCost, base.NeuronStepEnergyCost)
	sim.cloneEnergyThreshold = positive(config.CloneEnergyThreshold, sim.initialEnergy*1.8)
	sim.cloneEnergyCost = positive(config.CloneEnergyCost, sim.initialEnergy*0.7)
	sim.cloneChancePerTick = ratio(config.CloneChancePerTick, base.CloneChancePerTick)

	minNeuronCount := int(maxFloat(1, math.Floor(positive(float64(config.MinNeuronCount), float64(base.MinNeuronCount)))))
	maxNeuronCount := int(maxFloat(float64(minNeuronCount), math.Floor(positive(float64(config.MaxNeuronCount), float64(base.MaxNeuronCount)))))
	sim.genomeGenerationConfig = GenomeGenerationConfig{
		MinNeuronCount:         minNeuronCount,
		MaxNeuronCount:         maxNeuronCount,
		MutationRateBase:       ratio(config.MutationRateBase, base.MutationRateBase),
		MutationScaleBase:      positive(config.MutationScaleBase, base.MutationScaleBase),
		AddNeuronChanceBase:    ratio(config.AddNeuronChanceBase, base.AddNeuronChanceBase),
		RemoveNeuronChanceBase: ratio(config.RemoveNeuronChanceBase, base.RemoveNeuronChanceBase),
	}

	sim.treeDensity = ratio(config.TreeDensity, base.TreeDensity)
	sim.fruitDropChancePerTree = ratio(config.FruitDropPerTree, base.FruitDropPerTree)
	sim.maxFruitAroundTree = int(maxFloat(1, minFloat(4, math.Floor(positive(float64(config.MaxFruitAroundTree), float64(base.MaxFruitAroundTree))))))
	sim.maxInventory = positive(config.MaxInventory, base.MaxInventory)
	sim.fruitEnergyGain = positive(config.FruitEnergyGain, base.FruitEnergyGain)
	sim.tradeAmount = int(maxFloat(1, math.Floor(positive(float64(config.TradeAmount), float64(base.TradeAmount)))))

	sim.initialVitamin = positive(config.InitialVitamin, base.InitialVitamin)
	sim.maxVitamin = positive(config.MaxVitamin, base.MaxVitamin)
	sim.vitaminDecayPerTick = nonNegative(config.VitaminDecayPerTick, base.VitaminDecayPerTick)
	sim.vitaminGainPerFruit = positive(config.VitaminGainPerFruit, base.VitaminGainPerFruit)
	sim.vitaminNeedThreshold = positive(config.VitaminNeedThreshold, base.VitaminNeedThreshold)
	sim.vitaminDeficiencyEnergyPenalty = nonNegative(config.VitaminDeficiencyEnergyPenalty, base.VitaminDeficiencyEnergyPenalty)

	sim.Grid = NewGridWorld(config.Width, config.Height, sim.maxAgentSlots)
	sim.AgentIDs = make([]int, sim.maxAgentSlots)
	for i := 0; i < sim.maxAgentSlots; i++ {
		sim.AgentIDs[i] = i
	}

	sim.alive = make([]bool, sim.maxAgentSlots)
	sim.energy = make([]float64, sim.maxAgentSlots)
	sim.inventoryTotal = make([]float64, sim.maxAgentSlots)
	sim.inventoryApple = make([]float64, sim.maxAgentSlots)
	sim.inventoryBanana = make([]float64, sim.maxAgentSlots)
	sim.inventoryOrange = make([]float64, sim.maxAgentSlots)
	sim.vitaminA = make([]float64, sim.maxAgentSlots)
	sim.vitaminB = make([]float64, sim.maxAgentSlots)
	sim.vitaminC = make([]float64, sim.maxAgentSlots)
	sim.bornTick = make([]int, sim.maxAgentSlots)
	sim.deathTick = make([]int, sim.maxAgentSlots)
	sim.actionLog = make([][]ActionType, sim.maxAgentSlots)
	for i := range sim.deathTick {
		sim.deathTick[i] = -1
	}

	sim.treeTypeByCell = make([]FruitType, cellCount)
	sim.groundFruitTypeByCell = make([]FruitType, cellCount)

	sim.seedTrees()
	sim.seedAgentState()
	if err := sim.seedAgents(); err != nil {
		return nil, err
	}

	sim.observationSize = len(EncodeObservation(sim.GetObservation(0)))
	sim.genomes = make([]AgentGenome, sim.maxAgentSlots)
	for i := 0; i < sim.maxAgentSlots; i++ {
		sim.genomes[i] = CreateRandomGenome(sim.rng, sim.observationSize, sim.genomeGenerationConfig)
	}

	return sim, nil
}

func (s *Simulation) Tick() int {
	return s.tickCount
}

func (s *Simulation) GetActionSpace() []string {
	out := make([]string, 0, len(AllActions()))
	for _, action := range AllActions() {
		out = append(out, action.String())
	}
	return out
}

func (s *Simulation) GetObservationVector(agentID int) []float64 {
	return EncodeObservation(s.GetObservation(agentID))
}

func (s *Simulation) GetValidActionIndices(agentID int) []int {
	valid := s.GetValidActions(agentID)
	out := make([]int, 0, len(valid))
	for _, action := range valid {
		out = append(out, int(action))
	}
	return out
}

func (s *Simulation) HasTreeAt(x, y int) bool {
	if !s.inBounds(x, y) {
		return false
	}
	return s.treeTypeByCell[s.indexXY(x, y)] != FruitNone
}

func (s *Simulation) GetTreeKindAt(x, y int) FruitType {
	if !s.inBounds(x, y) {
		return FruitNone
	}
	return s.treeTypeByCell[s.indexXY(x, y)]
}

func (s *Simulation) GetGroundFruitAt(x, y int) int {
	if !s.inBounds(x, y) {
		return 0
	}
	if s.groundFruitTypeByCell[s.indexXY(x, y)] == FruitNone {
		return 0
	}
	return 1
}

func (s *Simulation) GetGroundFruitKindAt(x, y int) FruitType {
	if !s.inBounds(x, y) {
		return FruitNone
	}
	return s.groundFruitTypeByCell[s.indexXY(x, y)]
}

func (s *Simulation) GetTreeIndices() []int {
	out := make([]int, len(s.treeIndices))
	copy(out, s.treeIndices)
	return out
}

func (s *Simulation) Step(actionProvider ActionProvider) TickStats {
	s.tickCount++
	s.dropFruitFromTrees()

	movedAgents := 0
	deathsThisTick := 0
	collectedFruit := 0
	eatenFruit := 0
	tradedFruit := 0
	actionHistogram := ActionHistogram{}

	s.rng.Shuffle(s.AgentIDs)
	for _, agentID := range s.AgentIDs {
		if !s.IsAgentAlive(agentID) {
			continue
		}
		if s.bornTick[agentID] == s.tickCount {
			continue
		}

		s.decayVitamins(agentID)

		observation := s.GetObservation(agentID)
		validActions := s.GetValidActions(agentID)
		chosenAction := ActionStay
		if actionProvider != nil {
			chosenAction = s.pickValidAction(actionProvider(agentID, observation, validActions), validActions)
		} else {
			chosenAction = s.selectActionFromGenome(agentID, observation, validActions)
		}

		actionHistogram.Inc(chosenAction)
		s.actionLog[agentID] = append(s.actionLog[agentID], chosenAction)

		effect := s.executeAction(agentID, chosenAction)
		if effect.Moved {
			movedAgents++
		}
		collectedFruit += effect.CollectedFruit
		eatenFruit += effect.EatenFruit
		tradedFruit += effect.TradedFruit

		deficiencyCount := s.countVitaminDeficiencies(agentID)
		cost := s.idleEnergyCost
		if effect.Moved {
			cost = s.moveEnergyCost
		}
		cost += float64(deficiencyCount)*s.vitaminDeficiencyEnergyPenalty + s.getNeuronEnergyCost(agentID)
		s.energy[agentID] = maxFloat(0, s.energy[agentID]-cost)

		if s.energy[agentID] <= 0 {
			s.markAgentDead(agentID)
			deathsThisTick++
		}
	}

	living := s.summarizeLivingAgents()
	ground := s.countGroundFruit()
	cellCount := s.Config.Width * s.Config.Height
	occupancy := 0.0
	if cellCount > 0 {
		occupancy = float64(living.AliveAgents) / float64(cellCount)
	}

	avgEnergy := 0.0
	avgInventory := 0.0
	avgVitaminA := 0.0
	avgVitaminB := 0.0
	avgVitaminC := 0.0
	if living.AliveAgents > 0 {
		den := float64(living.AliveAgents)
		avgEnergy = living.TotalEnergy / den
		avgInventory = living.TotalInventory / den
		avgVitaminA = living.TotalVitaminA / den
		avgVitaminB = living.TotalVitaminB / den
		avgVitaminC = living.TotalVitaminC / den
	}

	return TickStats{
		Tick:             s.tickCount,
		MovedAgents:      movedAgents,
		Occupancy:        occupancy,
		AliveAgents:      living.AliveAgents,
		DeathsThisTick:   deathsThisTick,
		AvgEnergy:        avgEnergy,
		AvgInventory:     avgInventory,
		CollectedFruit:   collectedFruit,
		EatenFruit:       eatenFruit,
		TradedFruit:      tradedFruit,
		GroundFruitTotal: ground.Apple + ground.Banana + ground.Orange,
		GroundApple:      ground.Apple,
		GroundBanana:     ground.Banana,
		GroundOrange:     ground.Orange,
		AvgVitaminA:      avgVitaminA,
		AvgVitaminB:      avgVitaminB,
		AvgVitaminC:      avgVitaminC,
		DeficientAgents:  living.DeficientAgents,
		ActionHistogram:  actionHistogram,
	}
}

func (s *Simulation) GetAgentPosition(agentID int) Coord {
	return s.Grid.GetPosition(agentID)
}

func (s *Simulation) GetAgentState(agentID int) AgentSnapshot {
	pos := s.Grid.GetPosition(agentID)
	return AgentSnapshot{
		ID:        agentID,
		X:         pos.X,
		Y:         pos.Y,
		Energy:    s.energy[agentID],
		Inventory: s.inventoryTotal[agentID],
		InventoryByFruit: FruitInventory{
			Apple:  s.inventoryApple[agentID],
			Banana: s.inventoryBanana[agentID],
			Orange: s.inventoryOrange[agentID],
		},
		Vitamins: VitaminLevels{
			VitaminA: s.vitaminA[agentID],
			VitaminB: s.vitaminB[agentID],
			VitaminC: s.vitaminC[agentID],
		},
		Alive: s.IsAgentAlive(agentID),
	}
}

func (s *Simulation) IsAgentAlive(agentID int) bool {
	return agentID >= 0 && agentID < len(s.alive) && s.alive[agentID]
}

func (s *Simulation) Snapshot() WorldSnapshot {
	world := WorldSnapshot{
		Width:                 s.Config.Width,
		Height:                s.Config.Height,
		TreeTypeByCell:        make([]int, len(s.treeTypeByCell)),
		GroundFruitTypeByCell: make([]int, len(s.groundFruitTypeByCell)),
		AgentX:                make([]int, len(s.AgentIDs)),
		AgentY:                make([]int, len(s.AgentIDs)),
		Alive:                 make([]bool, len(s.AgentIDs)),
		TreeIndices:           make([]int, len(s.treeIndices)),
	}
	for i, code := range s.treeTypeByCell {
		world.TreeTypeByCell[i] = int(code)
	}
	for i, code := range s.groundFruitTypeByCell {
		world.GroundFruitTypeByCell[i] = int(code)
	}
	for _, agentID := range s.AgentIDs {
		pos := s.Grid.GetPosition(agentID)
		world.AgentX[agentID] = pos.X
		world.AgentY[agentID] = pos.Y
		world.Alive[agentID] = s.IsAgentAlive(agentID)
	}
	copy(world.TreeIndices, s.treeIndices)
	return world
}

func (s *Simulation) GetAgentDetail(agentID int) (AgentDetail, error) {
	if agentID < 0 || agentID >= len(s.AgentIDs) {
		return AgentDetail{}, fmt.Errorf("agent out of range: %d", agentID)
	}

	state := s.GetAgentState(agentID)
	log := s.actionLog[agentID]
	counts := ActionHistogram{}
	for _, action := range log {
		counts.Inc(action)
	}

	recentCount := 20
	if recentCount > len(log) {
		recentCount = len(log)
	}
	recent := make([]string, 0, recentCount)
	for i := len(log) - recentCount; i < len(log); i++ {
		if i >= 0 {
			recent = append(recent, log[i].String())
		}
	}

	age := 0
	if state.Alive {
		age = s.tickCount - s.bornTick[agentID]
	} else {
		age = s.deathTick[agentID] - s.bornTick[agentID]
	}

	genome := s.genomes[agentID]
	return AgentDetail{
		ID:               state.ID,
		X:                state.X,
		Y:                state.Y,
		Energy:           state.Energy,
		Inventory:        state.Inventory,
		InventoryByFruit: state.InventoryByFruit,
		Vitamins:         state.Vitamins,
		Alive:            state.Alive,
		BornTick:         s.bornTick[agentID],
		DeathTick:        s.deathTick[agentID],
		Age:              age,
		TotalActions:     len(log),
		ActionCounts:     counts,
		RecentActions:    recent,
		NeuronCount:      len(genome.Neurons),
		NeuronEnergyCost: s.getNeuronEnergyCost(agentID),
		Genome: GenomeSummary{
			MutationRate:       genome.MutationRate,
			MutationScale:      genome.MutationScale,
			AddNeuronChance:    genome.AddNeuronChance,
			RemoveNeuronChance: genome.RemoveNeuronChance,
			FruitBiasApple:     genome.FruitBiasApple,
			FruitBiasBanana:    genome.FruitBiasBanana,
			FruitBiasOrange:    genome.FruitBiasOrange,
		},
	}, nil
}

func (s *Simulation) AgentAt(x, y int) int {
	if !s.inBounds(x, y) {
		return -1
	}
	return s.Grid.Occupancy[s.indexXY(x, y)]
}

func (s *Simulation) GetObservation(agentID int) Observation {
	self := s.GetAgentState(agentID)
	if !self.Alive {
		return Observation{
			Tick:      s.tickCount,
			Self:      self,
			Neighbors: NeighborObservation{North: -1, East: -1, South: -1, West: -1},
			Resources: ResourceObservation{},
			Social:    SocialObservation{AdjacentAgents: 0},
		}
	}

	pos := s.Grid.GetPosition(agentID)
	idx := s.index(pos)
	nFruit := s.fruitCodeAt(pos.X, pos.Y-1)
	eFruit := s.fruitCodeAt(pos.X+1, pos.Y)
	sFruit := s.fruitCodeAt(pos.X, pos.Y+1)
	wFruit := s.fruitCodeAt(pos.X-1, pos.Y)

	neighborFruitNorth := 0
	neighborFruitEast := 0
	neighborFruitSouth := 0
	neighborFruitWest := 0
	if nFruit != FruitNone {
		neighborFruitNorth = 1
	}
	if eFruit != FruitNone {
		neighborFruitEast = 1
	}
	if sFruit != FruitNone {
		neighborFruitSouth = 1
	}
	if wFruit != FruitNone {
		neighborFruitWest = 1
	}

	hasTree := 0
	groundFruit := 0
	if s.treeTypeByCell[idx] != FruitNone {
		hasTree = 1
	}
	if s.groundFruitTypeByCell[idx] != FruitNone {
		groundFruit = 1
	}

	return Observation{
		Tick: s.tickCount,
		Self: self,
		Neighbors: NeighborObservation{
			North: s.sense(pos.X, pos.Y-1),
			East:  s.sense(pos.X+1, pos.Y),
			South: s.sense(pos.X, pos.Y+1),
			West:  s.sense(pos.X-1, pos.Y),
		},
		Resources: ResourceObservation{
			HasTree:            hasTree,
			GroundFruit:        groundFruit,
			TreeType:           s.treeTypeByCell[idx],
			GroundFruitType:    s.groundFruitTypeByCell[idx],
			NeighborFruitNorth: neighborFruitNorth,
			NeighborFruitEast:  neighborFruitEast,
			NeighborFruitSouth: neighborFruitSouth,
			NeighborFruitWest:  neighborFruitWest,
		},
		Social: SocialObservation{
			AdjacentAgents: s.countAdjacentAliveAgents(pos),
		},
	}
}

func (s *Simulation) GetValidActions(agentID int) []ActionType {
	if !s.IsAgentAlive(agentID) {
		return []ActionType{ActionStay}
	}

	pos := s.Grid.GetPosition(agentID)
	idx := s.index(pos)
	valid := []ActionType{ActionStay}

	if s.sense(pos.X, pos.Y-1) == 0 {
		valid = append(valid, ActionMoveNorth)
	}
	if s.sense(pos.X+1, pos.Y) == 0 {
		valid = append(valid, ActionMoveEast)
	}
	if s.sense(pos.X, pos.Y+1) == 0 {
		valid = append(valid, ActionMoveSouth)
	}
	if s.sense(pos.X-1, pos.Y) == 0 {
		valid = append(valid, ActionMoveWest)
	}

	if s.groundFruitTypeByCell[idx] != FruitNone && s.inventoryTotal[agentID] < s.maxInventory {
		valid = append(valid, ActionCollectFruit)
	}
	if s.inventoryTotal[agentID] > 0 {
		valid = append(valid, ActionEatFruit)
	}
	if s.inventoryTotal[agentID] > 0 && s.countAdjacentAliveAgents(pos) > 0 {
		valid = append(valid, ActionTradeFruit)
	}
	if s.canCloneSelf(agentID, pos) {
		valid = append(valid, ActionCloneSelf)
	}

	return valid
}

func (s *Simulation) pickValidAction(chosen ActionType, validActions []ActionType) ActionType {
	for _, action := range validActions {
		if action == chosen {
			return chosen
		}
	}
	return ActionStay
}

func (s *Simulation) selectActionFromGenome(agentID int, observation Observation, validActions []ActionType) ActionType {
	validActionIndices := make([]int, 0, len(validActions))
	for _, action := range validActions {
		validActionIndices = append(validActionIndices, int(action))
	}
	featureVector := EncodeObservation(observation)
	chosenActionIndex := ChooseActionIndexFromGenome(s.genomes[agentID], featureVector, validActionIndices, s.rng)
	return s.pickValidAction(actionFromIndex(chosenActionIndex), validActions)
}

func (s *Simulation) executeAction(agentID int, action ActionType) actionEffect {
	effect := actionEffect{}
	if !s.IsAgentAlive(agentID) {
		return effect
	}

	current := s.Grid.GetPosition(agentID)
	idx := s.index(current)
	move := func(target Coord) {
		effect.Moved = s.Grid.Move(agentID, target)
	}

	switch action {
	case ActionStay:
		return effect
	case ActionMoveNorth:
		move(Coord{X: current.X, Y: current.Y - 1})
		return effect
	case ActionMoveEast:
		move(Coord{X: current.X + 1, Y: current.Y})
		return effect
	case ActionMoveSouth:
		move(Coord{X: current.X, Y: current.Y + 1})
		return effect
	case ActionMoveWest:
		move(Coord{X: current.X - 1, Y: current.Y})
		return effect
	case ActionCollectFruit:
		code := s.groundFruitTypeByCell[idx]
		if code == FruitNone || s.inventoryTotal[agentID] >= s.maxInventory {
			return effect
		}
		s.groundFruitTypeByCell[idx] = FruitNone
		s.addFruitToInventory(agentID, code, 1)
		effect.CollectedFruit = 1
		return effect
	case ActionEatFruit:
		code := s.pickFruitToEat(agentID)
		if code == FruitNone {
			return effect
		}
		s.removeFruitFromInventory(agentID, code, 1)
		s.energy[agentID] = minFloat(s.maxEnergy, s.energy[agentID]+s.fruitEnergyGain)
		s.gainVitaminFromFruit(agentID, code, 1)
		effect.EatenFruit = 1
		return effect
	case ActionTradeFruit:
		effect.TradedFruit = s.tradeFruitWithNeighbor(agentID, current)
		return effect
	case ActionCloneSelf:
		s.cloneSelf(agentID, current)
		return effect
	default:
		return effect
	}
}

func (s *Simulation) tradeFruitWithNeighbor(agentID int, from Coord) int {
	neighbors := s.Grid.Neighbors4(from)
	candidates := make([]int, 0, len(neighbors))
	for _, cell := range neighbors {
		other := s.Grid.Occupancy[s.index(cell)]
		if other >= 0 && s.IsAgentAlive(other) {
			candidates = append(candidates, other)
		}
	}
	if len(candidates) == 0 {
		return 0
	}

	receiverID := candidates[s.rng.Int(0, len(candidates))]
	code := s.pickFruitToTrade(agentID)
	if code == FruitNone {
		return 0
	}

	senderStock := s.getFruitInventory(agentID, code)
	receiverCapacity := s.maxInventory - s.inventoryTotal[receiverID]
	amount := minFloat(float64(s.tradeAmount), minFloat(senderStock, receiverCapacity))
	if amount <= 0 {
		return 0
	}

	s.removeFruitFromInventory(agentID, code, amount)
	s.addFruitToInventory(receiverID, code, amount)
	return int(amount)
}

func (s *Simulation) canCloneSelf(agentID int, from Coord) bool {
	if !s.IsAgentAlive(agentID) {
		return false
	}
	if len(s.deadAgentSlots) == 0 {
		return false
	}
	if s.energy[agentID] < s.cloneEnergyThreshold {
		return false
	}
	if s.energy[agentID] <= s.cloneEnergyCost+s.getNeuronEnergyCost(agentID) {
		return false
	}
	return len(s.getEmptyNeighborCells(from)) > 0
}

func (s *Simulation) cloneSelf(agentID int, from Coord) bool {
	if !s.canCloneSelf(agentID, from) {
		return false
	}
	if s.rng.Next() > s.cloneChancePerTick {
		return false
	}

	spawnCandidates := s.getEmptyNeighborCells(from)
	if len(spawnCandidates) == 0 {
		return false
	}
	childID := s.takeDeadAgentSlot()
	if childID < 0 {
		return false
	}

	spawn := spawnCandidates[s.rng.Int(0, len(spawnCandidates))]
	if !s.Grid.IsEmpty(spawn) {
		s.deadAgentSlots = append(s.deadAgentSlots, childID)
		return false
	}

	transferredEnergy := maxFloat(1, minFloat(s.cloneEnergyCost, s.energy[agentID]-1))
	s.energy[agentID] = maxFloat(1, s.energy[agentID]-transferredEnergy)

	s.alive[childID] = true
	s.energy[childID] = transferredEnergy
	s.inventoryTotal[childID] = 0
	s.inventoryApple[childID] = 0
	s.inventoryBanana[childID] = 0
	s.inventoryOrange[childID] = 0
	s.vitaminA[childID] = maxFloat(0, s.vitaminA[agentID]*0.8)
	s.vitaminB[childID] = maxFloat(0, s.vitaminB[agentID]*0.8)
	s.vitaminC[childID] = maxFloat(0, s.vitaminC[agentID]*0.8)
	s.bornTick[childID] = s.tickCount
	s.deathTick[childID] = -1
	s.actionLog[childID] = s.actionLog[childID][:0]
	s.genomes[childID] = CloneGenomeWithMutation(s.genomes[agentID], s.rng, s.observationSize, s.genomeGenerationConfig)
	if err := s.Grid.Place(childID, spawn); err != nil {
		s.alive[childID] = false
		s.deadAgentSlots = append(s.deadAgentSlots, childID)
		return false
	}
	return true
}

func (s *Simulation) getEmptyNeighborCells(from Coord) []Coord {
	neighbors := s.Grid.Neighbors4(from)
	out := make([]Coord, 0, len(neighbors))
	for _, cell := range neighbors {
		if s.Grid.Occupancy[s.index(cell)] == -1 {
			out = append(out, cell)
		}
	}
	return out
}

func (s *Simulation) takeDeadAgentSlot() int {
	if len(s.deadAgentSlots) == 0 {
		return -1
	}
	index := s.rng.Int(0, len(s.deadAgentSlots))
	slot := s.deadAgentSlots[index]
	s.deadAgentSlots = append(s.deadAgentSlots[:index], s.deadAgentSlots[index+1:]...)
	return slot
}

func (s *Simulation) markAgentDead(agentID int) {
	if !s.IsAgentAlive(agentID) {
		return
	}
	s.alive[agentID] = false
	s.deathTick[agentID] = s.tickCount
	s.Grid.Remove(agentID)
	s.deadAgentSlots = append(s.deadAgentSlots, agentID)
}

func (s *Simulation) getNeuronEnergyCost(agentID int) float64 {
	return float64(len(s.genomes[agentID].Neurons)) * s.neuronStepEnergyCost
}

func (s *Simulation) pickFruitToEat(agentID int) FruitType {
	if s.inventoryTotal[agentID] <= 0 {
		return FruitNone
	}

	candidates := []struct {
		Code  FruitType
		Score float64
	}{
		{
			Code:  FruitApple,
			Score: (s.vitaminNeedThreshold-s.vitaminA[agentID])*s.getFruitBias(agentID, FruitApple) + s.inventoryApple[agentID]*0.1,
		},
		{
			Code:  FruitBanana,
			Score: (s.vitaminNeedThreshold-s.vitaminB[agentID])*s.getFruitBias(agentID, FruitBanana) + s.inventoryBanana[agentID]*0.1,
		},
		{
			Code:  FruitOrange,
			Score: (s.vitaminNeedThreshold-s.vitaminC[agentID])*s.getFruitBias(agentID, FruitOrange) + s.inventoryOrange[agentID]*0.1,
		},
	}

	bestCode := FruitNone
	bestScore := math.Inf(-1)
	for _, c := range candidates {
		if s.getFruitInventory(agentID, c.Code) <= 0 {
			continue
		}
		if c.Score > bestScore {
			bestScore = c.Score
			bestCode = c.Code
		}
	}
	return bestCode
}

func (s *Simulation) pickFruitToTrade(agentID int) FruitType {
	apple := s.inventoryApple[agentID]
	banana := s.inventoryBanana[agentID]
	orange := s.inventoryOrange[agentID]
	if apple <= 0 && banana <= 0 && orange <= 0 {
		return FruitNone
	}

	options := []struct {
		Code     FruitType
		Pressure float64
	}{
		{Code: FruitApple, Pressure: apple / maxFloat(0.1, s.getFruitBias(agentID, FruitApple))},
		{Code: FruitBanana, Pressure: banana / maxFloat(0.1, s.getFruitBias(agentID, FruitBanana))},
		{Code: FruitOrange, Pressure: orange / maxFloat(0.1, s.getFruitBias(agentID, FruitOrange))},
	}

	bestCode := FruitNone
	bestPressure := math.Inf(-1)
	for _, option := range options {
		if s.getFruitInventory(agentID, option.Code) <= 0 {
			continue
		}
		if option.Pressure > bestPressure {
			bestPressure = option.Pressure
			bestCode = option.Code
		}
	}
	return bestCode
}

func (s *Simulation) addFruitToInventory(agentID int, code FruitType, amount float64) {
	if amount <= 0 {
		return
	}
	s.inventoryTotal[agentID] += amount
	switch code {
	case FruitApple:
		s.inventoryApple[agentID] += amount
	case FruitBanana:
		s.inventoryBanana[agentID] += amount
	case FruitOrange:
		s.inventoryOrange[agentID] += amount
	}
}

func (s *Simulation) removeFruitFromInventory(agentID int, code FruitType, amount float64) {
	if amount <= 0 {
		return
	}
	s.inventoryTotal[agentID] = maxFloat(0, s.inventoryTotal[agentID]-amount)
	switch code {
	case FruitApple:
		s.inventoryApple[agentID] = maxFloat(0, s.inventoryApple[agentID]-amount)
	case FruitBanana:
		s.inventoryBanana[agentID] = maxFloat(0, s.inventoryBanana[agentID]-amount)
	case FruitOrange:
		s.inventoryOrange[agentID] = maxFloat(0, s.inventoryOrange[agentID]-amount)
	}
}

func (s *Simulation) getFruitInventory(agentID int, code FruitType) float64 {
	switch code {
	case FruitApple:
		return s.inventoryApple[agentID]
	case FruitBanana:
		return s.inventoryBanana[agentID]
	case FruitOrange:
		return s.inventoryOrange[agentID]
	default:
		return 0
	}
}

func (s *Simulation) getFruitBias(agentID int, code FruitType) float64 {
	genome := s.genomes[agentID]
	switch code {
	case FruitApple:
		return genome.FruitBiasApple
	case FruitBanana:
		return genome.FruitBiasBanana
	case FruitOrange:
		return genome.FruitBiasOrange
	default:
		return 1
	}
}

func (s *Simulation) gainVitaminFromFruit(agentID int, code FruitType, amount float64) {
	switch code {
	case FruitApple:
		s.vitaminA[agentID] = minFloat(s.maxVitamin, s.vitaminA[agentID]+s.vitaminGainPerFruit*amount)
	case FruitBanana:
		s.vitaminB[agentID] = minFloat(s.maxVitamin, s.vitaminB[agentID]+s.vitaminGainPerFruit*amount)
	case FruitOrange:
		s.vitaminC[agentID] = minFloat(s.maxVitamin, s.vitaminC[agentID]+s.vitaminGainPerFruit*amount)
	}
}

func (s *Simulation) decayVitamins(agentID int) {
	s.vitaminA[agentID] = maxFloat(0, s.vitaminA[agentID]-s.vitaminDecayPerTick)
	s.vitaminB[agentID] = maxFloat(0, s.vitaminB[agentID]-s.vitaminDecayPerTick)
	s.vitaminC[agentID] = maxFloat(0, s.vitaminC[agentID]-s.vitaminDecayPerTick)
}

func (s *Simulation) countVitaminDeficiencies(agentID int) int {
	count := 0
	if s.vitaminA[agentID] < s.vitaminNeedThreshold {
		count++
	}
	if s.vitaminB[agentID] < s.vitaminNeedThreshold {
		count++
	}
	if s.vitaminC[agentID] < s.vitaminNeedThreshold {
		count++
	}
	return count
}

func (s *Simulation) countAdjacentAliveAgents(from Coord) int {
	count := 0
	for _, cell := range s.Grid.Neighbors4(from) {
		other := s.Grid.Occupancy[s.index(cell)]
		if other >= 0 && s.IsAgentAlive(other) {
			count++
		}
	}
	return count
}

func (s *Simulation) dropFruitFromTrees() {
	if s.fruitDropChancePerTree <= 0 || len(s.treeIndices) == 0 {
		return
	}

	for _, treeIdx := range s.treeIndices {
		if s.rng.Next() > s.fruitDropChancePerTree {
			continue
		}
		coord := s.coordFromIndex(treeIdx)
		slots := s.Grid.Neighbors4(coord)
		fruitAround := 0
		available := make([]Coord, 0, len(slots))
		for _, slot := range slots {
			idx := s.index(slot)
			if s.groundFruitTypeByCell[idx] != FruitNone {
				fruitAround++
			} else {
				available = append(available, slot)
			}
		}
		if fruitAround >= s.maxFruitAroundTree || len(available) == 0 {
			continue
		}
		target := available[s.rng.Int(0, len(available))]
		s.groundFruitTypeByCell[s.index(target)] = s.treeTypeByCell[treeIdx]
	}
}

func (s *Simulation) countGroundFruit() groundCounts {
	out := groundCounts{}
	for _, code := range s.groundFruitTypeByCell {
		switch code {
		case FruitApple:
			out.Apple++
		case FruitBanana:
			out.Banana++
		case FruitOrange:
			out.Orange++
		}
	}
	return out
}

func (s *Simulation) summarizeLivingAgents() livingSummary {
	out := livingSummary{}
	for _, agentID := range s.AgentIDs {
		if !s.IsAgentAlive(agentID) {
			continue
		}
		out.AliveAgents++
		out.TotalEnergy += s.energy[agentID]
		out.TotalInventory += s.inventoryTotal[agentID]
		out.TotalVitaminA += s.vitaminA[agentID]
		out.TotalVitaminB += s.vitaminB[agentID]
		out.TotalVitaminC += s.vitaminC[agentID]
		if s.countVitaminDeficiencies(agentID) > 0 {
			out.DeficientAgents++
		}
	}
	return out
}

func (s *Simulation) sense(x, y int) int {
	if !s.inBounds(x, y) {
		return -1
	}
	if s.Grid.Occupancy[s.indexXY(x, y)] == -1 {
		return 0
	}
	return 1
}

func (s *Simulation) seedTrees() {
	cells := s.Config.Width * s.Config.Height
	for idx := 0; idx < cells; idx++ {
		if s.rng.Next() < s.treeDensity {
			s.treeTypeByCell[idx] = s.randomFruitCode()
			s.treeIndices = append(s.treeIndices, idx)
		}
	}
	if s.treeDensity > 0 && len(s.treeIndices) == 0 {
		idx := s.rng.Int(0, cells)
		s.treeTypeByCell[idx] = s.randomFruitCode()
		s.treeIndices = append(s.treeIndices, idx)
	}
}

func (s *Simulation) seedAgentState() {
	s.deadAgentSlots = s.deadAgentSlots[:0]
	for _, agentID := range s.AgentIDs {
		active := agentID < s.initialAgentCount
		s.alive[agentID] = active
		if active {
			s.energy[agentID] = s.initialEnergy
			s.vitaminA[agentID] = s.initialVitamin
			s.vitaminB[agentID] = s.initialVitamin
			s.vitaminC[agentID] = s.initialVitamin
		} else {
			s.energy[agentID] = 0
			s.vitaminA[agentID] = 0
			s.vitaminB[agentID] = 0
			s.vitaminC[agentID] = 0
			s.deadAgentSlots = append(s.deadAgentSlots, agentID)
		}
		s.inventoryTotal[agentID] = 0
		s.inventoryApple[agentID] = 0
		s.inventoryBanana[agentID] = 0
		s.inventoryOrange[agentID] = 0
		s.bornTick[agentID] = 0
		s.deathTick[agentID] = -1
		s.actionLog[agentID] = s.actionLog[agentID][:0]
	}
}

func (s *Simulation) seedAgents() error {
	for agentID := 0; agentID < s.initialAgentCount; agentID++ {
		cell := s.randomEmptyCell()
		if err := s.Grid.Place(agentID, cell); err != nil {
			return err
		}
	}
	return nil
}

func (s *Simulation) randomEmptyCell() Coord {
	for {
		candidate := Coord{X: s.rng.Int(0, s.Config.Width), Y: s.rng.Int(0, s.Config.Height)}
		if s.Grid.IsEmpty(candidate) {
			return candidate
		}
	}
}

func (s *Simulation) randomFruitCode() FruitType {
	return FruitType(s.rng.Int(int(FruitApple), int(FruitOrange)+1))
}

func (s *Simulation) inBounds(x, y int) bool {
	return x >= 0 && x < s.Config.Width && y >= 0 && y < s.Config.Height
}

func (s *Simulation) fruitCodeAt(x, y int) FruitType {
	if !s.inBounds(x, y) {
		return FruitNone
	}
	return s.groundFruitTypeByCell[s.indexXY(x, y)]
}

func (s *Simulation) index(coord Coord) int {
	return s.indexXY(coord.X, coord.Y)
}

func (s *Simulation) indexXY(x, y int) int {
	return y*s.Config.Width + x
}

func (s *Simulation) coordFromIndex(index int) Coord {
	x := index % s.Config.Width
	y := index / s.Config.Width
	return Coord{X: x, Y: y}
}

func actionFromIndex(index int) ActionType {
	if index < 0 || index >= len(AllActions()) {
		return ActionStay
	}
	return ActionType(index)
}

func positive(value, fallback float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return fallback
	}
	return value
}

func nonNegative(value, fallback float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return fallback
	}
	return value
}

func ratio(value, fallback float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (s *Simulation) Validate() error {
	if s.Config.Width <= 0 || s.Config.Height <= 0 {
		return fmt.Errorf("invalid grid dimensions")
	}
	if s.initialAgentCount <= 0 {
		return fmt.Errorf("invalid initial agent count")
	}
	return nil
}
