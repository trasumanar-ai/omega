export type DecisionUsage = {
  promptTokens: number
  completionTokens: number
  totalTokens: number
}

export type ResourceStockpile = {
  coal: number
  oil: number
  copper: number
  silicon: number
}

export type CountryAssets = {
  coalPlant: number
  oilPlant: number
  solarFarm: number
  windFarm: number
}

export type ResourceQuoteSnapshot = {
  resource: string
  net: number
  limitPrice: number
}

export type CountrySnapshot = {
  id: string
  name: string
  x: number
  y: number
  alive: boolean
  stability: number
  treasury: number
  energyReserve: number
  reserveCapacity: number
  gridCapacity: number
  infrastructure: number
  computeEfficiency: number
  baseDemand: number
  deposits: ResourceStockpile
  stockpiles: ResourceStockpile
  assets: CountryAssets
  lastDemand: number
  lastGeneratedEnergy: number
  lastDeliveredEnergy: number
  lastImportedEnergy: number
  lastExportedEnergy: number
  lastShortage: number
  lastTreasuryDelta: number
  lastTradeRevenue: number
  lastTradeSpend: number
  lastIndustrialRevenue: number
  lastEmergencySpend: number
  lastBuildSpend: number
  lastBuildFocus: string
  lastBuildKind: string
  lastBuildSuccess: boolean
  lastBuildFailure: string
  lastDecisionSource: string
  lastDecisionModel: string
  lastDecisionSummary: string
  lastDecisionError: string
  lastDecisionUsage: DecisionUsage
  marketQuotes: ResourceQuoteSnapshot[] | null
}

export type RouteSnapshot = {
  a: string
  b: string
  capacity: number
  lastEnergyAB: number
  lastEnergyBA: number
  lastCargoAB: number
  lastCargoBA: number
  effectiveCap: number
}

export type TickStats = {
  tick: number
  livingCountries: number
  totalEnergyGenerated: number
  totalEnergyDelivered: number
  totalEnergyTraded: number
  totalCargoTraded: number
  totalShortage: number
  totalTreasury: number
  avgComputeEfficiency: number
  energyBuilds: number
  computeBuilds: number
  infraBuilds: number
  coalRemaining: number
  oilRemaining: number
  copperRemaining: number
  siliconRemaining: number
}

export type CountryConfig = {
  id: string
  name: string
  x: number
  y: number
  deposits: ResourceStockpile
}

export type EconConfig = {
  marketAlgorithm: string
  countries: CountryConfig[]
}

export type WorldSnapshot = {
  tick: number
  countries: CountrySnapshot[]
  routes: RouteSnapshot[]
}

export type MarketCountrySnapshot = {
  id: string
  name: string
  treasury: number
  treasuryDelta: number
  energyReserve: number
  demand: number
  generatedEnergy: number
  deliveredEnergy: number
  importedEnergy: number
  exportedEnergy: number
  shortage: number
  tradeRevenue: number
  tradeSpend: number
  industrialRevenue: number
  emergencySpend: number
  buildSpend: number
  lastBuildFocus: string
  lastBuildKind: string
  quotes: ResourceQuoteSnapshot[] | null
}

export type MarketSnapshot = {
  energyPrice: number
  emergencyEnergyPrice: number
  coalPrice: number
  oilPrice: number
  copperPrice: number
  siliconPrice: number
  energyBuildTreasury: number
  computeBuildTreasury: number
  infrastructureTreasury: number
  totalDemand: number
  totalGenerated: number
  totalDelivered: number
  totalImported: number
  totalExported: number
  totalShortage: number
  totalTreasury: number
  countries: MarketCountrySnapshot[]
}

export type EconRunSummary = {
  tickCount: number
  currentTick: number
  livingCountries: number
  finalEnergyDelivered: number
  finalEnergyTraded: number
  finalCargoTraded: number
  finalShortage: number
  finalTreasury: number
}

export type EconRunInfo = {
  id: string
  status: string
  startedAt: string
  updatedAt: string
  completedAt?: string
  filePath: string
  llmMode: string
  llmModel: string
  recordCount: number
  summary: EconRunSummary
}

export type EconTickRecord = {
  tick: number
  recordedAt: string
  stats: TickStats
  world: WorldSnapshot
  market: MarketSnapshot
  events: string[]
}

export type EconRunRecord = EconRunInfo & {
  config: EconConfig
  records: EconTickRecord[]
}

