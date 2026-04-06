package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type tool struct {
	Type     string     `json:"type"`
	Function toolSchema `json:"function"`
}

type toolSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

func main() {
	agentDir := flag.String("agent", "agents/president", "path to agent workspace")
	apiKey := flag.String("api-key", "", "OpenRouter API key")
	model := flag.String("model", "minimax/minimax-m2.7", "LLM model")
	govURL := flag.String("gov-url", "http://localhost:8090", "government API URL")
	agentKey := flag.String("agent-key", "", "agent API key (from .env)")
	govID := flag.String("gov-id", "", "government ID (from .env)")
	flag.Parse()

	// Load .env first, then resolve all values
	loadDotEnv(*agentDir)

	if *apiKey == "" {
		*apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if *agentKey == "" {
		*agentKey = os.Getenv("OMEGA_API_KEY")
	}
	if *govID == "" {
		*govID = os.Getenv("OMEGA_GOV_ID")
	}

	if *apiKey == "" {
		fmt.Fprintln(os.Stderr, "error: set OPENROUTER_API_KEY or use -api-key")
		os.Exit(1)
	}
	if *agentKey == "" || *govID == "" {
		fmt.Fprintln(os.Stderr, "error: set -agent-key and -gov-id, or have .env with PRESIDENT_API_KEY and OMEGA_GOV_ID")
		os.Exit(1)
	}

	// Load persona
	soul := readFile(*agentDir + "/SOUL.md")
	agents := readFile(*agentDir + "/AGENTS.md")
	identity := readFile(*agentDir + "/IDENTITY.md")

	systemPrompt := buildSystemPrompt(soul, agents, identity, *govID)

	// Build tool executor
	exec := &toolExecutor{govURL: *govURL, govID: *govID, apiKey: *agentKey}

	history := []message{{Role: "system", Content: systemPrompt}}

	// Print agent info
	name := "Agent"
	for _, line := range strings.Split(identity, "\n") {
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
	}
	fmt.Printf("\033[1m%s\033[0m is ready. Type your message. (ctrl+d to quit)\n\n", name)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\033[36myou:\033[0m ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		history = append(history, message{Role: "user", Content: input})

		// Tool loop: keep calling LLM until it responds without tool calls
		for {
			resp, err := callLLM(*apiKey, *model, history, govTools())
			if err != nil {
				fmt.Printf("\033[31merror: %v\033[0m\n", err)
				break
			}

			if len(resp.ToolCalls) > 0 {
				// Show tool usage
				history = append(history, *resp)
				for _, tc := range resp.ToolCalls {
					fmt.Printf("\033[33m  [%s]\033[0m", tc.Function.Name)
					result := exec.execute(tc.Function.Name, tc.Function.Arguments)
					fmt.Printf(" \033[2m%s\033[0m\n", truncate(result, 80))
					history = append(history, message{
						Role:       "tool",
						ToolCallID: tc.ID,
						Name:       tc.Function.Name,
						Content:    result,
					})
				}
				continue // let LLM process tool results
			}

			// Final text response
			if resp.Content != "" {
				fmt.Printf("\n\033[1m%s:\033[0m %s\n\n", name, resp.Content)
			}
			history = append(history, *resp)
			break
		}
	}
	fmt.Println()
}

func buildSystemPrompt(soul, agents, identity, govID string) string {
	var b strings.Builder
	if soul != "" {
		b.WriteString(soul)
		b.WriteString("\n\n")
	}
	if agents != "" {
		b.WriteString(agents)
		b.WriteString("\n\n")
	}
	b.WriteString("# Context\n\n")
	b.WriteString(fmt.Sprintf("You are in a live conversation. Your government ID is %s.\n", govID))
	b.WriteString("You have tools to interact with the government API. Use them to check state before making decisions.\n")
	b.WriteString("Keep responses conversational and concise. You are speaking directly to a citizen or advisor.\n")
	b.WriteString(fmt.Sprintf("Current time: %s\n", time.Now().Format("2006-01-02 15:04")))
	return b.String()
}

func govTools() []tool {
	return []tool{
		mkTool("check_balance", "Check your current bank balance", nil),
		mkTool("list_citizens", "List all citizens in the government", nil),
		mkTool("economic_stats", "Get money supply, Gini coefficient, citizen count", nil),
		mkTool("view_ledger", "View recent transactions", nil),
		mkTool("transfer_money", "Transfer money to another citizen", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"to_id":  map[string]any{"type": "string", "description": "recipient agent ID"},
				"amount": map[string]any{"type": "number", "description": "amount to transfer"},
			},
			"required": []string{"to_id", "amount"},
		}),
		mkTool("list_contracts", "List your contracts", nil),
		mkTool("create_contract", "Create a payment contract with another citizen", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"payee_id":    map[string]any{"type": "string", "description": "agent ID of the payee"},
				"amount":      map[string]any{"type": "number", "description": "payment amount"},
				"description": map[string]any{"type": "string", "description": "what the contract is for"},
			},
			"required": []string{"payee_id", "amount", "description"},
		}),
		mkTool("list_firms", "List all firms/organizations", nil),
		mkTool("create_firm", "Create a new firm/organization", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string", "description": "firm name"},
			},
			"required": []string{"name"},
		}),
		mkTool("get_government_info", "Get government details and configuration", nil),
	}
}

