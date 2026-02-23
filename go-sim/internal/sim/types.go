package sim

import "fmt"

type Coord struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type ActionType uint8

const (
	ActionStay ActionType = iota
	ActionMoveNorth
	ActionMoveEast
	ActionMoveSouth
	ActionMoveWest
	ActionCollectFruit
	ActionEatFruit
	ActionTradeFruit
	ActionCloneSelf
)

var actionNames = [...]string{
	"stay",
	"move_north",
	"move_east",
	"move_south",
	"move_west",
	"collect_fruit",
	"eat_fruit",
	"trade_fruit",
	"clone_self",
}

func (a ActionType) String() string {
	idx := int(a)
	if idx < 0 || idx >= len(actionNames) {
		return "stay"
	}
	return actionNames[idx]
}

func ParseAction(name string) (ActionType, error) {
	for i, v := range actionNames {
		if v == name {
			return ActionType(i), nil
		}
	}
	return ActionStay, fmt.Errorf("unknown action: %s", name)
}

func AllActions() []ActionType {
	return []ActionType{
		ActionStay,
		ActionMoveNorth,
		ActionMoveEast,
		ActionMoveSouth,
		ActionMoveWest,
		ActionCollectFruit,
		ActionEatFruit,
		ActionTradeFruit,
		ActionCloneSelf,
	}
}

type ActionHistogram struct {
	Stay         int `json:"stay"`
	MoveNorth    int `json:"move_north"`
	MoveEast     int `json:"move_east"`
	MoveSouth    int `json:"move_south"`
	MoveWest     int `json:"move_west"`
	CollectFruit int `json:"collect_fruit"`
	EatFruit     int `json:"eat_fruit"`
	TradeFruit   int `json:"trade_fruit"`
	CloneSelf    int `json:"clone_self"`
}

func (h *ActionHistogram) Inc(action ActionType) {
	switch action {
	case ActionStay:
		h.Stay++
	case ActionMoveNorth:
		h.MoveNorth++
	case ActionMoveEast:
		h.MoveEast++
	case ActionMoveSouth:
		h.MoveSouth++
	case ActionMoveWest:
		h.MoveWest++
	case ActionCollectFruit:
		h.CollectFruit++
	case ActionEatFruit:
		h.EatFruit++
	case ActionTradeFruit:
		h.TradeFruit++
	case ActionCloneSelf:
		h.CloneSelf++
	}
}

type FruitType uint8

const (
	FruitNone FruitType = iota
	FruitApple
	FruitBanana
	FruitOrange
)

var fruitNames = [...]string{"none", "apple", "banana", "orange"}

func (f FruitType) String() string {
	idx := int(f)
	if idx < 0 || idx >= len(fruitNames) {
		return "none"
	}
	return fruitNames[idx]
}

type FruitInventory struct {
	Apple  float64 `json:"apple"`
	Banana float64 `json:"banana"`
	Orange float64 `json:"orange"`
}

type VitaminLevels struct {
	VitaminA float64 `json:"vitamin_a"`
	VitaminB float64 `json:"vitamin_b"`
	VitaminC float64 `json:"vitamin_c"`
}

type AgentSnapshot struct {
	ID               int            `json:"id"`
	X                int            `json:"x"`
	Y                int            `json:"y"`
	Energy           float64        `json:"energy"`
	Inventory        float64        `json:"inventory"`
	InventoryByFruit FruitInventory `json:"inventoryByFruit"`
	Vitamins         VitaminLevels  `json:"vitamins"`
	Alive            bool           `json:"alive"`
}

type NeighborObservation struct {
	North int `json:"north"`
	East  int `json:"east"`
	South int `json:"south"`
	West  int `json:"west"`
}

type ResourceObservation struct {
	HasTree            int       `json:"hasTree"`
	GroundFruit        int       `json:"groundFruit"`
	TreeType           FruitType `json:"treeType"`
	GroundFruitType    FruitType `json:"groundFruitType"`
	NeighborFruitNorth int       `json:"neighborFruitNorth"`
	NeighborFruitEast  int       `json:"neighborFruitEast"`
	NeighborFruitSouth int       `json:"neighborFruitSouth"`
	NeighborFruitWest  int       `json:"neighborFruitWest"`
}

type SocialObservation struct {
	AdjacentAgents int `json:"adjacentAgents"`
}