export type EconCompareScenario = {
  algorithm: string
  label: string
  summary: {
    finalTick: number
    livingCountries: number
    finalEnergyDelivered: number
    finalShortage: number
    finalTreasury: number
    finalComputeEfficiency: number
    cumulativeEnergyTraded: number
    cumulativeCargoTraded: number
    cumulativeEnergyDelivered: number
    peakShortage: number
  }
  statsHistory: TickStats[]
  finalWorld: WorldSnapshot
  finalMarket: MarketSnapshot
  recentEvents: string[]
}

export type EconCompareResponse = {
  count: number
  scenarios: EconCompareScenario[]
}

export type EconLabPreset = {
  id: string
  label: string
  description: string
}

export type EconLabScenario = {
  id: string
  label: string
  presetId: string
  presetLabel: string
  algorithm: string
  summary: {
    finalTick: number
    livingCountries: number
    finalEnergyDelivered: number
    finalShortage: number
    finalTreasury: number
    finalComputeEfficiency: number
    cumulativeEnergyTraded: number
    cumulativeCargoTraded: number
    cumulativeEnergyDelivered: number
    peakShortage: number
  }
  statsHistory: TickStats[]
  world: WorldSnapshot
  market: MarketSnapshot
  recentEvents: string[]
}

export type EconLabSession = {
  id: string
  name: string
  createdAt: string
  updatedAt: string
  tick: number
  presetIds: string[]
  algorithmIds: string[]
  scenarios: EconLabScenario[]
}

export type EconLabResponse = {
  presets: EconLabPreset[]
  algorithms: string[]
  sessions: EconLabSession[]
}

export type EconLabArchiveScenarioScore = {
  scenarioId: string
  label: string
  livingCountries: number
  energyDelivered: number
  energyTraded: number
  peakShortage: number
  finalTreasury: number
  overallScore: number
}

export type EconLabArchiveSummary = {
  id: string
  name: string
  notes: string
  savedAt: string
  filePath: string
  sessionId: string
  sessionName: string
  tick: number
  scenarioCount: number
  presetIds: string[]
  algorithmIds: string[]
  winners: {
    bestOverall: string
    bestSurvival: string
    bestTrade: string
    bestTreasury: string
    lowestShortage: string
  }
  rankings: EconLabArchiveScenarioScore[]
}

export type EconLabArchiveRecord = {
  summary: EconLabArchiveSummary
  session: EconLabSession
}

export type EconExperimentRanking = {
  rank: number
  scenarioId: string
  label: string
  overallScore: number
  living: number
  peakShortage: number
  energyTrade: number
  energyDelivered: number
  treasury: number
}

export type EconExperimentComparison = {
  scenarioId: string
  label: string
  againstControlId: string
  deltaLivingCountries: number
  deltaPeakShortage: number
  deltaEnergyDelivered: number
  deltaEnergyTraded: number
  deltaFinalTreasury: number
  deltaOverallScore: number
  supportsHypothesisHint: boolean
}

export type EconExperimentTracePoint = {
  tick: number
  livingCountries: number
  totalEnergyGenerated: number
  totalEnergyDelivered: number
  totalEnergyTraded: number
  totalCargoTraded: number
  totalShortage: number
  totalTreasury: number
  avgComputeEfficiency: number
}

export type EconExperimentScenario = {
  spec: {
    id: string
    label: string
    role: string
    preset: string
    algorithm: string
    notes: string
  }
  summary: {
    finalTick: number
    livingCountries: number
    finalEnergyDelivered: number
    finalShortage: number
    finalTreasury: number
    finalComputeEfficiency: number
    cumulativeEnergyTraded: number
    cumulativeCargoTraded: number
    cumulativeEnergyDelivered: number
    peakShortage: number
  }
  trace: EconExperimentTracePoint[]
}

export type EconExperimentInfo = {
  id: string
  name: string
  question: string
  hypothesis: string
  startedAt: string
  endedAt: string
  outputPath: string
  decisionMode: string
  decisionModel: string
  controlScenarioId: string
  scenarioCount: number
  independentVariables: string[]
  dependentMetrics: string[]
  rankings: EconExperimentRanking[]
}

export type EconExperimentRecord = EconExperimentInfo & {
  version: number
  notes: string
  sampleEvery: number
  steps: number
  durationMs: number
  comparisons: EconExperimentComparison[]
  scenarios: EconExperimentScenario[]
}

export type EconStateResponse = {
  config: EconConfig
  stats: TickStats
  world: WorldSnapshot
  market: MarketSnapshot
  run: EconRunInfo
  recentEvents: string[]
  llmMode: string
  llmModel: string
}
