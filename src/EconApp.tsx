import { useEffect, useState, type ReactNode } from 'react'
import { TimeSeriesChart } from './ui/charts/TimeSeriesChart'
import { SparkLine } from './ui/charts/SparkLine'
import { useEconSimulation } from './hooks/useEconSimulation'
import type {
  CountrySnapshot,
  EconConfig,
  EconExperimentRecord,
  EconLabArchiveRecord,
  EconLabScenario,
  EconLabSession,
  EconRunRecord,
  MarketCountrySnapshot,
  MarketSnapshot,
  ResourceQuoteSnapshot,
  ResourceStockpile,
  WorldSnapshot,
} from './econ/types'

const NAV_ITEMS = [
  { id: 'overview', label: 'Overview', icon: '\u25A3' },
  { id: 'compare', label: 'Lab', icon: '\u25A5' },
  { id: 'experiments', label: 'Experiments', icon: '\u2697' },
  { id: 'market', label: 'Market', icon: '$' },
  { id: 'countries', label: 'Countries', icon: '\u2630' },
  { id: 'routes', label: 'Routes', icon: '\u27F7' },
  { id: 'llm', label: 'LLM', icon: '\u2699' },
  { id: 'runs', label: 'Runs', icon: '\u25CB' },
] as const

const RESOURCE_META: Array<{ key: keyof ResourceStockpile; label: string; color: string }> = [
  { key: 'coal', label: 'Coal', color: '#4b5563' },
  { key: 'oil', label: 'Oil', color: '#7c3f00' },
  { key: 'copper', label: 'Copper', color: '#b45309' },
  { key: 'silicon', label: 'Silicon', color: '#0f766e' },
]

const LAB_RECIPES = [
  {
    id: 'algo_default',
    label: 'Default x Algorithms',
    description: 'Keep resources fixed and compare all market algorithms.',
    presets: ['default'],
    algorithms: ['no_trade', 'linear', 'scarcity_spike'],
    suggestedName: 'Default x Algorithms',
  },
  {
    id: 'linear_resources',
    label: 'Linear x Resources',
    description: 'Keep linear matching fixed and vary resource distributions.',
    presets: ['default', 'fossil_skew', 'renewable_skew', 'metal_bottleneck'],
    algorithms: ['linear'],
    suggestedName: 'Linear x Resources',
  },
  {
    id: 'spike_resources',
    label: 'Spike x Resources',
    description: 'Stress each resource preset under scarcity-spike pricing.',
    presets: ['default', 'fossil_skew', 'renewable_skew', 'metal_bottleneck'],
    algorithms: ['scarcity_spike'],
    suggestedName: 'Spike x Resources',
  },
  {
    id: 'full_matrix',
    label: 'Full Matrix',
    description: 'Cross every preset with every algorithm in one sweep.',
    presets: ['default', 'fossil_skew', 'renewable_skew', 'metal_bottleneck'],
    algorithms: ['no_trade', 'linear', 'scarcity_spike'],
    suggestedName: 'Full Matrix',
  },
] as const

