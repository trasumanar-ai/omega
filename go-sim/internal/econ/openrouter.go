package econ

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type OpenRouterConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	AppName     string
	SiteURL     string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}

type OpenRouterProvider struct {
	cfg    OpenRouterConfig
	client *http.Client
}

func LoadOpenRouterConfigFromEnv() (OpenRouterConfig, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	if apiKey == "" {
		return OpenRouterConfig{}, fmt.Errorf("OPENROUTER_API_KEY is not set")
	}

	baseURL := strings.TrimSpace(os.Getenv("OPENROUTER_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	model := strings.TrimSpace(os.Getenv("ECON_LLM_MODEL"))
	if model == "" {
		model = "minimax/minimax-m2.5"
	}
	appName := strings.TrimSpace(os.Getenv("ECON_LLM_APP_NAME"))
	if appName == "" {
		appName = "omega-econ"
	}
	siteURL := strings.TrimSpace(os.Getenv("ECON_LLM_SITE_URL"))
	maxTokens := envInt("ECON_LLM_MAX_TOKENS", 80)
	temperature := envFloat("ECON_LLM_TEMPERATURE", 0.1)
	timeout := time.Duration(envInt("ECON_LLM_TIMEOUT_SECONDS", 45)) * time.Second

	return OpenRouterConfig{
		APIKey:      apiKey,
		BaseURL:     strings.TrimRight(baseURL, "/"),
		Model:       model,
		AppName:     appName,
		SiteURL:     siteURL,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Timeout:     timeout,
	}, nil
}

func NewOpenRouterProvider(cfg OpenRouterConfig) *OpenRouterProvider {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	return &OpenRouterProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *OpenRouterProvider) ProviderFunc() DecisionProvider {
	return func(observation CountryObservation) DecisionResult {
		return p.Decide(observation)
	}
}

func (p *OpenRouterProvider) Decide(observation CountryObservation) DecisionResult {
	payload, err := json.Marshal(buildPromptPayload(observation))
	if err != nil {
		return DecisionResult{
			Decision: CountryDecision{BuildFocus: BuildHold},
			Source:   "openrouter",
			Model:    p.cfg.Model,
			Error:    fmt.Sprintf("marshal_prompt_payload: %v", err),
		}
	}

	reqBody := openRouterRequest{
		Model: p.cfg.Model,
		Messages: []openRouterMessage{
			{
				Role: "system",
				Content: strings.TrimSpace(
					"You are the infrastructure planner for one country in a four-country survival economy. " +
						"Goal: keep your country alive as long as possible. " +
						"Choose exactly one build focus: energy, compute, infrastructure, or hold. " +
						"compute only improves energy efficiency and reduces losses; it does not create new agents yet. " +
						"Prefer energy when reserves are low or shortages are present. " +
						"Prefer infrastructure when delivery or trade looks bottlenecked. " +
						"Prefer compute when energy is stable and there is enough copper and silicon. " +
						"Respond with JSON only in this exact shape: " +
						"{\"buildFocus\":\"energy|compute|infrastructure|hold\",\"reason\":\"short sentence\"}",
				),
			},
			{
				Role:    "user",
				Content: string(payload),
			},
		},
		Temperature: p.cfg.Temperature,
		MaxTokens:   p.cfg.MaxTokens,
		ResponseFormat: &openRouterResponseFormat{
			Type: "json_object",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return DecisionResult{
			Decision: CountryDecision{BuildFocus: BuildHold},
			Source:   "openrouter",
			Model:    p.cfg.Model,
			Error:    fmt.Sprintf("marshal_request: %v", err),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return DecisionResult{
			Decision: CountryDecision{BuildFocus: BuildHold},
			Source:   "openrouter",
			Model:    p.cfg.Model,
			Error:    fmt.Sprintf("new_request: %v", err),
		}
	}
	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if p.cfg.SiteURL != "" {
		req.Header.Set("HTTP-Referer", p.cfg.SiteURL)
	}
	if p.cfg.AppName != "" {
		req.Header.Set("X-Title", p.cfg.AppName)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return DecisionResult{
			Decision: CountryDecision{BuildFocus: BuildHold},
			Source:   "openrouter",
			Model:    p.cfg.Model,
			Error:    fmt.Sprintf("request_failed: %v", err),
		}
	}
	defer resp.Body.Close()

	rawResp, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return DecisionResult{
			Decision: CountryDecision{BuildFocus: BuildHold},
			Source:   "openrouter",
			Model:    p.cfg.Model,
			Error:    fmt.Sprintf("read_response: %v", err),
		}
	}

	if resp.StatusCode >= 300 {
		return DecisionResult{
			Decision: CountryDecision{BuildFocus: BuildHold},
			Source:   "openrouter",
			Model:    p.cfg.Model,
			Error:    fmt.Sprintf("http_%d: %s", resp.StatusCode, strings.TrimSpace(string(rawResp))),
		}
	}

	var parsed openRouterResponse
	if err := json.Unmarshal(rawResp, &parsed); err != nil {
		return DecisionResult{
			Decision: CountryDecision{BuildFocus: BuildHold},
			Source:   "openrouter",
			Model:    p.cfg.Model,
			Error:    fmt.Sprintf("decode_response: %v", err),
		}
	}
	if len(parsed.Choices) == 0 {
		return DecisionResult{
			Decision: CountryDecision{BuildFocus: BuildHold},
			Source:   "openrouter",
			Model:    p.cfg.Model,
			Error:    "no_choices_in_response",
		}
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	rawJSON := extractJSONObject(content)
	if rawJSON == "" {
		rawJSON = content
	}

	var decision struct {
		BuildFocus BuildFocus `json:"buildFocus"`
		Reason     string     `json:"reason"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &decision); err != nil {
		return DecisionResult{
			Decision: CountryDecision{BuildFocus: BuildHold},
			Source:   "openrouter",
			Model:    p.cfg.Model,
			Error:    fmt.Sprintf("parse_decision: %v | content=%s", err, truncate(content, 180)),
			Usage: DecisionUsage{
				PromptTokens:     parsed.Usage.PromptTokens,
				CompletionTokens: parsed.Usage.CompletionTokens,
				TotalTokens:      parsed.Usage.TotalTokens,
			},
		}
	}

	return DecisionResult{
		Decision: CountryDecision{BuildFocus: decision.BuildFocus},
		Source:   "openrouter",
		Model:    p.cfg.Model,
		Summary:  strings.TrimSpace(decision.Reason),
		Usage: DecisionUsage{
			PromptTokens:     parsed.Usage.PromptTokens,
			CompletionTokens: parsed.Usage.CompletionTokens,
			TotalTokens:      parsed.Usage.TotalTokens,
		},
	}
}

type openRouterRequest struct {
	Model          string                    `json:"model"`
	Messages       []openRouterMessage       `json:"messages"`
	Temperature    float64                   `json:"temperature,omitempty"`
	MaxTokens      int                       `json:"max_tokens,omitempty"`
	ResponseFormat *openRouterResponseFormat `json:"response_format,omitempty"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponseFormat struct {
	Type string `json:"type"`
}

type openRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func buildPromptPayload(observation CountryObservation) map[string]any {
	country := observation.Country
	neighbors := make([]map[string]any, 0, len(observation.Neighbors))
	for _, neighbor := range observation.Neighbors {
		neighbors = append(neighbors, map[string]any{
			"id":                neighbor.ID,
			"stability":         round2(neighbor.Stability),
			"energyReserve":     round2(neighbor.EnergyReserve),
			"computeEfficiency": round2(neighbor.ComputeEfficiency),
			"infrastructure":    round2(neighbor.Infrastructure),
		})
	}

	return map[string]any{
		"tick": observation.Tick,
		"country": map[string]any{
			"id":                country.ID,
			"name":              country.Name,
			"stability":         round2(country.Stability),
			"treasury":          round2(country.Treasury),
			"energyReserve":     round2(country.EnergyReserve),
			"reserveCapacity":   round2(country.ReserveCapacity),
			"gridCapacity":      round2(country.GridCapacity),
			"computeEfficiency": round2(country.ComputeEfficiency),
			"baseDemand":        round2(country.BaseDemand),
			"deposits": map[string]any{
				"coal":    round2(country.Deposits.Coal),
				"oil":     round2(country.Deposits.Oil),
				"copper":  round2(country.Deposits.Copper),
				"silicon": round2(country.Deposits.Silicon),
			},
			"stockpiles": map[string]any{
				"coal":    round2(country.Stockpiles.Coal),
				"oil":     round2(country.Stockpiles.Oil),
				"copper":  round2(country.Stockpiles.Copper),
				"silicon": round2(country.Stockpiles.Silicon),
			},
			"assets": map[string]any{
				"coalPlant": round2(country.Assets.CoalPlant),
				"oilPlant":  round2(country.Assets.OilPlant),
				"solarFarm": round2(country.Assets.SolarFarm),
				"windFarm":  round2(country.Assets.WindFarm),
			},
			"lastGeneratedEnergy": round2(country.LastGeneratedEnergy),
			"lastDeliveredEnergy": round2(country.LastDeliveredEnergy),
			"lastImportedEnergy":  round2(country.LastImportedEnergy),
			"lastExportedEnergy":  round2(country.LastExportedEnergy),
			"lastShortage":        round2(country.LastShortage),
			"lastBuildFocus":      country.LastBuildFocus,
			"lastBuildSuccess":    country.LastBuildSuccess,
		},
		"neighbors": neighbors,
		"world": map[string]any{
			"livingCountries":      observation.WorldTotals.LivingCountries,
			"totalEnergyDelivered": round2(observation.WorldTotals.TotalEnergyDelivered),
			"totalEnergyTraded":    round2(observation.WorldTotals.TotalEnergyTraded),
			"totalCargoTraded":     round2(observation.WorldTotals.TotalCargoTraded),
			"totalShortage":        round2(observation.WorldTotals.TotalShortage),
			"avgComputeEfficiency": round2(observation.WorldTotals.AvgComputeEfficiency),
		},
	}
}

func extractJSONObject(input string) string {
	start := strings.IndexByte(input, '{')
	end := strings.LastIndexByte(input, '}')
	if start == -1 || end == -1 || end < start {
		return ""
	}
	return input[start : end+1]
}

func truncate(input string, max int) string {
	if len(input) <= max {
		return input
	}
	return input[:max]
}

func round2(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil {
		return fallback
	}
	return parsed
}

func envFloat(name string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	var parsed float64
	if _, err := fmt.Sscanf(value, "%f", &parsed); err != nil {
		return fallback
	}
	return parsed
}
