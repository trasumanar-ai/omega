package econ

type BuildFocus string

const (
	BuildHold           BuildFocus = "hold"
	BuildEnergy         BuildFocus = "energy"
	BuildCompute        BuildFocus = "compute"
	BuildInfrastructure BuildFocus = "infrastructure"
)

type BuildKind string

const (
	BuildKindNone        BuildKind = "none"
	BuildKindCoalPlant   BuildKind = "coal_plant"
	BuildKindOilPlant    BuildKind = "oil_plant"
	BuildKindSolarFarm   BuildKind = "solar_farm"
	BuildKindWindFarm    BuildKind = "wind_farm"
	BuildKindComputeHub  BuildKind = "compute_hub"
	BuildKindGridUpgrade BuildKind = "grid_upgrade"
)

type MarketAlgorithm string

const (
	MarketAlgorithmNoTrade       MarketAlgorithm = "no_trade"
	MarketAlgorithmLinear        MarketAlgorithm = "linear"
	MarketAlgorithmScarcitySpike MarketAlgorithm = "scarcity_spike"
)

type ResourceStockpile struct {
	Coal    float64 `json:"coal"`
	Oil     float64 `json:"oil"`
	Copper  float64 `json:"copper"`
	Silicon float64 `json:"silicon"`
}

type DepositSnapshot struct {
	Coal    float64 `json:"coal"`
	Oil     float64 `json:"oil"`
	Copper  float64 `json:"copper"`
	Silicon float64 `json:"silicon"`
}

type ExtractorConfig struct {
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

type CountryConfig struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	X                 float64         `json:"x"`
	Y                 float64         `json:"y"`
	Deposits          DepositSnapshot `json:"deposits"`
	Extractors        ExtractorConfig `json:"extractors"`
	Assets            AssetConfig     `json:"assets"`
	SunPotential      float64         `json:"sunPotential"`
	WindPotential     float64         `json:"windPotential"`
	BaseDemand        float64         `json:"baseDemand"`
	StartingTreasury  float64         `json:"startingTreasury"`
	StartingReserve   float64         `json:"startingReserve"`
	Infrastructure    float64         `json:"infrastructure"`
	ComputeEfficiency float64         `json:"computeEfficiency"`
}

type RouteConfig struct {
	A        string  `json:"a"`
	B        string  `json:"b"`
	Capacity float64 `json:"capacity"`
}

