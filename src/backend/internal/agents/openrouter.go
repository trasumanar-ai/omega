package agents

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

	"omega/backend/internal/world"
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
		model = "deepseek/deepseek-chat-v3-0324"
	}
	appName := strings.TrimSpace(os.Getenv("ECON_LLM_APP_NAME"))
	if appName == "" {
		appName = "omega-econ"
	}
	siteURL := strings.TrimSpace(os.Getenv("ECON_LLM_SITE_URL"))
	maxTokens := envInt("ECON_LLM_MAX_TOKENS", 500)
	temperature := envFloat("ECON_LLM_TEMPERATURE", 0.3)
	timeout := time.Duration(envInt("ECON_LLM_TIMEOUT_SECONDS", 60)) * time.Second

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
		timeout = 60 * time.Second
	}
	return &OpenRouterProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

func (p *OpenRouterProvider) ProviderFunc() world.DecisionProvider {
	return func(obs world.CountryObservation) world.DecisionResult {
		return p.Decide(obs)
	}
}

func (p *OpenRouterProvider) Decide(obs world.CountryObservation) world.DecisionResult {
	fail := func(msg string) world.DecisionResult {
		return world.DecisionResult{
			Actions: []world.AgentAction{{Action: world.ActionHold}},
			Source:  "openrouter",
			Model:   p.cfg.Model,
			Error:   msg,
		}
	}

	payload, err := json.Marshal(obs)
	if err != nil {
		return fail(fmt.Sprintf("marshal: %v", err))
	}

	reqBody := orRequest{
		Model: p.cfg.Model,
		Messages: []orMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: string(payload)},
		},
		Temperature:    p.cfg.Temperature,
		MaxTokens:      p.cfg.MaxTokens,
		ResponseFormat: &orFormat{Type: "json_object"},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fail(fmt.Sprintf("marshal_req: %v", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fail(fmt.Sprintf("new_req: %v", err))
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
		return fail(fmt.Sprintf("request: %v", err))
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fail(fmt.Sprintf("read: %v", err))
	}
	if resp.StatusCode >= 300 {
		return fail(fmt.Sprintf("http_%d: %s", resp.StatusCode, strings.TrimSpace(string(raw))))
	}

	var parsed orResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fail(fmt.Sprintf("decode: %v", err))
	}
	if len(parsed.Choices) == 0 {
		return fail("no_choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	usage := world.DecisionUsage{
		PromptTokens:     parsed.Usage.PromptTokens,
		CompletionTokens: parsed.Usage.CompletionTokens,
		TotalTokens:      parsed.Usage.TotalTokens,
	}

	actions, reason, err := parseResponse(content)
	if err != nil {
		return world.DecisionResult{
			Actions: []world.AgentAction{{Action: world.ActionHold}},
			Source:  "openrouter", Model: p.cfg.Model,
			Error: fmt.Sprintf("parse: %v | %s", err, truncate(content, 200)),
			Usage: usage, RawText: content,
		}
	}

	return world.DecisionResult{
		Actions: actions,
		Source:  "openrouter", Model: p.cfg.Model,
		Summary: reason, Usage: usage, RawText: content,
	}
}

func parseResponse(content string) ([]world.AgentAction, string, error) {
	raw := extractJSON(content)
	if raw == "" {
		raw = content
	}

	var resp struct {
		Actions []world.AgentAction `json:"actions"`
		Reason  string              `json:"reason"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, "", err
	}
	if len(resp.Actions) == 0 {
		return []world.AgentAction{{Action: world.ActionHold}}, resp.Reason, nil
	}

	valid := make([]world.AgentAction, 0, len(resp.Actions))
	for _, a := range resp.Actions {
		switch a.Action {
		case world.ActionPropose, world.ActionAccept, world.ActionBuild, world.ActionBroadcast, world.ActionSend, world.ActionHold:
			valid = append(valid, a)
		}
	}
	if len(valid) == 0 {
		return []world.AgentAction{{Action: world.ActionHold}}, resp.Reason, nil
	}
	return valid, resp.Reason, nil
}

const systemPrompt = `You manage a country in a survival economy. Goal: stay alive as long as possible.

You die when stability hits 0. Stability drops when you can't produce enough energy.
Energy comes from burning coal/oil in plants and from solar/wind farms.
Each tick you extract resources from deposits (finite). You must trade with others for what you lack.

Available actions (return as many as needed per tick):

1. propose — Offer a direct trade to another country
   {"action":"propose","to":"country_id","offerResource":"coal","offerAmount":5,"wantResource":"silicon","wantAmount":3}
   Resources: coal, oil, copper, silicon, money

2. accept — Accept a pending trade proposal (see proposals in your observation)
   {"action":"accept","proposalId":123}

3. build — Build infrastructure (costs copper + silicon from your stockpiles)
   {"action":"build","build":"coal_plant|oil_plant|solar_farm|wind_farm|compute_hub"}

4. broadcast — Send a message to ALL countries
   {"action":"broadcast","message":"I need silicon, willing to trade coal"}

5. send — Send a private message to one country
   {"action":"send","to":"country_id","message":"your message"}

6. hold — Do nothing
   {"action":"hold"}

Respond with JSON: {"actions":[...], "reason":"brief explanation"}

Tips:
- Check pending proposals in your observation — accept good deals
- Propose trades for resources you need. Be specific about amounts.
- Build solar/wind farms to reduce dependence on finite coal/oil
- Building costs copper + silicon, so trade for those if you don't have enough
- Money exists as a tradeable resource but has no inherent value — it's worth what others will trade for it
- Communicate with others to find trade partners`

// --- HTTP types ---

type orRequest struct {
	Model          string      `json:"model"`
	Messages       []orMessage `json:"messages"`
	Temperature    float64     `json:"temperature,omitempty"`
	MaxTokens      int         `json:"max_tokens,omitempty"`
	ResponseFormat *orFormat   `json:"response_format,omitempty"`
}

type orMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type orFormat struct {
	Type string `json:"type"`
}

type orResponse struct {
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

// moved round2, clamp to simulation.go

func extractJSON(s string) string {
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start == -1 || end == -1 || end < start {
		return ""
	}
	return s[start : end+1]
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func envInt(name string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return fallback
	}
	return n
}

func envFloat(name string, fallback float64) float64 {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	var f float64
	if _, err := fmt.Sscanf(v, "%f", &f); err != nil {
		return fallback
	}
	return f
}