type Observation struct {
	Tick      int                 `json:"tick"`
	Self      AgentSnapshot       `json:"self"`
	Neighbors NeighborObservation `json:"neighbors"`
	Resources ResourceObservation `json:"resources"`
	Social    SocialObservation   `json:"social"`
}

type TickStats struct {
	Tick             int             `json:"tick"`
	MovedAgents      int             `json:"movedAgents"`
	Occupancy        float64         `json:"occupancy"`
	AliveAgents      int             `json:"aliveAgents"`
	DeathsThisTick   int             `json:"deathsThisTick"`
	AvgEnergy        float64         `json:"avgEnergy"`
	AvgInventory     float64         `json:"avgInventory"`
	CollectedFruit   int             `json:"collectedFruit"`
	EatenFruit       int             `json:"eatenFruit"`
	TradedFruit      int             `json:"tradedFruit"`
	GroundFruitTotal int             `json:"groundFruitTotal"`
	GroundApple      int             `json:"groundApple"`
	GroundBanana     int             `json:"groundBanana"`
	GroundOrange     int             `json:"groundOrange"`
	AvgVitaminA      float64         `json:"avgVitaminA"`
	AvgVitaminB      float64         `json:"avgVitaminB"`
	AvgVitaminC      float64         `json:"avgVitaminC"`
	DeficientAgents  int             `json:"deficientAgents"`
	ActionHistogram  ActionHistogram `json:"actionHistogram"`
}

type GenomeSummary struct {
	MutationRate       float64 `json:"mutationRate"`
	MutationScale      float64 `json:"mutationScale"`
	AddNeuronChance    float64 `json:"addNeuronChance"`
	RemoveNeuronChance float64 `json:"removeNeuronChance"`
	FruitBiasApple     float64 `json:"fruitBiasApple"`
	FruitBiasBanana    float64 `json:"fruitBiasBanana"`
	FruitBiasOrange    float64 `json:"fruitBiasOrange"`
}

type AgentDetail struct {
	ID               int             `json:"id"`
	X                int             `json:"x"`
	Y                int             `json:"y"`
	Energy           float64         `json:"energy"`
	Inventory        float64         `json:"inventory"`
	InventoryByFruit FruitInventory  `json:"inventoryByFruit"`
	Vitamins         VitaminLevels   `json:"vitamins"`
	Alive            bool            `json:"alive"`
	BornTick         int             `json:"bornTick"`
	DeathTick        int             `json:"deathTick"`
	Age              int             `json:"age"`
	TotalActions     int             `json:"totalActions"`
	ActionCounts     ActionHistogram `json:"actionCounts"`
	RecentActions    []string        `json:"recentActions"`
	NeuronCount      int             `json:"neuronCount"`
	NeuronEnergyCost float64         `json:"neuronEnergyCost"`
	Genome           GenomeSummary   `json:"genome"`
}

type WorldSnapshot struct {
	Width                 int     `json:"width"`
	Height                int     `json:"height"`
	TreeTypeByCell        []uint8 `json:"treeTypeByCell"`
	GroundFruitTypeByCell []uint8 `json:"groundFruitTypeByCell"`
	AgentX                []int   `json:"agentX"`
	AgentY                []int   `json:"agentY"`
	Alive                 []bool  `json:"alive"`
	TreeIndices           []int   `json:"treeIndices"`
}