type Config struct {
	Countries       []CountryConfig `json:"countries"`
	Routes          []RouteConfig   `json:"routes"`
	MarketAlgorithm MarketAlgorithm `json:"marketAlgorithm"`

	CoalEnergyPerUnit float64 `json:"coalEnergyPerUnit"`
	OilEnergyPerUnit  float64 `json:"oilEnergyPerUnit"`
	SolarUnitOutput   float64 `json:"solarUnitOutput"`
	WindUnitOutput    float64 `json:"windUnitOutput"`

	ComputeEfficiencyDividend float64 `json:"computeEfficiencyDividend"`
	LocalLossRate             float64 `json:"localLossRate"`
	TradeLossRate             float64 `json:"tradeLossRate"`
	ComputeLossDividend       float64 `json:"computeLossDividend"`
	EnergyReserveBase         float64 `json:"energyReserveBase"`
	EnergyReservePerInfra     float64 `json:"energyReservePerInfra"`
	GridCapacityBase          float64 `json:"gridCapacityBase"`
	GridCapacityPerInfra      float64 `json:"gridCapacityPerInfra"`
	RouteInfraScale           float64 `json:"routeInfraScale"`

	ComputeUpkeepBase  float64 `json:"computeUpkeepBase"`
	ComputeUpkeepScale float64 `json:"computeUpkeepScale"`
	InfraUpkeepScale   float64 `json:"infraUpkeepScale"`

	EnergySalePrice        float64 `json:"energySalePrice"`
	EmergencyEnergyPrice   float64 `json:"emergencyEnergyPrice"`
	EmergencyEnergyPerTick float64 `json:"emergencyEnergyPerTick"`
	CoalPrice              float64 `json:"coalPrice"`
	OilPrice               float64 `json:"oilPrice"`
	CopperPrice            float64 `json:"copperPrice"`
	SiliconPrice           float64 `json:"siliconPrice"`

	IndustrialOutputPrice float64 `json:"industrialOutputPrice"`
	IndustrialOutputScale float64 `json:"industrialOutputScale"`
	StabilityPenalty      float64 `json:"stabilityPenalty"`
	StabilityRecovery     float64 `json:"stabilityRecovery"`

	EnergyBuildTreasury float64 `json:"energyBuildTreasury"`
	EnergyBuildCopper   float64 `json:"energyBuildCopper"`
	EnergyBuildSilicon  float64 `json:"energyBuildSilicon"`
	CoalPlantBuildGain  float64 `json:"coalPlantBuildGain"`
	OilPlantBuildGain   float64 `json:"oilPlantBuildGain"`
	SolarBuildGain      float64 `json:"solarBuildGain"`
	WindBuildGain       float64 `json:"windBuildGain"`

	ComputeBuildTreasury   float64 `json:"computeBuildTreasury"`
	ComputeBuildCopper     float64 `json:"computeBuildCopper"`
	ComputeBuildSilicon    float64 `json:"computeBuildSilicon"`
	ComputeEfficiencyGain  float64 `json:"computeEfficiencyGain"`
	MaxComputeEfficiency   float64 `json:"maxComputeEfficiency"`
	InfrastructureTreasury float64 `json:"infrastructureTreasury"`
	InfrastructureCopper   float64 `json:"infrastructureCopper"`
	InfrastructureGain     float64 `json:"infrastructureGain"`

	TradeEnergyShare float64 `json:"tradeEnergyShare"`
	TradeCargoShare  float64 `json:"tradeCargoShare"`
}

type TickStats struct {
	Tick                 int     `json:"tick"`
	LivingCountries      int     `json:"livingCountries"`
	TotalEnergyGenerated float64 `json:"totalEnergyGenerated"`
	TotalEnergyDelivered float64 `json:"totalEnergyDelivered"`
	TotalEnergyTraded    float64 `json:"totalEnergyTraded"`
	TotalCargoTraded     float64 `json:"totalCargoTraded"`
	TotalShortage        float64 `json:"totalShortage"`
	TotalTreasury        float64 `json:"totalTreasury"`
	AvgComputeEfficiency float64 `json:"avgComputeEfficiency"`
	EnergyBuilds         int     `json:"energyBuilds"`
	ComputeBuilds        int     `json:"computeBuilds"`
	InfraBuilds          int     `json:"infraBuilds"`
	CoalRemaining        float64 `json:"coalRemaining"`
	OilRemaining         float64 `json:"oilRemaining"`
	CopperRemaining      float64 `json:"copperRemaining"`
	SiliconRemaining     float64 `json:"siliconRemaining"`
}

type CountryDecision struct {
	BuildFocus BuildFocus `json:"buildFocus"`
}

type DecisionUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

type DecisionResult struct {
	Decision CountryDecision `json:"decision"`
	Source   string          `json:"source"`
	Model    string          `json:"model"`
	Summary  string          `json:"summary"`
	Error    string          `json:"error"`
	Usage    DecisionUsage   `json:"usage"`
}

type CountryObservation struct {
	Tick        int             `json:"tick"`
	Country     CountrySnapshot `json:"country"`
	Neighbors   []NeighborView  `json:"neighbors"`
	WorldTotals TickStats       `json:"worldTotals"`
}

type DecisionProvider func(observation CountryObservation) DecisionResult

type NeighborView struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Stability         float64 `json:"stability"`
	EnergyReserve     float64 `json:"energyReserve"`
	ComputeEfficiency float64 `json:"computeEfficiency"`
	Infrastructure    float64 `json:"infrastructure"`
}

