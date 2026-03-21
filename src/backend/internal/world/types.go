package world

// --- Build types ---

type BuildKind string

const (
	BuildKindNone       BuildKind = "none"
	BuildKindCoalPlant  BuildKind = "coal_plant"
	BuildKindOilPlant   BuildKind = "oil_plant"
	BuildKindSolarFarm  BuildKind = "solar_farm"
	BuildKindWindFarm   BuildKind = "wind_farm"
	BuildKindComputeHub BuildKind = "compute_hub"
)

// --- Resource types ---

type Resources struct {
	Coal    float64 `json:"coal"`
	Oil     float64 `json:"oil"`
	Copper  float64 `json:"copper"`
	Silicon float64 `json:"silicon"`
}

type AssetConfig struct {
	CoalPlant float64 `json:"coalPlant"`
	OilPlant  float64 `json:"oilPlant"`
	SolarFarm float64 `json:"solarFarm"`
	WindFarm  float64 `json:"windFarm"`
}

// --- Country config ---

type CountryConfig struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Deposits      Resources   `json:"deposits"`
	Extractors    Resources   `json:"extractors"`
	Assets        AssetConfig `json:"assets"`
	SunPotential  float64     `json:"sunPotential"`
	WindPotential float64     `json:"windPotential"`
	BaseDemand    float64     `json:"baseDemand"`
	StartingMoney float64     `json:"startingMoney"`
}

// --- Config ---

type Config struct {
	Countries []CountryConfig `json:"countries"`

	// energy physics
	CoalEnergyPerUnit float64 `json:"coalEnergyPerUnit"`
	OilEnergyPerUnit  float64 `json:"oilEnergyPerUnit"`
	SolarUnitOutput   float64 `json:"solarUnitOutput"`
	WindUnitOutput    float64 `json:"windUnitOutput"`
	EnergyLossRate    float64 `json:"energyLossRate"`

	// build costs (resources only)
	EnergyBuildCopper   float64 `json:"energyBuildCopper"`
	EnergyBuildSilicon  float64 `json:"energyBuildSilicon"`
	PlantBuildGain      float64 `json:"plantBuildGain"`
	SolarBuildGain      float64 `json:"solarBuildGain"`
	WindBuildGain       float64 `json:"windBuildGain"`
	ComputeBuildCopper  float64 `json:"computeBuildCopper"`
	ComputeBuildSilicon float64 `json:"computeBuildSilicon"`
	ComputeGain         float64 `json:"computeGain"`
	MaxCompute          float64 `json:"maxCompute"`

	// survival
	StabilityPenalty  float64 `json:"stabilityPenalty"`
	StabilityRecovery float64 `json:"stabilityRecovery"`
}

// --- Agent actions (tool calls) ---

type ActionKind string

const (
	ActionPropose   ActionKind = "propose"   // direct barter proposal to another agent
	ActionAccept    ActionKind = "accept"    // accept a pending proposal
	ActionBuild     ActionKind = "build"     // build infrastructure
	ActionBroadcast ActionKind = "broadcast" // message to all
	ActionSend      ActionKind = "send"      // message to one
	ActionHold      ActionKind = "hold"      // do nothing
)

type AgentAction struct {
	Action ActionKind `json:"action"`

	// propose: direct trade
	To            string  `json:"to,omitempty"`            // target country ID
	OfferResource string  `json:"offerResource,omitempty"` // what you're giving (coal/oil/copper/silicon/money)
	OfferAmount   float64 `json:"offerAmount,omitempty"`
	WantResource  string  `json:"wantResource,omitempty"` // what you want back
	WantAmount    float64 `json:"wantAmount,omitempty"`

	// accept
	ProposalID int `json:"proposalId,omitempty"`

	// build
	Build BuildKind `json:"build,omitempty"`

	// broadcast / send
	Message string `json:"message,omitempty"`
}

// Proposal is a pending trade offer between two agents
type Proposal struct {
	ID            int     `json:"id"`
	From          string  `json:"from"`
	To            string  `json:"to"`
	OfferResource string  `json:"offerResource"`
	OfferAmount   float64 `json:"offerAmount"`
	WantResource  string  `json:"wantResource"`
	WantAmount    float64 `json:"wantAmount"`
	Tick          int     `json:"tick"` // when proposed
}

type AgentMsg struct {
	From    string `json:"from"`
	Message string `json:"message"`
}

type MemoryEntry struct {
	Tick     int           `json:"tick"`
	Actions  []AgentAction `json:"actions,omitempty"`
	Results  []string      `json:"results,omitempty"`
	Messages []AgentMsg    `json:"messages,omitempty"`
}

// --- LLM provider ---

type DecisionUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

type DecisionResult struct {
	Actions []AgentAction `json:"actions"`
	Source  string        `json:"source"`
	Model   string        `json:"model"`
	Summary string        `json:"summary"`
	Error   string        `json:"error"`
	Usage   DecisionUsage `json:"usage"`
	RawText string        `json:"rawText,omitempty"`
}

type CountryObservation struct {
	Tick      int             `json:"tick"`
	You       CountrySnapshot `json:"you"`
	Others    []PublicView    `json:"others"`
	Proposals []Proposal      `json:"proposals"` // pending proposals TO you
	Inbox     []AgentMsg      `json:"inbox"`
	Memory    []MemoryEntry   `json:"memory"`
}

type DecisionProvider func(observation CountryObservation) DecisionResult