type SimulationConfig struct {
	Width                          int     `json:"width"`
	Height                         int     `json:"height"`
	AgentCount                     int     `json:"agentCount"`
	MaxAgentSlots                  int     `json:"maxAgentSlots,omitempty"`
	InitialEnergy                  float64 `json:"initialEnergy,omitempty"`
	MaxEnergy                      float64 `json:"maxEnergy,omitempty"`
	MoveEnergyCost                 float64 `json:"moveEnergyCost,omitempty"`
	IdleEnergyCost                 float64 `json:"idleEnergyCost,omitempty"`
	NeuronStepEnergyCost           float64 `json:"neuronStepEnergyCost,omitempty"`
	MinNeuronCount                 int     `json:"minNeuronCount,omitempty"`
	MaxNeuronCount                 int     `json:"maxNeuronCount,omitempty"`
	MutationRateBase               float64 `json:"mutationRateBase,omitempty"`
	MutationScaleBase              float64 `json:"mutationScaleBase,omitempty"`
	AddNeuronChanceBase            float64 `json:"addNeuronChanceBase,omitempty"`
	RemoveNeuronChanceBase         float64 `json:"removeNeuronChanceBase,omitempty"`
	CloneEnergyThreshold           float64 `json:"cloneEnergyThreshold,omitempty"`
	CloneEnergyCost                float64 `json:"cloneEnergyCost,omitempty"`
	CloneChancePerTick             float64 `json:"cloneChancePerTick,omitempty"`
	TreeDensity                    float64 `json:"treeDensity,omitempty"`
	FruitDropPerTree               float64 `json:"fruitDropPerTree,omitempty"`
	MaxFruitAroundTree             int     `json:"maxFruitAroundTree,omitempty"`
	MaxInventory                   float64 `json:"maxInventory,omitempty"`
	FruitEnergyGain                float64 `json:"fruitEnergyGain,omitempty"`
	TradeAmount                    int     `json:"tradeAmount,omitempty"`
	InitialVitamin                 float64 `json:"initialVitamin,omitempty"`
	MaxVitamin                     float64 `json:"maxVitamin,omitempty"`
	VitaminDecayPerTick            float64 `json:"vitaminDecayPerTick,omitempty"`
	VitaminGainPerFruit            float64 `json:"vitaminGainPerFruit,omitempty"`
	VitaminNeedThreshold           float64 `json:"vitaminNeedThreshold,omitempty"`
	VitaminDeficiencyEnergyPenalty float64 `json:"vitaminDeficiencyEnergyPenalty,omitempty"`
}

type ActionProvider func(agentID int, observation Observation, validActions []ActionType) ActionType

func EncodeObservation(obs Observation) []float64 {
	alive := 0.0
	if obs.Self.Alive {
		alive = 1
	}
	return []float64{
		float64(obs.Tick),
		float64(obs.Self.X),
		float64(obs.Self.Y),
		obs.Self.Energy,
		obs.Self.Inventory,
		obs.Self.InventoryByFruit.Apple,
		obs.Self.InventoryByFruit.Banana,
		obs.Self.InventoryByFruit.Orange,
		obs.Self.Vitamins.VitaminA,
		obs.Self.Vitamins.VitaminB,
		obs.Self.Vitamins.VitaminC,
		alive,
		float64(obs.Neighbors.North),
		float64(obs.Neighbors.East),
		float64(obs.Neighbors.South),
		float64(obs.Neighbors.West),
		float64(obs.Resources.HasTree),
		float64(obs.Resources.GroundFruit),
		float64(obs.Resources.TreeType),
		float64(obs.Resources.GroundFruitType),
		float64(obs.Resources.NeighborFruitNorth),
		float64(obs.Resources.NeighborFruitEast),
		float64(obs.Resources.NeighborFruitSouth),
		float64(obs.Resources.NeighborFruitWest),
		float64(obs.Social.AdjacentAgents),
	}
}

func DefaultConfig() SimulationConfig {
	return SimulationConfig{
		Width:                          80,
		Height:                         50,
		AgentCount:                     240,
		MaxAgentSlots:                  420,
		InitialEnergy:                  80,
		MaxEnergy:                      220,
		MoveEnergyCost:                 1,
		IdleEnergyCost:                 0.25,
		NeuronStepEnergyCost:           0.05,
		MinNeuronCount:                 4,
		MaxNeuronCount:                 14,
		MutationRateBase:               0.08,
		MutationScaleBase:              0.20,
		AddNeuronChanceBase:            0.06,
		RemoveNeuronChanceBase:         0.04,
		CloneEnergyThreshold:           145,
		CloneEnergyCost:                56,
		CloneChancePerTick:             1,
		TreeDensity:                    0.08,
		FruitDropPerTree:               0.35,
		MaxFruitAroundTree:             4,
		MaxInventory:                   50,
		FruitEnergyGain:                6,
		TradeAmount:                    1,
		InitialVitamin:                 60,
		MaxVitamin:                     100,
		VitaminDecayPerTick:            0.45,
		VitaminGainPerFruit:            26,
		VitaminNeedThreshold:           25,
		VitaminDeficiencyEnergyPenalty: 0.6,
	}
}