type CountryAssetsSnapshot struct {
	CoalPlant float64 `json:"coalPlant"`
	OilPlant  float64 `json:"oilPlant"`
	SolarFarm float64 `json:"solarFarm"`
	WindFarm  float64 `json:"windFarm"`
}

type ResourceQuoteSnapshot struct {
	Resource   string  `json:"resource"`
	Net        float64 `json:"net"`
	LimitPrice float64 `json:"limitPrice"`
}

type CountrySnapshot struct {
	ID                    string                  `json:"id"`
	Name                  string                  `json:"name"`
	X                     float64                 `json:"x"`
	Y                     float64                 `json:"y"`
	Alive                 bool                    `json:"alive"`
	Stability             float64                 `json:"stability"`
	Treasury              float64                 `json:"treasury"`
	EnergyReserve         float64                 `json:"energyReserve"`
	ReserveCapacity       float64                 `json:"reserveCapacity"`
	GridCapacity          float64                 `json:"gridCapacity"`
	Infrastructure        float64                 `json:"infrastructure"`
	ComputeEfficiency     float64                 `json:"computeEfficiency"`
	BaseDemand            float64                 `json:"baseDemand"`
	Deposits              DepositSnapshot         `json:"deposits"`
	Stockpiles            ResourceStockpile       `json:"stockpiles"`
	Assets                CountryAssetsSnapshot   `json:"assets"`
	LastDemand            float64                 `json:"lastDemand"`
	LastGeneratedEnergy   float64                 `json:"lastGeneratedEnergy"`
	LastDeliveredEnergy   float64                 `json:"lastDeliveredEnergy"`
	LastImportedEnergy    float64                 `json:"lastImportedEnergy"`
	LastExportedEnergy    float64                 `json:"lastExportedEnergy"`
	LastShortage          float64                 `json:"lastShortage"`
	LastTreasuryDelta     float64                 `json:"lastTreasuryDelta"`
	LastTradeRevenue      float64                 `json:"lastTradeRevenue"`
	LastTradeSpend        float64                 `json:"lastTradeSpend"`
	LastIndustrialRevenue float64                 `json:"lastIndustrialRevenue"`
	LastEmergencySpend    float64                 `json:"lastEmergencySpend"`
	LastBuildSpend        float64                 `json:"lastBuildSpend"`
	LastBuildFocus        BuildFocus              `json:"lastBuildFocus"`
	LastBuildKind         BuildKind               `json:"lastBuildKind"`
	LastBuildSuccess      bool                    `json:"lastBuildSuccess"`
	LastBuildFailure      string                  `json:"lastBuildFailure"`
	LastDecisionSource    string                  `json:"lastDecisionSource"`
	LastDecisionModel     string                  `json:"lastDecisionModel"`
	LastDecisionSummary   string                  `json:"lastDecisionSummary"`
	LastDecisionError     string                  `json:"lastDecisionError"`
	LastDecisionUsage     DecisionUsage           `json:"lastDecisionUsage"`
	MarketQuotes          []ResourceQuoteSnapshot `json:"marketQuotes"`
}

type RouteSnapshot struct {
	A            string  `json:"a"`
	B            string  `json:"b"`
	Capacity     float64 `json:"capacity"`
	LastEnergyAB float64 `json:"lastEnergyAB"`
	LastEnergyBA float64 `json:"lastEnergyBA"`
	LastCargoAB  float64 `json:"lastCargoAB"`
	LastCargoBA  float64 `json:"lastCargoBA"`
	EffectiveCap float64 `json:"effectiveCap"`
}

type WorldSnapshot struct {
	Tick      int               `json:"tick"`
	Countries []CountrySnapshot `json:"countries"`
	Routes    []RouteSnapshot   `json:"routes"`
}