export function EconApp() {
  const {
    config,
    world,
    market,
    stats,
    statsHistory,
    recentEvents,
    currentRun,
    runs,
    experiments,
    lab,
    labArchive,
    labRunningAll,
    running,
    speed,
    setSpeed,
    llmMode,
    llmModel,
    backendStatus,
    toggle,
    step,
    reset,
    toggleAllLab,
    toggleLabSession,
    stepAllLab,
    stepLabSession,
    resetLabSession,
    resetAllLab,
    createLabSession,
    deleteLabSession,
    saveLabSession,
    loadExperiments,
    loadRuns,
    loadExperimentRecord,
    loadRunRecord,
    loadLabArchiveRecord,
    isLabSessionRunning,
  } = useEconSimulation()

  const [activeSection, setActiveSection] = useState('overview')
  const [selectedCountryId, setSelectedCountryId] = useState<string | null>(null)
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null)
  const [selectedLabSessionId, setSelectedLabSessionId] = useState<string | null>(null)
  const [selectedLabScenarioId, setSelectedLabScenarioId] = useState<string | null>(null)
  const [selectedRun, setSelectedRun] = useState<EconRunRecord | null>(null)
  const [runError, setRunError] = useState<string | null>(null)
  const [selectedExperimentId, setSelectedExperimentId] = useState<string | null>(null)
  const [selectedExperiment, setSelectedExperiment] = useState<EconExperimentRecord | null>(null)
  const [experimentError, setExperimentError] = useState<string | null>(null)
  const [selectedArchiveId, setSelectedArchiveId] = useState<string | null>(null)
  const [selectedArchive, setSelectedArchive] = useState<EconLabArchiveRecord | null>(null)
  const [archiveError, setArchiveError] = useState<string | null>(null)
  const [selectedPresetIds, setSelectedPresetIds] = useState<string[]>([])
  const [selectedAlgorithms, setSelectedAlgorithms] = useState<string[]>([])
  const [draftSessionName, setDraftSessionName] = useState('')
  const [draftArchiveName, setDraftArchiveName] = useState('')
  const [draftArchiveNotes, setDraftArchiveNotes] = useState('')
  const [stepBatchCount, setStepBatchCount] = useState(12)

  useEffect(() => {
    if (!world?.countries.length) return
    if (!selectedCountryId || !world.countries.some(country => country.id === selectedCountryId)) {
      setSelectedCountryId(world.countries[0].id)
    }
  }, [selectedCountryId, world])

  useEffect(() => {
    if (!selectedRunId && currentRun?.id) {
      setSelectedRunId(currentRun.id)
    }
  }, [currentRun, selectedRunId])

  useEffect(() => {
    if (!selectedExperimentId && experiments.length > 0) {
      setSelectedExperimentId(experiments[0].id)
    }
  }, [experiments, selectedExperimentId])

  useEffect(() => {
    if (!lab?.sessions.length) return
    if (!selectedLabSessionId || !lab.sessions.some(session => session.id === selectedLabSessionId)) {
      setSelectedLabSessionId(lab.sessions[0].id)
    }
  }, [lab, selectedLabSessionId])

  const selectedLabSession = lab?.sessions.find(session => session.id === selectedLabSessionId) ?? null

  useEffect(() => {
    if (!selectedLabSession?.scenarios.length) return
    if (!selectedLabScenarioId || !selectedLabSession.scenarios.some(scenario => scenario.id === selectedLabScenarioId)) {
      setSelectedLabScenarioId(selectedLabSession.scenarios[0].id)
    }
  }, [selectedLabScenarioId, selectedLabSession])

  useEffect(() => {
    if (!labArchive.length || !selectedArchiveId) return
    if (!labArchive.some(record => record.id === selectedArchiveId)) {
      setSelectedArchiveId(labArchive[0].id)
    }
  }, [labArchive, selectedArchiveId])

  useEffect(() => {
    if (!lab) return
    const currentSession = selectedLabSessionId
      ? lab.sessions.find(session => session.id === selectedLabSessionId) ?? null
      : null
    if (currentSession) {
      setSelectedPresetIds(currentSession.presetIds.length > 0 ? [...currentSession.presetIds] : lab.presets.map(preset => preset.id))
      setSelectedAlgorithms(currentSession.algorithmIds.length > 0 ? [...currentSession.algorithmIds] : [...lab.algorithms])
      return
    }
    if (selectedPresetIds.length === 0) {
      setSelectedPresetIds(lab.presets.map(preset => preset.id))
    }
    if (selectedAlgorithms.length === 0) {
      setSelectedAlgorithms([...lab.algorithms])
    }
  }, [lab, selectedAlgorithms.length, selectedLabSessionId, selectedPresetIds.length])

  useEffect(() => {
    if (!selectedRunId) return
    let cancelled = false
    setRunError(null)
    void loadRunRecord(selectedRunId)
      .then(run => {
        if (!cancelled) {
          setSelectedRun(run)
        }
      })
      .catch(err => {
        if (!cancelled) {
          setRunError(err instanceof Error ? err.message : 'Failed to load run')
        }
      })
    return () => {
      cancelled = true
    }
  }, [loadRunRecord, selectedRunId])

  useEffect(() => {
    if (!selectedExperimentId) {
      setSelectedExperiment(null)
      setExperimentError(null)
      return
    }
    let cancelled = false
    setExperimentError(null)
    void loadExperimentRecord(selectedExperimentId)
      .then(record => {
        if (!cancelled) {
          setSelectedExperiment(record)
        }
      })
      .catch(err => {
        if (!cancelled) {
          setExperimentError(err instanceof Error ? err.message : 'Failed to load experiment')
        }
      })
    return () => {
      cancelled = true
    }
  }, [loadExperimentRecord, selectedExperimentId])

  useEffect(() => {
    if (!selectedArchiveId) {
      setSelectedArchive(null)
      setArchiveError(null)
      return
    }
    let cancelled = false
    setArchiveError(null)
    void loadLabArchiveRecord(selectedArchiveId)
      .then(record => {
        if (!cancelled) {
          setSelectedArchive(record)
        }
      })
      .catch(err => {
        if (!cancelled) {
          setArchiveError(err instanceof Error ? err.message : 'Failed to load archive')
        }
      })
    return () => {
      cancelled = true
    }
  }, [loadLabArchiveRecord, selectedArchiveId])

  const selectedCountry = world?.countries.find(country => country.id === selectedCountryId) ?? null
  const selectedMarketCountry = market?.countries.find(country => country.id === selectedCountryId) ?? null
  const selectedLabScenario = selectedLabSession?.scenarios.find(scenario => scenario.id === selectedLabScenarioId) ?? null
  const mapWorld = activeSection === 'compare'
    ? (selectedArchive?.session.scenarios[0]?.world ?? selectedLabScenario?.world ?? selectedLabSession?.scenarios[0]?.world ?? null)
    : world
  const compareSessionRunning = selectedLabSession ? isLabSessionRunning(selectedLabSession.id) : false
  const estimatedInstances = selectedPresetIds.length * selectedAlgorithms.length
  const variableMode = describeVariableMode(selectedPresetIds.length, selectedAlgorithms.length)
  return (
    <div className="shell econ-shell">
      <nav className="navbar econ-navbar">
        <div className="navbar-brand">
          <span>omega</span>
          <small>econ</small>
        </div>
        <div className="navbar-items">
          {NAV_ITEMS.map(item => (
            <button
              key={item.id}
              className="navbar-item"
              data-active={activeSection === item.id}
              onClick={() => setActiveSection(item.id)}
            >
              <span className="navbar-icon">{item.icon}</span>
              <span>{item.label}</span>
            </button>
          ))}
        </div>
      </nav>

      <aside className="content-panel econ-panel">
        <div className="cp-body">
          <div className="econ-status-strip">
            <StatusBadge tone={backendStatus.connected ? 'ok' : 'warn'}>
              {backendStatus.connected ? 'Backend Connected' : 'Backend Down'}
            </StatusBadge>
            <StatusBadge tone={llmMode === 'openrouter' ? 'accent' : 'neutral'}>
              {llmMode === 'openrouter' ? (llmModel || 'OpenRouter') : 'Heuristic'}
            </StatusBadge>
            {currentRun && (
              <StatusBadge tone="neutral">
                run {currentRun.id}
              </StatusBadge>
            )}
          </div>

          <div className="btn-row">
            <button className="btn btn-primary" onClick={toggle}>
              {running ? 'Pause' : 'Start'}
            </button>
            <button className="btn" onClick={() => void step(1)} disabled={running}>Step</button>
            <button className="btn" onClick={() => void reset()}>Reset</button>
          </div>

          <div className="inline-field">
            <span className="inline-label">Speed</span>
            <div className="inline-slider">
              <input
                type="range"
                min={120}
                max={1800}
                step={20}
                value={speed}
                onChange={e => setSpeed(Number(e.target.value))}
              />
              <span className="inline-hint">{speed}ms</span>
            </div>
          </div>

          {backendStatus.error && (
            <div className="econ-warning-card">{backendStatus.error}</div>
          )}

          {activeSection === 'overview' && stats && (
            <>
              <div className="cp-stats-grid econ-stat-grid">
                <StatCard label="Tick" value={String(stats.tick)} />
                <StatCard label="Alive" value={String(stats.livingCountries)} />
                <StatCard label="Energy" value={stats.totalEnergyDelivered.toFixed(1)} />
                <StatCard label="Trade" value={stats.totalEnergyTraded.toFixed(1)} />
                <StatCard label="Treasury" value={stats.totalTreasury.toFixed(1)} />
                <StatCard label="Compute" value={stats.avgComputeEfficiency.toFixed(2)} />
              </div>

              <EventFeed title="Recent Events" events={recentEvents} />

              <SparkSection label={`Delivered Energy — ${stats.totalEnergyDelivered.toFixed(1)}`}>
                <SparkLine data={statsHistory} series={[{ key: 'totalEnergyDelivered', color: '#0f766e' }]} height={46} />
              </SparkSection>

              <SparkSection label={`Treasury Pool — ${stats.totalTreasury.toFixed(1)}`}>
                <SparkLine data={statsHistory} series={[{ key: 'totalTreasury', color: '#155e75' }]} height={46} />
              </SparkSection>

              <TimeSeriesChart
                title="Energy System"
                data={statsHistory}
                height={172}
                series={[
                  { key: 'totalEnergyDelivered', color: '#0f766e', label: 'Delivered' },
                  { key: 'totalEnergyGenerated', color: '#14532d', label: 'Generated' },
                  { key: 'totalEnergyTraded', color: '#0f4c81', label: 'Traded' },
                ]}
              />

              <TimeSeriesChart
                title="Pressure And Capital"
                data={statsHistory}
                height={164}
                series={[
                  { key: 'livingCountries', color: '#171717', label: 'Alive' },
                  { key: 'totalShortage', color: '#b45309', label: 'Shortage', type: 'area' },
                  { key: 'totalTreasury', color: '#155e75', label: 'Treasury' },
                ]}
              />
            </>
          )}

          {activeSection === 'market' && market && (
            <>
              <div className="cp-stats-grid econ-stat-grid">
                <StatCard label="Energy $" value={market.energyPrice.toFixed(1)} />
                <StatCard label="Emergency $" value={market.emergencyEnergyPrice.toFixed(1)} />
                <StatCard label="Coal $" value={market.coalPrice.toFixed(1)} />
                <StatCard label="Oil $" value={market.oilPrice.toFixed(1)} />
                <StatCard label="Copper $" value={market.copperPrice.toFixed(1)} />
                <StatCard label="Silicon $" value={market.siliconPrice.toFixed(1)} />
                <StatCard label="Demand" value={market.totalDemand.toFixed(1)} />
                <StatCard label="Treasury" value={market.totalTreasury.toFixed(1)} />
              </div>

              <EventFeed title="Market Tape" events={recentEvents} />

              <div className="econ-market-side-list">
                {market.countries
                  .slice()
                  .sort((a, b) => b.treasuryDelta - a.treasuryDelta)
                  .map(country => (
                    <div key={country.id} className="econ-country-item econ-market-country-item">
                      <div>
                        <strong>{country.name}</strong>
                        <span>{country.lastBuildFocus || 'hold'}</span>
                      </div>
                      <div>
                        <small>cash {country.treasury.toFixed(1)}</small>
                        <small>delta {formatSigned(country.treasuryDelta)}</small>
                      </div>
                    </div>
                  ))}
              </div>
            </>
          )}

          {activeSection === 'compare' && lab && (
            <>
              <div className="econ-run-current econ-lab-summary-card">
                <strong>Experiment Workbench</strong>
                <span>{lab.sessions.length} sessions</span>
                <small>{lab.sessions.reduce((sum, session) => sum + session.scenarios.length, 0)} total instances · {variableMode}</small>
              </div>

              <RecipeGroup
                recipes={LAB_RECIPES}
                onApply={recipe => {
                  setSelectedPresetIds([...recipe.presets])
                  setSelectedAlgorithms([...recipe.algorithms])
                  setDraftSessionName(recipe.suggestedName)
                }}
              />

              <div className="econ-setup-grid">
                <div className="econ-info-card">
                  <div className="econ-info-card-head">
                    <strong>Experiment Setup</strong>
                    <StatusBadge tone="accent">{estimatedInstances} instances</StatusBadge>
                  </div>
                  <div className="econ-builder-metrics">
                    <StatCard label="Presets" value={String(selectedPresetIds.length)} />
                    <StatCard label="Algorithms" value={String(selectedAlgorithms.length)} />
                    <StatCard label="Mode" value={variableMode} />
                  </div>
                  <div className="inline-field">
                    <span className="inline-label">Batch</span>
                    <div className="inline-slider">
                      <input
                        type="number"
                        min={1}
                        max={1200}
                        step={1}
                        value={stepBatchCount}
                        onChange={e => setStepBatchCount(Math.max(1, Number(e.target.value) || 1))}
                      />
                      <span className="inline-hint">ticks</span>
                    </div>
                  </div>
                </div>

                <div className="econ-info-card">
                  <div className="econ-info-card-head">
                    <strong>Create Session</strong>
                    <span className="econ-note">selected matrix becomes a new run set</span>
                  </div>
                  <div className="econ-session-create econ-session-create-stack">
                    <input
                      className="econ-session-input"
                      type="text"
                      placeholder="new session name"
                      value={draftSessionName}
                      onChange={e => setDraftSessionName(e.target.value)}
                    />
                    <button
                      className="btn"
                      onClick={() => {
                        const promise = createLabSession({
                          name: draftSessionName,
                          presets: selectedPresetIds,
                          algorithms: selectedAlgorithms,
                        })
                        void promise.then(result => {
                          const created = result.sessions[0]
                          if (created) {
                            setSelectedLabSessionId(created.id)
                            setSelectedLabScenarioId(created.scenarios[0]?.id ?? null)
                            setSelectedArchiveId(null)
                          }
                          setDraftSessionName('')
                        })
                      }}
                    >
                      New Session
                    </button>
                  </div>
                </div>
              </div>

              <div className="btn-row">
                <button
                  className="btn btn-primary"
                  onClick={() => {
                    if (!selectedLabSession) return
                    toggleLabSession(selectedLabSession.id)
                  }}
                  disabled={!selectedLabSession}
                >
                  {compareSessionRunning ? 'Pause Session' : 'Run Session'}
                </button>
                <button
                  className="btn"
                  onClick={() => {
                    if (!selectedLabSession) return
                    void stepLabSession(selectedLabSession.id, 1)
                  }}
                  disabled={!selectedLabSession || compareSessionRunning}
                >
                  Step Session
                </button>
                <button
                  className="btn"
                  onClick={toggleAllLab}
                >
                  {labRunningAll ? 'Pause All' : 'Run All'}
                </button>
              </div>

              <div className="btn-row">
                <button className="btn" onClick={() => void stepAllLab(stepBatchCount)} disabled={labRunningAll}>Dispatch All x{stepBatchCount}</button>
                <button className="btn" onClick={() => void resetAllLab()}>Reset All</button>
                <button
                  className="btn"
                  onClick={() => {
                    if (!selectedLabSession) return
                    void stepLabSession(selectedLabSession.id, stepBatchCount)
                  }}
                  disabled={!selectedLabSession || compareSessionRunning}
                >
                  Dispatch Session x{stepBatchCount}
                </button>
              </div>

              <div className="btn-row">
                <button
                  className="btn"
                  onClick={() => {
                    if (!selectedLabSession) return
                    void resetLabSession(selectedLabSession.id, { presets: selectedPresetIds, algorithms: selectedAlgorithms })
                  }}
                  disabled={!selectedLabSession}
                >
                  Rebuild Session
                </button>
                <button className="btn" onClick={() => setSelectedArchiveId(null)}>
                  Focus Live Session
                </button>
              </div>

              <SelectionGroup
                title="Presets"
                options={lab.presets.map(preset => ({ id: preset.id, label: preset.label, hint: preset.description }))}
                selected={selectedPresetIds}
                onToggle={id => setSelectedPresetIds(toggleSelection(selectedPresetIds, id))}
              />

              <SelectionGroup
                title="Algorithms"
                options={lab.algorithms.map(algorithm => ({ id: algorithm, label: algorithm, hint: 'market matching mode' }))}
                selected={selectedAlgorithms}
                onToggle={id => setSelectedAlgorithms(toggleSelection(selectedAlgorithms, id))}
              />

              <div className="econ-run-current">
                <strong>Lab Sessions</strong>
                <span>{lab.sessions.length} sessions</span>
                <small>{lab.sessions.reduce((sum, session) => sum + session.scenarios.length, 0)} total instances</small>
              </div>

              <div className="econ-session-list">
                {lab.sessions.map(session => (
                  <button
                    key={session.id}
                    className="econ-run-item econ-session-item"
                    data-selected={session.id === selectedLabSessionId}
                    onClick={() => {
                      setSelectedLabSessionId(session.id)
                      setSelectedArchiveId(null)
                    }}
                  >
                    <div>
                      <strong>{session.name}</strong>
                      <span>{session.scenarios.length} instances · tick {session.tick}</span>
                    </div>
                    <div>
                      <small>{compareSessionHeadline(session)}</small>
                      <small>{isLabSessionRunning(session.id) ? 'running' : 'idle'}</small>
                    </div>
                  </button>
                ))}
              </div>

              {selectedLabSession && (
                <>
                  <div className="econ-run-current">
                    <strong>{selectedLabSession.name}</strong>
                    <span>{selectedLabSession.scenarios.length} instances</span>
                    <small>tick {selectedLabSession.tick} · {compareSessionHeadline(selectedLabSession)}</small>
                  </div>

                  <div className="econ-info-card">
                    <div className="econ-info-card-head">
                      <strong>Save Result Snapshot</strong>
                      <span className="econ-note">archive current session state</span>
                    </div>
                    <div className="econ-session-create econ-session-create-stack">
                      <input
                        className="econ-session-input"
                        type="text"
                        placeholder="archive name"
                        value={draftArchiveName}
                        onChange={e => setDraftArchiveName(e.target.value)}
                      />
                      <textarea
                        className="econ-session-notes"
                        placeholder="what changed in this experiment?"
                        value={draftArchiveNotes}
                        onChange={e => setDraftArchiveNotes(e.target.value)}
                      />
                      <button
                        className="btn"
                        onClick={() => {
                          if (!selectedLabSession) return
                          void saveLabSession(selectedLabSession.id, {
                            name: draftArchiveName,
                            notes: draftArchiveNotes,
                          }).then(records => {
                            const latest = records[0]
                            setSelectedArchiveId(latest?.id ?? null)
                            setDraftArchiveName('')
                            setDraftArchiveNotes('')
                          })
                        }}
                      >
                        Save To Archive
                      </button>
                    </div>
                  </div>

                  <div className="btn-row">
                    <button
                      className="btn"
                      onClick={() => {
                        if (!selectedLabSession) return
                        void deleteLabSession(selectedLabSession.id).then(result => {
                          const nextSession = result.sessions[0] ?? null
                          setSelectedLabSessionId(nextSession?.id ?? null)
                          setSelectedLabScenarioId(nextSession?.scenarios[0]?.id ?? null)
                        })
                      }}
                      disabled={!selectedLabSession}
                    >
                      Delete Session
                    </button>
                  </div>

                  <div className="econ-run-list">
                    {selectedLabSession.scenarios.map(scenario => (
                      <button
                        key={scenario.id}
                        className="econ-run-item"
                        data-selected={scenario.id === selectedLabScenarioId}
                        onClick={() => {
                          setSelectedLabScenarioId(scenario.id)
                          setSelectedArchiveId(null)
                        }}
                      >
                        <div>
                          <strong>{scenario.label}</strong>
                          <span>{scenario.presetLabel} · {scenario.algorithm}</span>
                        </div>
                        <div>
                          <small>alive {scenario.summary.livingCountries}</small>
                          <small>trade {scenario.summary.cumulativeEnergyTraded.toFixed(1)} · short {scenario.summary.peakShortage.toFixed(1)}</small>
                        </div>
                      </button>
                    ))}
                  </div>
                </>
              )}

              <div className="econ-run-current">
                <strong>Saved Results</strong>
                <span>{labArchive.length} archived experiments</span>
                <small>global result store</small>
              </div>

              <div className="econ-session-list">
                {labArchive.map(record => (
                  <button
                    key={record.id}
                    className="econ-run-item econ-session-item"
                    data-selected={record.id === selectedArchiveId}
                    onClick={() => setSelectedArchiveId(record.id)}
                  >
                    <div>
                      <strong>{record.name}</strong>
                      <span>{record.sessionName} · tick {record.tick}</span>
                    </div>
                    <div>
                      <small>{record.winners.bestOverall || 'no winner'}</small>
                      <small>{formatStamp(record.savedAt)}</small>
                    </div>
                  </button>
                ))}
              </div>
            </>
          )}

          {activeSection === 'experiments' && (
            <>
              <div className="btn-row">
                <button className="btn" onClick={() => void loadExperiments()}>Refresh Experiments</button>
              </div>

              <div className="econ-run-current">
                <strong>CLI Experiments</strong>
                <span>{experiments.length} saved runs</span>
                <small>generated by `omega-econ experiment run`</small>
              </div>

              <div className="econ-run-list">
                {experiments.map(experiment => (
                  <button
                    key={experiment.id}
                    className="econ-run-item"
                    data-selected={experiment.id === selectedExperimentId}
                    onClick={() => setSelectedExperimentId(experiment.id)}
                  >
                    <div>
                      <strong>{experiment.name}</strong>
                      <span>{experiment.scenarioCount} scenarios · {experiment.decisionMode}</span>
                    </div>
                    <div>
                      <small>{experiment.rankings[0]?.label ?? 'no ranking'}</small>
                      <small>{formatStamp(experiment.endedAt)}</small>
                    </div>
                  </button>
                ))}
              </div>
            </>
          )}

          {activeSection === 'countries' && world && (
            <div className="econ-country-list">
              {world.countries.map(country => (
                <button
                  key={country.id}
                  className="econ-country-item"
                  data-selected={country.id === selectedCountryId}
                  onClick={() => setSelectedCountryId(country.id)}
                >
                  <div>
                    <strong>{country.name}</strong>
                    <span>{country.alive ? 'alive' : 'down'}</span>
                  </div>
                  <div>
                    <small>reserve {country.energyReserve.toFixed(1)}</small>
                    <small>cash {country.treasury.toFixed(1)}</small>
                  </div>
                </button>
              ))}
            </div>
          )}

          {activeSection === 'routes' && world && (
            <div className="econ-route-list">
              {world.routes.map(route => (
                <div key={`${route.a}-${route.b}`} className="econ-route-card">
                  <div className="econ-route-head">
                    <strong>{route.a}</strong>
                    <span>{'\u2194'}</span>
                    <strong>{route.b}</strong>
                  </div>
                  <div className="econ-route-metrics">
                    <span>cap {route.effectiveCap.toFixed(1)}</span>
                    <span>energy {(route.lastEnergyAB + route.lastEnergyBA).toFixed(1)}</span>
                    <span>cargo {(route.lastCargoAB + route.lastCargoBA).toFixed(1)}</span>
                  </div>
                </div>
              ))}
            </div>
          )}

          {activeSection === 'llm' && world && (
            <div className="econ-llm-list">
              {world.countries.map(country => (
                <div key={country.id} className="econ-llm-card">
                  <div className="econ-llm-head">
                    <strong>{country.name}</strong>
                    <StatusBadge tone={country.lastDecisionSource === 'openrouter' ? 'accent' : 'neutral'}>
                      {country.lastDecisionSource || 'none'}
                    </StatusBadge>
                  </div>
                  <div className="econ-llm-meta">
                    <span>{country.lastDecisionModel || 'builtin/default'}</span>
                    <span>{country.lastBuildFocus || 'hold'}</span>
                  </div>
                  <p>{country.lastDecisionSummary || country.lastBuildFailure || 'no decision note yet'}</p>
                  <div className="econ-llm-usage">
                    <span>prompt {country.lastDecisionUsage.promptTokens}</span>
                    <span>completion {country.lastDecisionUsage.completionTokens}</span>
                    <span>total {country.lastDecisionUsage.totalTokens}</span>
                  </div>
                  {country.lastDecisionError && (
                    <div className="econ-llm-error">{country.lastDecisionError}</div>
                  )}
                </div>
              ))}
            </div>
          )}

          {activeSection === 'runs' && (
            <>
              <div className="btn-row">
                <button className="btn" onClick={() => void loadRuns()}>Refresh Runs</button>
              </div>
              {currentRun && (
                <div className="econ-run-current">
                  <strong>Current Run</strong>
                  <span>{currentRun.id}</span>
                  <small>{currentRun.recordCount} records · tick {currentRun.summary.currentTick}</small>
                </div>
              )}
              <div className="econ-run-list">
                {runs.map(run => (
                  <button
                    key={run.id}
                    className="econ-run-item"
                    data-selected={run.id === selectedRunId}
                    onClick={() => setSelectedRunId(run.id)}
                  >
                    <div>
                      <strong>{run.id}</strong>
                      <span>{run.status}</span>
                    </div>
                    <div>
                      <small>tick {run.summary.currentTick}</small>
                      <small>alive {run.summary.livingCountries}</small>
                    </div>
                  </button>
                ))}
              </div>
            </>
          )}
        </div>
      </aside>

      <main className="econ-main">
        <div className="econ-topbar">
          <div>
            <h1>Country Energy Network</h1>
            <p>Static resources, passive extraction, treasury flows, build decisions, live route flows.</p>
          </div>
          {stats && (
            <div className="econ-topbar-metrics">
              <MetricChip label="Shortage" value={stats.totalShortage.toFixed(1)} tone={stats.totalShortage > 0 ? 'warn' : 'ok'} />
              <MetricChip label="Trade" value={stats.totalEnergyTraded.toFixed(1)} tone="accent" />
              <MetricChip label="Treasury" value={stats.totalTreasury.toFixed(1)} tone="neutral" />
              <MetricChip label="Tick" value={String(stats.tick)} tone="neutral" />
            </div>
          )}
        </div>

        <section className="econ-map-card">
          {mapWorld ? (
            <EconNetworkMap
              world={mapWorld}
              selectedCountryId={activeSection === 'compare' ? null : selectedCountryId}
              onSelectCountry={activeSection === 'compare' ? () => {} : setSelectedCountryId}
            />
          ) : (
            <div className="econ-map-empty">waiting for economy state...</div>
          )}
        </section>

        {activeSection === 'market' && market ? (
          <section className="econ-detail-card">
            <MarketDetail market={market} />
          </section>
        ) : null}

        {activeSection === 'compare' && (selectedArchive || selectedLabScenario || selectedLabSession) ? (
          <section className="econ-detail-card">
            {selectedArchive ? (
              <ArchiveDetail archive={selectedArchive} archiveError={archiveError} />
            ) : selectedLabSession ? (
              <CompareSessionDetail session={selectedLabSession} selectedScenario={selectedLabScenario} />
            ) : null}
          </section>
        ) : null}

        {activeSection === 'runs' ? (
          <section className="econ-detail-card">
            <RunDetail run={selectedRun} runError={runError} />
          </section>
        ) : null}

        {activeSection === 'experiments' ? (
          <section className="econ-detail-card">
            <ExperimentDetail experiment={selectedExperiment} experimentError={experimentError} />
          </section>
        ) : null}

        {activeSection !== 'market' && activeSection !== 'runs' && activeSection !== 'experiments' && selectedCountry ? (
          <section className="econ-detail-card">
            <div className="econ-detail-head">
              <div>
                <h2>{selectedCountry.name}</h2>
                <p>{selectedCountry.alive ? 'Operating' : 'Collapsed'} · {selectedCountry.lastBuildFocus || 'hold'} · {selectedCountry.lastDecisionSource || 'no source'}</p>
              </div>
              <StatusBadge tone={countryTone(selectedCountry)}>
                stability {selectedCountry.stability.toFixed(0)}
              </StatusBadge>
            </div>

            <div className="cp-stats-grid econ-stat-grid econ-detail-stats">
              <StatCard label="Reserve" value={selectedCountry.energyReserve.toFixed(1)} />
              <StatCard label="Grid Cap" value={selectedCountry.gridCapacity.toFixed(1)} />
              <StatCard label="Compute" value={selectedCountry.computeEfficiency.toFixed(2)} />
              <StatCard label="Treasury" value={selectedCountry.treasury.toFixed(1)} />
              <StatCard label="Demand" value={selectedCountry.lastDemand.toFixed(1)} />
              <StatCard label="Shortage" value={selectedCountry.lastShortage.toFixed(1)} />
            </div>

            <div className="econ-resource-grid">
              <ResourcePanel title="Deposits" current={selectedCountry.deposits} initial={initialDeposits(config, selectedCountry.id)} />
              <ResourcePanel title="Stockpiles" current={selectedCountry.stockpiles} />
              <AssetPanel country={selectedCountry} />
              <FinancePanel country={selectedCountry} marketCountry={selectedMarketCountry} />
              <QuotePanel title="Posted Quotes" quotes={selectedCountry.marketQuotes ?? []} />
              <DecisionPanel country={selectedCountry} />
            </div>
          </section>
        ) : null}
      </main>
    </div>
  )
}