type AgentCommand struct {
	From         string      `json:"from"`
	ObservedTick int         `json:"observedTick"`
	Action       AgentAction `json:"action"`
	Source       string      `json:"source"`
	Summary      string      `json:"summary"`
	Error        string      `json:"error"`
}

// --- Snapshots (for API/frontend) ---

type PublicView struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Alive     bool    `json:"alive"`
	Stability float64 `json:"stability"`
}

type CountrySnapshot struct {
	ID                  string      `json:"id"`
	Name                string      `json:"name"`
	Alive               bool        `json:"alive"`
	Stability           float64     `json:"stability"`
	Money               float64     `json:"money"`
	Deposits            Resources   `json:"deposits"`
	Stockpiles          Resources   `json:"stockpiles"`
	Assets              AssetConfig `json:"assets"`
	EnergyProduced      float64     `json:"energyProduced"`
	EnergyNeeded        float64     `json:"energyNeeded"`
	Shortage            float64     `json:"shortage"`
	LastBuild           BuildKind   `json:"lastBuild"`
	LastBuildOk         bool        `json:"lastBuildOk"`
	LastDecisionSource  string      `json:"lastDecisionSource"`
	LastDecisionSummary string      `json:"lastDecisionSummary"`
	LastDecisionError   string      `json:"lastDecisionError"`
	TradesCompleted     int         `json:"tradesCompleted"`
}

type TradeRecord struct {
	Tick          int     `json:"tick"`
	From          string  `json:"from"`
	To            string  `json:"to"`
	OfferResource string  `json:"offerResource"`
	OfferAmount   float64 `json:"offerAmount"`
	WantResource  string  `json:"wantResource"`
	WantAmount    float64 `json:"wantAmount"`
}

type WorldSnapshot struct {
	Tick      int               `json:"tick"`
	Countries []CountrySnapshot `json:"countries"`
	Trades    []TradeRecord     `json:"trades"`    // trades executed this tick
	Proposals []Proposal        `json:"proposals"` // pending proposals
	Messages  []AgentMsg        `json:"messages"`  // broadcasts this tick
}

type TickStats struct {
	Tick            int     `json:"tick"`
	LivingCountries int     `json:"livingCountries"`
	TotalTrades     int     `json:"totalTrades"`
	TotalShortage   float64 `json:"totalShortage"`
	TotalMoney      float64 `json:"totalMoney"`
}

// --- Default config ---

func DefaultConfig() Config {
	return Config{
		Countries: []CountryConfig{
			{
				ID:           "coalreach",
				Name:         "Coalreach",
				Deposits:     Resources{Coal: 800, Oil: 60, Copper: 200, Silicon: 40},
				Extractors:   Resources{Coal: 8, Oil: 0.5, Copper: 1.5, Silicon: 0.3},
				Assets:       AssetConfig{CoalPlant: 8, OilPlant: 1, SolarFarm: 1, WindFarm: 0.5},
				SunPotential: 0.5, WindPotential: 0.4,
				BaseDemand:    20,
				StartingMoney: 100,
			},
			{
				ID:           "petroport",
				Name:         "Petroport",
				Deposits:     Resources{Coal: 150, Oil: 600, Copper: 120, Silicon: 50},
				Extractors:   Resources{Coal: 1.5, Oil: 7, Copper: 1, Silicon: 0.4},
				Assets:       AssetConfig{CoalPlant: 2, OilPlant: 7, SolarFarm: 0.8, WindFarm: 0.5},
				SunPotential: 0.6, WindPotential: 0.45,
				BaseDemand:    20,
				StartingMoney: 100,
			},
			{
				ID:           "silica",
				Name:         "Silica",
				Deposits:     Resources{Coal: 100, Oil: 60, Copper: 100, Silicon: 700},
				Extractors:   Resources{Coal: 1, Oil: 0.5, Copper: 0.8, Silicon: 6},
				Assets:       AssetConfig{CoalPlant: 1.5, OilPlant: 1, SolarFarm: 4, WindFarm: 1},
				SunPotential: 1.0, WindPotential: 0.5,
				BaseDemand:    18,
				StartingMoney: 100,
			},
			{
				ID:           "aurora",
				Name:         "Aurora",
				Deposits:     Resources{Coal: 80, Oil: 50, Copper: 300, Silicon: 80},
				Extractors:   Resources{Coal: 0.8, Oil: 0.5, Copper: 2.5, Silicon: 0.7},
				Assets:       AssetConfig{CoalPlant: 1.5, OilPlant: 1, SolarFarm: 3, WindFarm: 5},
				SunPotential: 0.9, WindPotential: 1.2,
				BaseDemand:    18,
				StartingMoney: 100,
			},
		},

		CoalEnergyPerUnit: 2.5,
		OilEnergyPerUnit:  2.8,
		SolarUnitOutput:   2.0,
		WindUnitOutput:    2.0,
		EnergyLossRate:    0.1,

		EnergyBuildCopper:   5,
		EnergyBuildSilicon:  2,
		PlantBuildGain:      3.0,
		SolarBuildGain:      2.5,
		WindBuildGain:       2.5,
		ComputeBuildCopper:  4,
		ComputeBuildSilicon: 6,
		ComputeGain:         0.1,
		MaxCompute:          1.0,

		StabilityPenalty:  0.5,
		StabilityRecovery: 0.3,
	}
}