type MarketCountrySnapshot struct {
	ID                string                  `json:"id"`
	Name              string                  `json:"name"`
	Treasury          float64                 `json:"treasury"`
	TreasuryDelta     float64                 `json:"treasuryDelta"`
	EnergyReserve     float64                 `json:"energyReserve"`
	Demand            float64                 `json:"demand"`
	GeneratedEnergy   float64                 `json:"generatedEnergy"`
	DeliveredEnergy   float64                 `json:"deliveredEnergy"`
	ImportedEnergy    float64                 `json:"importedEnergy"`
	ExportedEnergy    float64                 `json:"exportedEnergy"`
	Shortage          float64                 `json:"shortage"`
	TradeRevenue      float64                 `json:"tradeRevenue"`
	TradeSpend        float64                 `json:"tradeSpend"`
	IndustrialRevenue float64                 `json:"industrialRevenue"`
	EmergencySpend    float64                 `json:"emergencySpend"`
	BuildSpend        float64                 `json:"buildSpend"`
	LastBuildFocus    BuildFocus              `json:"lastBuildFocus"`
	LastBuildKind     BuildKind               `json:"lastBuildKind"`
	Quotes            []ResourceQuoteSnapshot `json:"quotes"`
}

type MarketSnapshot struct {
	EnergyPrice            float64                 `json:"energyPrice"`
	EmergencyEnergyPrice   float64                 `json:"emergencyEnergyPrice"`
	CoalPrice              float64                 `json:"coalPrice"`
	OilPrice               float64                 `json:"oilPrice"`
	CopperPrice            float64                 `json:"copperPrice"`
	SiliconPrice           float64                 `json:"siliconPrice"`
	EnergyBuildTreasury    float64                 `json:"energyBuildTreasury"`
	ComputeBuildTreasury   float64                 `json:"computeBuildTreasury"`
	InfrastructureTreasury float64                 `json:"infrastructureTreasury"`
	TotalDemand            float64                 `json:"totalDemand"`
	TotalGenerated         float64                 `json:"totalGenerated"`
	TotalDelivered         float64                 `json:"totalDelivered"`
	TotalImported          float64                 `json:"totalImported"`
	TotalExported          float64                 `json:"totalExported"`
	TotalShortage          float64                 `json:"totalShortage"`
	TotalTreasury          float64                 `json:"totalTreasury"`
	Countries              []MarketCountrySnapshot `json:"countries"`
}