function EconNetworkMap({
  world,
  selectedCountryId,
  onSelectCountry,
}: {
  world: WorldSnapshot
  selectedCountryId: string | null
  onSelectCountry: (countryId: string) => void
}) {
  return (
    <svg className="econ-map-svg" viewBox="0 0 1000 680" preserveAspectRatio="xMidYMid meet">
      <defs>
        <linearGradient id="econStageGlow" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#fbf8ef" />
          <stop offset="100%" stopColor="#eef4f0" />
        </linearGradient>
      </defs>
      <rect x="0" y="0" width="1000" height="680" rx="28" fill="url(#econStageGlow)" />

      {world.routes.map(route => {
        const from = world.countries.find(country => country.id === route.a)
        const to = world.countries.find(country => country.id === route.b)
        if (!from || !to) return null

        const x1 = 120 + from.x * 760
        const y1 = 80 + from.y * 500
        const x2 = 120 + to.x * 760
        const y2 = 80 + to.y * 500
        const energyFlow = route.lastEnergyAB + route.lastEnergyBA
        const cargoFlow = route.lastCargoAB + route.lastCargoBA
        const width = 2 + Math.min(10, route.effectiveCap * 0.18)

        return (
          <g key={`${route.a}-${route.b}`}>
            <line x1={x1} y1={y1} x2={x2} y2={y2} stroke="rgba(17,24,39,0.14)" strokeWidth={width} strokeLinecap="round" />
            {energyFlow > 0.05 && (
              <line
                className="econ-energy-flow"
                x1={x1}
                y1={y1}
                x2={x2}
                y2={y2}
                stroke="#0f766e"
                strokeWidth={Math.max(2, width * 0.55)}
                strokeLinecap="round"
              />
            )}
            {cargoFlow > 0.05 && (
              <line
                className="econ-cargo-flow"
                x1={x1}
                y1={y1}
                x2={x2}
                y2={y2}
                stroke="#b45309"
                strokeWidth={1.8}
                strokeLinecap="round"
              />
            )}
            <text x={(x1 + x2) / 2} y={(y1 + y2) / 2 - 10} className="econ-route-label">
              {energyFlow.toFixed(1)}e / {cargoFlow.toFixed(1)}c
            </text>
          </g>
        )
      })}

      {world.countries.map(country => {
        const cx = 120 + country.x * 760
        const cy = 80 + country.y * 500
        const radius = 32 + country.computeEfficiency * 11
        const reserveRatio = Math.max(0.08, Math.min(1, country.energyReserve / Math.max(1, country.reserveCapacity)))

        return (
          <g
            key={country.id}
            className="econ-node"
            data-selected={country.id === selectedCountryId}
            onClick={() => onSelectCountry(country.id)}
          >
            <circle
              cx={cx}
              cy={cy}
              r={radius + 11}
              fill="none"
              stroke="rgba(15,118,110,0.16)"
              strokeWidth="8"
              strokeDasharray={`${reserveRatio * 240} 240`}
              transform={`rotate(-90 ${cx} ${cy})`}
            />
            <circle
              cx={cx}
              cy={cy}
              r={radius}
              fill={nodeFill(country)}
              stroke={country.id === selectedCountryId ? '#111827' : 'rgba(17,24,39,0.16)'}
              strokeWidth={country.id === selectedCountryId ? 3 : 1.5}
            />
            <text x={cx} y={cy - 5} className="econ-node-title">{country.name}</text>
            <text x={cx} y={cy + 16} className="econ-node-subtitle">
              {country.lastShortage > 0 ? `shortage ${country.lastShortage.toFixed(1)}` : `cash ${country.treasury.toFixed(0)}`}
            </text>
          </g>
        )
      })}
    </svg>
  )
}