func mkTool(name, desc string, params any) tool {
	if params == nil {
		params = map[string]any{"type": "object", "properties": map[string]any{}}
	}
	return tool{Type: "function", Function: toolSchema{Name: name, Description: desc, Parameters: params}}
}

// --- Tool executor ---

type toolExecutor struct {
	govURL string
	govID  string
	apiKey string
}

func (e *toolExecutor) execute(name, argsJSON string) string {
	var args map[string]any
	json.Unmarshal([]byte(argsJSON), &args)

	switch name {
	case "check_balance":
		return e.get("/bank/balance")
	case "list_citizens":
		return e.getPublic("/registry/agents")
	case "economic_stats":
		return e.getPublic("/bank/supply")
	case "view_ledger":
		return e.getPublic("/bank/ledger")
	case "transfer_money":
		return e.post("/bank/transfer", args)
	case "list_contracts":
		return e.get("/contracts")
	case "create_contract":
		body := map[string]any{
			"type":    "payment",
			"my_role": "payer",
			"terms": map[string]any{
				"amount":      args["amount"],
				"description": args["description"],
			},
			"parties": []map[string]any{
				{"agent_id": args["payee_id"], "role": "payee"},
			},
		}
		return e.post("/contracts", body)
	case "list_firms":
		return e.getPublic("/registry/firms")
	case "create_firm":
		return e.post("/registry/firms", args)
	case "get_government_info":
		return e.getNoAuth(fmt.Sprintf("%s/api/governments/%s", e.govURL, e.govID))
	default:
		return `{"error":"unknown tool"}`
	}
}

func (e *toolExecutor) get(path string) string {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/governments/%s%s", e.govURL, e.govID, path), nil)
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	return doHTTP(req)
}

func (e *toolExecutor) getPublic(path string) string {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/governments/%s%s", e.govURL, e.govID, path), nil)
	return doHTTP(req)
}

func (e *toolExecutor) getNoAuth(url string) string {
	req, _ := http.NewRequest("GET", url, nil)
	return doHTTP(req)
}

func (e *toolExecutor) post(path string, body any) string {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/governments/%s%s", e.govURL, e.govID, path), bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return doHTTP(req)
}

func doHTTP(req *http.Request) string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// --- LLM ---

func callLLM(apiKey, model string, messages []message, tools []tool) (*message, error) {
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"tools":    tools,
	}
	data, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://omega.gov")
	req.Header.Set("X-Title", "Omega Government")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message message `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &result.Choices[0].Message, nil
}

// --- Helpers ---

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func loadDotEnv(agentDir string) {
	// Try .env in current dir, then relative to agent dir
	paths := []string{".env", agentDir + "/../../.env"}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if k, v, ok := strings.Cut(line, "="); ok {
				os.Setenv(strings.TrimSpace(k), strings.TrimSpace(v))
			}
		}
		break
	}

	// Map agent-specific key based on agent dir name
	dir := strings.ToLower(agentDir)
	keyMap := map[string]string{
		"president": "PRESIDENT_API_KEY",
		"senator-1": "SENATOR1_API_KEY",
		"senator-2": "SENATOR2_API_KEY",
		"senator-3": "SENATOR3_API_KEY",
		"fed-chair": "FEDCHAIR_API_KEY",
	}
	for pattern, envKey := range keyMap {
		if strings.Contains(dir, pattern) {
			if v := os.Getenv(envKey); v != "" {
				os.Setenv("OMEGA_API_KEY", v)
			}
			break
		}
	}
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