func DefaultConfig() Config {
	return Config{
		MarketAlgorithm: MarketAlgorithmLinear,
		Countries: []CountryConfig{
			{
				ID:   "coalreach",
				Name: "Coalreach",
				X:    0.18,
				Y:    0.34,
				Deposits: DepositSnapshot{
					Coal: 720, Oil: 60, Copper: 220, Silicon: 40,
				},
				Extractors: ExtractorConfig{
					Coal: 7.2, Oil: 0.8, Copper: 1.8, Silicon: 0.4,
				},
				Assets: AssetConfig{
					CoalPlant: 9, OilPlant: 1.5, SolarFarm: 1.2, WindFarm: 0.8,
				},
				SunPotential:      0.55,
				WindPotential:     0.40,
				BaseDemand:        21,
				StartingTreasury:  140,
				StartingReserve:   18,
				Infrastructure:    2.3,
				ComputeEfficiency: 0.14,
			},
			{
				ID:   "petroport",
				Name: "Petroport",
				X:    0.42,
				Y:    0.72,
				Deposits: DepositSnapshot{
					Coal: 160, Oil: 560, Copper: 140, Silicon: 60,
				},
				Extractors: ExtractorConfig{
					Coal: 1.6, Oil: 6.1, Copper: 1.1, Silicon: 0.5,
				},
				Assets: AssetConfig{
					CoalPlant: 2.5, OilPlant: 8, SolarFarm: 1.0, WindFarm: 0.7,
				},
				SunPotential:      0.65,
				WindPotential:     0.45,
				BaseDemand:        22,
				StartingTreasury:  155,
				StartingReserve:   16,
				Infrastructure:    2.5,
				ComputeEfficiency: 0.18,
			},
			{
				ID:   "silica",
				Name: "Silica",
				X:    0.70,
				Y:    0.28,
				Deposits: DepositSnapshot{
					Coal: 120, Oil: 80, Copper: 120, Silicon: 620,
				},
				Extractors: ExtractorConfig{
					Coal: 1.2, Oil: 0.7, Copper: 1.0, Silicon: 5.6,
				},
				Assets: AssetConfig{
					CoalPlant: 2, OilPlant: 1.5, SolarFarm: 4.6, WindFarm: 1.1,
				},
				SunPotential:      1.05,
				WindPotential:     0.55,
				BaseDemand:        20,
				StartingTreasury:  150,
				StartingReserve:   14,
				Infrastructure:    2.2,
				ComputeEfficiency: 0.25,
			},
			{
				ID:   "aurora",
				Name: "Aurora",
				X:    0.83,
				Y:    0.68,
				Deposits: DepositSnapshot{
					Coal: 110, Oil: 70, Copper: 260, Silicon: 90,
				},
				Extractors: ExtractorConfig{
					Coal: 1.0, Oil: 0.7, Copper: 2.0, Silicon: 0.8,
				},
				Assets: AssetConfig{
					CoalPlant: 1.8, OilPlant: 1.2, SolarFarm: 3.4, WindFarm: 5.2,
				},
				SunPotential:      0.98,
				WindPotential:     1.22,
				BaseDemand:        19,
				StartingTreasury:  145,
				StartingReserve:   15,
				Infrastructure:    2.4,
				ComputeEfficiency: 0.22,
			},
		},
		Routes: []RouteConfig{
			{A: "coalreach", B: "petroport", Capacity: 16},
			{A: "coalreach", B: "silica", Capacity: 12},
			{A: "petroport", B: "aurora", Capacity: 14},
			{A: "silica", B: "aurora", Capacity: 16},
			{A: "petroport", B: "silica", Capacity: 10},
		},
		CoalEnergyPerUnit: 2.35,
		OilEnergyPerUnit:  2.55,
		SolarUnitOutput:   2.1,
		WindUnitOutput:    2.2,

		ComputeEfficiencyDividend: 0.55,
		LocalLossRate:             0.11,
		TradeLossRate:             0.09,
		ComputeLossDividend:       1.4,
		EnergyReserveBase:         14,
		EnergyReservePerInfra:     3.8,
		GridCapacityBase:          10,
		GridCapacityPerInfra:      8,
		RouteInfraScale:           0.14,

		ComputeUpkeepBase:  1.6,
		ComputeUpkeepScale: 0.2,
		InfraUpkeepScale:   0.9,

		EnergySalePrice:        1.6,
		EmergencyEnergyPrice:   3.2,
		EmergencyEnergyPerTick: 4.5,
		CoalPrice:              2.6,
		OilPrice:               2.9,
		CopperPrice:            3.8,
		SiliconPrice:           4.4,

		IndustrialOutputPrice: 2.2,
		IndustrialOutputScale: 0.34,
		StabilityPenalty:      1.15,
		StabilityRecovery:     0.35,

		EnergyBuildTreasury: 42,
		EnergyBuildCopper:   6,
		EnergyBuildSilicon:  1,
		CoalPlantBuildGain:  4.5,
		OilPlantBuildGain:   4.2,
		SolarBuildGain:      3.1,
		WindBuildGain:       3.1,

		ComputeBuildTreasury:   36,
		ComputeBuildCopper:     5,
		ComputeBuildSilicon:    8,
		ComputeEfficiencyGain:  0.12,
		MaxComputeEfficiency:   1.2,
		InfrastructureTreasury: 34,
		InfrastructureCopper:   7,
		InfrastructureGain:     0.55,

		TradeEnergyShare: 0.65,
		TradeCargoShare:  0.35,
	}
}

func KnownMarketAlgorithms() []MarketAlgorithm {
	return []MarketAlgorithm{
		MarketAlgorithmNoTrade,
		MarketAlgorithmLinear,
		MarketAlgorithmScarcitySpike,
	}
}