function ResourcePanel({
  title,
  current,
  initial,
}: {
  title: string
  current: ResourceStockpile
  initial?: ResourceStockpile | null
}) {
  return (
    <div className="econ-info-card">
      <div className="econ-info-card-head">
        <strong>{title}</strong>
      </div>
      <div className="econ-resource-list">
        {RESOURCE_META.map(resource => {
          const ratio = initial ? current[resource.key] / Math.max(1, initial[resource.key]) : Math.min(1, current[resource.key] / 100)
          return (
            <div key={resource.key} className="econ-resource-row">
              <div className="econ-resource-copy">
                <span>{resource.label}</span>
                <strong>{current[resource.key].toFixed(1)}</strong>
              </div>
              <div className="econ-meter">
                <div className="econ-meter-fill" style={{ width: `${Math.max(4, Math.min(100, ratio * 100))}%`, background: resource.color }} />
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function AssetPanel({ country }: { country: CountrySnapshot }) {
  return (
    <div className="econ-info-card">
      <div className="econ-info-card-head">
        <strong>Assets</strong>
      </div>
      <div className="econ-asset-list">
        <AssetRow label="Coal Plant" value={country.assets.coalPlant} />
        <AssetRow label="Oil Plant" value={country.assets.oilPlant} />
        <AssetRow label="Solar Farm" value={country.assets.solarFarm} />
        <AssetRow label="Wind Farm" value={country.assets.windFarm} />
        <AssetRow label="Infrastructure" value={country.infrastructure} />
      </div>
    </div>
  )
}

function FinancePanel({
  country,
  marketCountry,
}: {
  country: CountrySnapshot
  marketCountry: MarketCountrySnapshot | null
}) {
  return (
    <div className="econ-info-card">
      <div className="econ-info-card-head">
        <strong>Treasury Flow</strong>
        <StatusBadge tone={country.lastTreasuryDelta >= 0 ? 'ok' : 'warn'}>
          {formatSigned(country.lastTreasuryDelta)}
        </StatusBadge>
      </div>
      <div className="econ-asset-list">
        <AssetRow label="Treasury" value={country.treasury} />
        <AssetRow label="Trade Rev" value={country.lastTradeRevenue} />
        <AssetRow label="Trade Spend" value={country.lastTradeSpend} />
        <AssetRow label="Industrial Rev" value={country.lastIndustrialRevenue} />
        <AssetRow label="Emergency Spend" value={country.lastEmergencySpend} />
        <AssetRow label="Build Spend" value={country.lastBuildSpend} />
        {marketCountry ? <AssetRow label="Demand" value={marketCountry.demand} /> : null}
      </div>
    </div>
  )
}

function QuotePanel({ title, quotes }: { title: string; quotes: ResourceQuoteSnapshot[] | null | undefined }) {
  const safeQuotes = quotes ?? []
  return (
    <div className="econ-info-card">
      <div className="econ-info-card-head">
        <strong>{title}</strong>
        <span className="econ-note">+ buy / - sell</span>
      </div>
      <div className="econ-asset-list">
        {safeQuotes.map(quote => (
          <div key={quote.resource} className="econ-asset-row">
            <span>{quote.resource}</span>
            <strong>{formatSigned(quote.net)} @ {quote.limitPrice.toFixed(2)}</strong>
          </div>
        ))}
      </div>
    </div>
  )
}

function SelectionGroup({
  title,
  options,
  selected,
  onToggle,
}: {
  title: string
  options: Array<{ id: string; label: string; hint: string }>
  selected: string[]
  onToggle: (id: string) => void
}) {
  return (
    <div className="econ-info-card">
      <div className="econ-info-card-head">
        <strong>{title}</strong>
      </div>
      <div className="econ-selection-list">
        {options.map(option => (
          <button
            key={option.id}
            className="econ-selection-chip"
            data-selected={selected.includes(option.id)}
            onClick={() => onToggle(option.id)}
          >
            <span>{option.label}</span>
            <small>{option.hint}</small>
          </button>
        ))}
      </div>
    </div>
  )
}

function RecipeGroup({
  recipes,
  onApply,
}: {
  recipes: typeof LAB_RECIPES
  onApply: (recipe: (typeof LAB_RECIPES)[number]) => void
}) {
  return (
    <div className="econ-info-card">
      <div className="econ-info-card-head">
        <strong>Quick Recipes</strong>
        <span className="econ-note">one click experiment templates</span>
      </div>
      <div className="econ-recipe-list">
        {recipes.map(recipe => (
          <button key={recipe.id} className="econ-recipe-card" onClick={() => onApply(recipe)}>
            <strong>{recipe.label}</strong>
            <span>{recipe.description}</span>
            <small>{recipe.presets.length} presets · {recipe.algorithms.length} algorithms</small>
          </button>
        ))}
      </div>
    </div>
  )
}

function CompareSessionDetail({
  session,
  selectedScenario,
}: {
  session: EconLabSession
  selectedScenario: EconLabScenario | null
}) {
  const ranked = rankSessionScenarios(session.scenarios)
  return (
    <>
      <div className="econ-detail-head">
        <div>
          <h2>{session.name}</h2>
          <p>{session.scenarios.length} instances across {session.presetIds.length} presets and {session.algorithmIds.length} algorithms.</p>
        </div>
        <StatusBadge tone="accent">
          tick {session.tick}
        </StatusBadge>
      </div>

      <div className="cp-stats-grid econ-stat-grid econ-detail-stats">
        <StatCard label="Scenarios" value={String(session.scenarios.length)} />
        <StatCard label="Best Alive" value={String(Math.max(...session.scenarios.map(scenario => scenario.summary.livingCountries)))} />
        <StatCard label="Best Trade" value={Math.max(...session.scenarios.map(scenario => scenario.summary.cumulativeEnergyTraded)).toFixed(1)} />
        <StatCard label="Low Short" value={Math.min(...session.scenarios.map(scenario => scenario.summary.peakShortage)).toFixed(1)} />
        <StatCard label="Presets" value={String(session.presetIds.length)} />
        <StatCard label="Algorithms" value={String(session.algorithmIds.length)} />
      </div>

      <div className="econ-market-grid">
        <div className="econ-info-card">
          <div className="econ-info-card-head">
            <strong>Session Rankings</strong>
          </div>
          <div className="econ-ranking-list">
            {ranked.map((scenario, index) => (
              <div key={scenario.id} className="econ-ranking-row">
                <span>#{index + 1}</span>
                <strong>{scenario.label}</strong>
                <small>alive {scenario.summary.livingCountries} · trade {scenario.summary.cumulativeEnergyTraded.toFixed(1)} · short {scenario.summary.peakShortage.toFixed(1)}</small>
              </div>
            ))}
          </div>
        </div>

        <div className="econ-info-card">
          <div className="econ-info-card-head">
            <strong>Variable Axes</strong>
          </div>
          <div className="econ-asset-list">
            <MetaRow label="Mode" value={describeVariableMode(session.presetIds.length, session.algorithmIds.length)} />
            <MetaRow label="Presets" value={session.presetIds.join(', ')} />
            <MetaRow label="Algorithms" value={session.algorithmIds.join(', ')} />
          </div>
        </div>
      </div>

      {selectedScenario ? <LabDetail scenario={selectedScenario} /> : null}
    </>
  )
}

function ArchiveDetail({
  archive,
  archiveError,
}: {
  archive: EconLabArchiveRecord | null
  archiveError: string | null
}) {
  if (archiveError) {
    return <div className="econ-warning-card">{archiveError}</div>
  }

  if (!archive) {
    return <div className="econ-map-empty">waiting for archived experiment...</div>
  }

  return (
    <>
      <div className="econ-detail-head">
        <div>
          <h2>{archive.summary.name}</h2>
          <p>{archive.summary.sessionName} · saved {formatStamp(archive.summary.savedAt)}</p>
        </div>
        <StatusBadge tone="accent">
          tick {archive.summary.tick}
        </StatusBadge>
      </div>

      <div className="cp-stats-grid econ-stat-grid econ-detail-stats">
        <StatCard label="Scenarios" value={String(archive.summary.scenarioCount)} />
        <StatCard label="Presets" value={String(archive.summary.presetIds.length)} />
        <StatCard label="Algorithms" value={String(archive.summary.algorithmIds.length)} />
        <StatCard label="Tick" value={String(archive.summary.tick)} />
        <StatCard label="Ranked" value={String(archive.summary.rankings.length)} />
        <StatCard label="Top Score" value={archive.summary.rankings[0] ? archive.summary.rankings[0].overallScore.toFixed(0) : '0'} />
      </div>

      <div className="econ-market-grid">
        <div className="econ-info-card">
          <div className="econ-info-card-head">
            <strong>Archive Notes</strong>
          </div>
          <div className="econ-asset-list">
            <MetaRow label="File" value={archive.summary.filePath} mono />
            <MetaRow label="Session" value={archive.summary.sessionName} />
            <MetaRow label="Presets" value={archive.summary.presetIds.join(', ')} />
            <MetaRow label="Algorithms" value={archive.summary.algorithmIds.join(', ')} />
            <MetaRow label="Best Overall" value={archive.summary.winners.bestOverall || 'n/a'} />
            <MetaRow label="Best Trade" value={archive.summary.winners.bestTrade || 'n/a'} />
            <MetaRow label="Lowest Short" value={archive.summary.winners.lowestShortage || 'n/a'} />
          </div>
          <p className="econ-archive-notes">{archive.summary.notes || 'No notes saved for this experiment.'}</p>
        </div>

        <div className="econ-info-card">
          <div className="econ-info-card-head">
            <strong>Ranking Snapshot</strong>
          </div>
          <div className="econ-ranking-list">
            {archive.summary.rankings.slice(0, 6).map((scenario, index) => (
              <div key={scenario.scenarioId} className="econ-ranking-row">
                <span>#{index + 1}</span>
                <strong>{scenario.label}</strong>
                <small>score {scenario.overallScore.toFixed(0)} · alive {scenario.livingCountries} · short {scenario.peakShortage.toFixed(1)}</small>
              </div>
            ))}
          </div>
        </div>
      </div>
    </>
  )
}

function ExperimentDetail({
  experiment,
  experimentError,
}: {
  experiment: EconExperimentRecord | null
  experimentError: string | null
}) {
  if (experimentError) {
    return <div className="econ-warning-card">{experimentError}</div>
  }

  if (!experiment) {
    return <div className="econ-map-empty">waiting for experiment record...</div>
  }

  const topScenario = experiment.scenarios.find(scenario => scenario.spec.id === experiment.rankings[0]?.scenarioId) ?? experiment.scenarios[0] ?? null

  return (
    <>
      <div className="econ-detail-head">
        <div>
          <h2>{experiment.name}</h2>
          <p>{experiment.question}</p>
        </div>
        <StatusBadge tone="accent">
          {experiment.decisionMode}{experiment.decisionModel ? ` · ${experiment.decisionModel}` : ''}
        </StatusBadge>
      </div>

      <div className="cp-stats-grid econ-stat-grid econ-detail-stats">
        <StatCard label="Scenarios" value={String(experiment.scenarioCount)} />
        <StatCard label="Steps" value={String(experiment.steps)} />
        <StatCard label="Sample" value={String(experiment.sampleEvery)} />
        <StatCard label="Control" value={experiment.controlScenarioId || 'none'} />
        <StatCard label="Top Rank" value={experiment.rankings[0]?.label ?? 'n/a'} />
        <StatCard label="Duration" value={`${experiment.durationMs}ms`} />
      </div>

      <div className="econ-market-grid">
        <div className="econ-info-card">
          <div className="econ-info-card-head">
            <strong>Hypothesis</strong>
          </div>
          <p className="econ-archive-notes">{experiment.hypothesis}</p>
          <div className="econ-asset-list">
            <MetaRow label="Independent" value={experiment.independentVariables.join(', ')} />
            <MetaRow label="Dependent" value={experiment.dependentMetrics.join(', ')} />
            <MetaRow label="Started" value={formatStamp(experiment.startedAt)} />
            <MetaRow label="Ended" value={formatStamp(experiment.endedAt)} />
            <MetaRow label="File" value={experiment.outputPath} mono />
          </div>
        </div>

        <div className="econ-info-card">
          <div className="econ-info-card-head">
            <strong>Rankings</strong>
          </div>
          <div className="econ-ranking-list">
            {experiment.rankings.map(ranking => (
              <div key={ranking.scenarioId} className="econ-ranking-row">
                <span>#{ranking.rank}</span>
                <strong>{ranking.label}</strong>
                <small>score {ranking.overallScore.toFixed(0)} · alive {ranking.living} · short {ranking.peakShortage.toFixed(1)} · trade {ranking.energyTrade.toFixed(1)}</small>
              </div>
            ))}
          </div>
        </div>
      </div>

      {experiment.comparisons.length > 0 && (
        <div className="econ-info-card">
          <div className="econ-info-card-head">
            <strong>Control Comparison</strong>
          </div>
          <div className="econ-ranking-list">
            {experiment.comparisons.map(comparison => (
              <div key={comparison.scenarioId} className="econ-ranking-row">
                <strong>{comparison.label}</strong>
                <small>score {formatSigned(comparison.deltaOverallScore)} · alive {formatSigned(comparison.deltaLivingCountries)} · shortage {formatSigned(comparison.deltaPeakShortage)} · delivered {formatSigned(comparison.deltaEnergyDelivered)}</small>
              </div>
            ))}
          </div>
        </div>
      )}

      {topScenario && (
        <div className="econ-info-card">
          <div className="econ-info-card-head">
            <strong>Top Scenario Trace</strong>
            <span className="econ-note">{topScenario.spec.label}</span>
          </div>
          <TimeSeriesChart
            title="Top Scenario"
            data={topScenario.trace}
            height={180}
            series={[
              { key: 'totalEnergyDelivered', color: '#0f766e', label: 'Delivered' },
              { key: 'totalEnergyTraded', color: '#0f4c81', label: 'Trade' },
              { key: 'totalShortage', color: '#b45309', label: 'Shortage', type: 'area' },
            ]}
          />
        </div>
      )}
    </>
  )
}

function DecisionPanel({ country }: { country: CountrySnapshot }) {
  return (
    <div className="econ-info-card">
      <div className="econ-info-card-head">
        <strong>Decision</strong>
        <StatusBadge tone={country.lastDecisionSource === 'openrouter' ? 'accent' : 'neutral'}>
          {country.lastDecisionSource || 'none'}
        </StatusBadge>
      </div>
      <div className="econ-decision-body">
        <p>{country.lastDecisionSummary || country.lastBuildFailure || 'No decision summary yet.'}</p>
        <div className="econ-decision-meta">
          <span>{country.lastBuildFocus || 'hold'}</span>
          <span>{country.lastBuildKind || 'none'}</span>
          <span>{country.lastDecisionModel || 'builtin/default'}</span>
        </div>
        <div className="econ-decision-meta">
          <span>prompt {country.lastDecisionUsage.promptTokens}</span>
          <span>completion {country.lastDecisionUsage.completionTokens}</span>
          <span>total {country.lastDecisionUsage.totalTokens}</span>
        </div>
        {country.lastDecisionError && (
          <div className="econ-llm-error">{country.lastDecisionError}</div>
        )}
      </div>
    </div>
  )
}

function MarketDetail({ market }: { market: MarketSnapshot }) {
  return (
    <>
      <div className="econ-detail-head">
        <div>
          <h2>Market Snapshot</h2>
          <p>Para var. Şu anda treasury tutuluyor ve trade ile build harcamaları üzerinden akıyor.</p>
        </div>
        <StatusBadge tone={market.totalShortage > 0 ? 'warn' : 'ok'}>
          shortage {market.totalShortage.toFixed(1)}
        </StatusBadge>
      </div>

      <div className="cp-stats-grid econ-stat-grid econ-detail-stats">
        <StatCard label="Demand" value={market.totalDemand.toFixed(1)} />
        <StatCard label="Generated" value={market.totalGenerated.toFixed(1)} />
        <StatCard label="Delivered" value={market.totalDelivered.toFixed(1)} />
        <StatCard label="Imported" value={market.totalImported.toFixed(1)} />
        <StatCard label="Exported" value={market.totalExported.toFixed(1)} />
        <StatCard label="Treasury" value={market.totalTreasury.toFixed(1)} />
      </div>

      <div className="econ-market-grid">
        <div className="econ-info-card">
          <div className="econ-info-card-head">
            <strong>Prices</strong>
          </div>
          <div className="econ-money-grid">
            <MoneyCard label="Energy" value={market.energyPrice} />
            <MoneyCard label="Emergency" value={market.emergencyEnergyPrice} />
            <MoneyCard label="Coal" value={market.coalPrice} />
            <MoneyCard label="Oil" value={market.oilPrice} />
            <MoneyCard label="Copper" value={market.copperPrice} />
            <MoneyCard label="Silicon" value={market.siliconPrice} />
          </div>
        </div>

        <div className="econ-info-card">
          <div className="econ-info-card-head">
            <strong>Build Costs</strong>
          </div>
          <div className="econ-money-grid">
            <MoneyCard label="Energy Build" value={market.energyBuildTreasury} />
            <MoneyCard label="Compute Build" value={market.computeBuildTreasury} />
            <MoneyCard label="Infra Build" value={market.infrastructureTreasury} />
          </div>
        </div>
      </div>

      <div className="econ-market-country-grid">
        {market.countries
          .slice()
          .sort((a, b) => b.treasury - a.treasury)
          .map(country => (
            <div key={country.id} className="econ-info-card econ-market-country-card">
              <div className="econ-info-card-head">
                <strong>{country.name}</strong>
                <StatusBadge tone={country.treasuryDelta >= 0 ? 'ok' : 'warn'}>
                  {formatSigned(country.treasuryDelta)}
                </StatusBadge>
              </div>
              <div className="econ-asset-list">
                <AssetRow label="Treasury" value={country.treasury} />
                <AssetRow label="Demand" value={country.demand} />
                <AssetRow label="Generated" value={country.generatedEnergy} />
                <AssetRow label="Imported" value={country.importedEnergy} />
                <AssetRow label="Exported" value={country.exportedEnergy} />
                <AssetRow label="Trade Rev" value={country.tradeRevenue} />
                <AssetRow label="Trade Spend" value={country.tradeSpend} />
                <AssetRow label="Industrial Rev" value={country.industrialRevenue} />
                <AssetRow label="Build Spend" value={country.buildSpend} />
                <AssetRow label="Emergency" value={country.emergencySpend} />
                <QuoteInline quotes={country.quotes} />
              </div>
            </div>
          ))}
      </div>
    </>
  )
}

function QuoteInline({ quotes }: { quotes: ResourceQuoteSnapshot[] | null | undefined }) {
  const safeQuotes = quotes ?? []
  return (
    <div className="econ-quote-inline">
      {safeQuotes.map(quote => (
        <span key={quote.resource} className="econ-quote-pill" data-tone={quote.net > 0 ? 'buy' : quote.net < 0 ? 'sell' : 'flat'}>
          {quote.resource} {formatSigned(quote.net)} @ {quote.limitPrice.toFixed(2)}
        </span>
      ))}
    </div>
  )
}

function compareSessionHeadline(session: EconLabSession) {
  if (session.scenarios.length === 0) {
    return 'empty session'
  }
  const alive = session.scenarios.reduce((max, scenario) => Math.max(max, scenario.summary.livingCountries), 0)
  const topTrade = session.scenarios.reduce((max, scenario) => Math.max(max, scenario.summary.cumulativeEnergyTraded), 0)
  return `alive <= ${alive} · top trade ${topTrade.toFixed(1)}`
}

function describeVariableMode(presetCount: number, algorithmCount: number) {
  if (presetCount > 1 && algorithmCount > 1) return 'matrix sweep'
  if (presetCount > 1) return 'resource sweep'
  if (algorithmCount > 1) return 'algorithm sweep'
  return 'single scenario'
}

function rankSessionScenarios(scenarios: EconLabScenario[]) {
  return [...scenarios].sort((a, b) => {
    const scoreA = a.summary.livingCountries * 100000 + a.summary.cumulativeEnergyDelivered * 140 + a.summary.finalTreasury * 4 - a.summary.peakShortage * 150
    const scoreB = b.summary.livingCountries * 100000 + b.summary.cumulativeEnergyDelivered * 140 + b.summary.finalTreasury * 4 - b.summary.peakShortage * 150
    return scoreB - scoreA
  })
}

function LabDetail({ scenario }: { scenario: EconLabScenario }) {
  return (
    <>
      <div className="econ-detail-head">
        <div>
          <h2>{scenario.label}</h2>
          <p>{scenario.presetLabel} resource layout under {scenario.algorithm} market matching.</p>
        </div>
        <StatusBadge tone={scenario.summary.peakShortage > 0 ? 'warn' : 'ok'}>
          tick {scenario.summary.finalTick}
        </StatusBadge>
      </div>

      <div className="cp-stats-grid econ-stat-grid econ-detail-stats">
        <StatCard label="Alive" value={String(scenario.summary.livingCountries)} />
        <StatCard label="Trade" value={scenario.summary.cumulativeEnergyTraded.toFixed(1)} />
        <StatCard label="Cargo" value={scenario.summary.cumulativeCargoTraded.toFixed(1)} />
        <StatCard label="Peak Short" value={scenario.summary.peakShortage.toFixed(1)} />
        <StatCard label="Treasury" value={scenario.summary.finalTreasury.toFixed(1)} />
        <StatCard label="Compute" value={scenario.summary.finalComputeEfficiency.toFixed(2)} />
      </div>

      <div className="econ-market-grid">
        <div className="econ-info-card">
          <div className="econ-info-card-head">
            <strong>Scenario Trends</strong>
          </div>
          <SparkSection label="Delivered Energy">
            <SparkLine data={scenario.statsHistory} series={[{ key: 'totalEnergyDelivered', color: '#0f766e' }]} height={48} />
          </SparkSection>
          <SparkSection label="Shortage">
            <SparkLine data={scenario.statsHistory} series={[{ key: 'totalShortage', color: '#b45309' }]} height={48} />
          </SparkSection>
          <SparkSection label="Treasury">
            <SparkLine data={scenario.statsHistory} series={[{ key: 'totalTreasury', color: '#155e75' }]} height={48} />
          </SparkSection>
        </div>

        <EventFeed title="Recent Events" events={scenario.recentEvents} />
      </div>

      <MarketDetail market={scenario.market} />
    </>
  )
}

function RunDetail({
  run,
  runError,
}: {
  run: EconRunRecord | null
  runError: string | null
}) {
  if (runError) {
    return <div className="econ-warning-card">{runError}</div>
  }

  if (!run) {
    return <div className="econ-map-empty">waiting for run record...</div>
  }

  const latest = run.records[run.records.length - 1]
  const series = run.records.map(record => record.stats)

  return (
    <>
      <div className="econ-detail-head">
        <div>
          <h2>Run Record</h2>
          <p>{run.id} · {run.status} · {formatStamp(run.updatedAt)}</p>
        </div>
        <StatusBadge tone={run.summary.finalShortage > 0 ? 'warn' : 'ok'}>
          {run.recordCount} records
        </StatusBadge>
      </div>

      <div className="cp-stats-grid econ-stat-grid econ-detail-stats">
        <StatCard label="Tick" value={String(run.summary.currentTick)} />
        <StatCard label="Alive" value={String(run.summary.livingCountries)} />
        <StatCard label="Energy" value={run.summary.finalEnergyDelivered.toFixed(1)} />
        <StatCard label="Trade" value={run.summary.finalEnergyTraded.toFixed(1)} />
        <StatCard label="Cargo" value={run.summary.finalCargoTraded.toFixed(1)} />
        <StatCard label="Treasury" value={run.summary.finalTreasury.toFixed(1)} />
      </div>

      <div className="econ-run-meta">
        <div className="econ-info-card">
          <div className="econ-info-card-head">
            <strong>Run Meta</strong>
          </div>
          <div className="econ-asset-list">
            <MetaRow label="Started" value={formatStamp(run.startedAt)} />
            <MetaRow label="Updated" value={formatStamp(run.updatedAt)} />
            <MetaRow label="LLM Mode" value={`${run.llmMode}${run.llmModel ? ` · ${run.llmModel}` : ''}`} />
            <MetaRow label="File" value={run.filePath} mono />
          </div>
        </div>

        <EventFeed title="Last Tick Events" events={latest?.events ?? []} />
      </div>

      <TimeSeriesChart
        title="Run Energy"
        data={series}
        height={176}
        series={[
          { key: 'totalEnergyDelivered', color: '#0f766e', label: 'Delivered' },
          { key: 'totalEnergyTraded', color: '#0f4c81', label: 'Traded' },
          { key: 'totalShortage', color: '#b45309', label: 'Shortage', type: 'area' },
        ]}
      />

      <TimeSeriesChart
        title="Run Treasury"
        data={series}
        height={176}
        series={[
          { key: 'totalTreasury', color: '#155e75', label: 'Treasury' },
          { key: 'avgComputeEfficiency', color: '#0f766e', label: 'Compute' },
          { key: 'livingCountries', color: '#171717', label: 'Alive' },
        ]}
      />
    </>
  )
}

function EventFeed({ title, events }: { title: string; events: string[] }) {
  return (
    <div className="econ-info-card">
      <div className="econ-info-card-head">
        <strong>{title}</strong>
      </div>
      <div className="econ-event-feed">
        {events.length === 0 ? (
          <div className="econ-note">No events yet.</div>
        ) : (
          events.map((event, index) => (
            <div key={`${event}-${index}`} className="econ-event-item">{event}</div>
          ))
        )}
      </div>
    </div>
  )
}

function MoneyCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="econ-money-card">
      <span>{label}</span>
      <strong>{value.toFixed(1)}</strong>
    </div>
  )
}

function AssetRow({ label, value }: { label: string; value: number }) {
  return (
    <div className="econ-asset-row">
      <span>{label}</span>
      <strong>{value.toFixed(1)}</strong>
    </div>
  )
}

function MetaRow({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="econ-asset-row">
      <span>{label}</span>
      <strong className={mono ? 'econ-mono' : undefined}>{value}</strong>
    </div>
  )
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="stat-card">
      <span className="stat-card-value">{value}</span>
      <span className="stat-card-label">{label}</span>
    </div>
  )
}

function SparkSection({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="spark-section">
      <span className="spark-label">{label}</span>
      {children}
    </div>
  )
}

function StatusBadge({ children, tone }: { children: ReactNode; tone: 'ok' | 'warn' | 'neutral' | 'accent' }) {
  return <span className="econ-badge" data-tone={tone}>{children}</span>
}

function MetricChip({ label, value, tone }: { label: string; value: string; tone: 'ok' | 'warn' | 'neutral' | 'accent' }) {
  return (
    <div className="econ-metric-chip" data-tone={tone}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

function initialDeposits(config: EconConfig | null, countryId: string): ResourceStockpile | null {
  return config?.countries.find(country => country.id === countryId)?.deposits ?? null
}

function countryTone(country: CountrySnapshot): 'ok' | 'warn' | 'neutral' | 'accent' {
  if (!country.alive) return 'neutral'
  if (country.lastShortage > 0) return 'warn'
  if (country.computeEfficiency > 0.65) return 'accent'
  return 'ok'
}

function nodeFill(country: CountrySnapshot) {
  if (!country.alive) return '#6b7280'
  if (country.lastShortage > 1) return '#b45309'
  if (country.computeEfficiency > 0.75) return '#0f766e'
  return '#155e75'
}

function formatSigned(value: number) {
  if (value > 0) return `+${value.toFixed(1)}`
  if (value < 0) return value.toFixed(1)
  return '0.0'
}

function formatStamp(value: string) {
  if (!value) return 'n/a'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function toggleSelection(current: string[], next: string) {
  if (current.includes(next)) {
    return current.length === 1 ? current : current.filter(value => value !== next)
  }
  return [...current, next]
}
