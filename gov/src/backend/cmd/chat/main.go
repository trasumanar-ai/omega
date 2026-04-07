package main

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	model := flag.String("model", "stepfun/step-3.5-flash", "LLM model")
	govURL := flag.String("gov-url", "http://localhost:8090", "government API URL")
	govID := flag.String("gov-id", "", "government ID")
	flag.Parse()

	// Load .env
	loadDotEnv(*agentDir)
	if *apiKey == "" {
		*apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if *govID == "" {
		*govID = os.Getenv("OMEGA_GOV_ID")
	}

	if *apiKey == "" {
		fmt.Fprintln(os.Stderr, "error: set OPENROUTER_API_KEY or use -api-key")
		os.Exit(1)
	}
	if *govID == "" {
		fmt.Fprintln(os.Stderr, "error: set -gov-id or OMEGA_GOV_ID in .env")
		os.Exit(1)
	}

	// Load or generate key pair
	keyDir := filepath.Join(*agentDir, "keys")
	privKey, pubHex := loadOrGenerateKeys(keyDir)

	// Register if needed
	agentID := ensureRegistered(*govURL, *govID, *agentDir, pubHex)

	// Load persona
	soul := readFile(*agentDir + "/SOUL.md")
	agents := readFile(*agentDir + "/AGENTS.md")
	identity := readFile(*agentDir + "/IDENTITY.md")
	systemPrompt := buildSystemPrompt(soul, agents, identity, *govID, agentID)

	exec := &toolExecutor{govURL: *govURL, govID: *govID, privKey: privKey, pubHex: pubHex}
	history := []message{{Role: "system", Content: systemPrompt}}

	name := "Agent"
	for _, line := range strings.Split(identity, "\n") {
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
	}
	fmt.Printf("\033[1m%s\033[0m is ready. (id: %s)\nType your message. ctrl+d to quit.\n\n", name, agentID[:12])

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

		for {
			resp, err := callLLM(*apiKey, *model, history, govTools())
			if err != nil {
				fmt.Printf("\033[31merror: %v\033[0m\n", err)
				break
			}

			if len(resp.ToolCalls) > 0 {
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
				continue
			}

			if resp.Content != "" {
				fmt.Printf("\n\033[1m%s:\033[0m %s\n\n", name, resp.Content)
			}
			history = append(history, *resp)
			break
		}
	}
	fmt.Println()
}

// --- Key management ---

func loadOrGenerateKeys(dir string) (ed25519.PrivateKey, string) {
	privPath := filepath.Join(dir, "omega.key")
	pubPath := filepath.Join(dir, "omega.pub")

	// Try loading existing keys
	if privData, err := os.ReadFile(privPath); err == nil {
		if pubData, err := os.ReadFile(pubPath); err == nil {
			privBytes, _ := hex.DecodeString(strings.TrimSpace(string(privData)))
			pubHex := strings.TrimSpace(string(pubData))
			if len(privBytes) == ed25519.PrivateKeySize {
				return ed25519.PrivateKey(privBytes), pubHex
			}
		}
	}

	// Generate new key pair
	os.MkdirAll(dir, 0700)
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating keys: %v\n", err)
		os.Exit(1)
	}

	os.WriteFile(privPath, []byte(hex.EncodeToString(priv)), 0600)
	pubHex := hex.EncodeToString(pub)
	os.WriteFile(pubPath, []byte(pubHex), 0644)

	fmt.Printf("Generated key pair in %s/\n", dir)
	return priv, pubHex
}

func ensureRegistered(govURL, govID, agentDir, pubHex string) string {
	// Check if already registered
	resp, err := http.Get(fmt.Sprintf("%s/api/governments/%s/registry/agents", govURL, govID))
	if err == nil {
		defer resp.Body.Close()
		var agents []struct {
			ID        string `json:"id"`
			PublicKey string `json:"public_key"`
		}
		json.NewDecoder(resp.Body).Decode(&agents)
		for _, a := range agents {
			if a.PublicKey == pubHex {
				return a.ID
			}
		}
	}

	// Not registered — register now
	identity := readFile(agentDir + "/IDENTITY.md")
	name := "Agent"
	for _, line := range strings.Split(identity, "\n") {
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
	}

	soul := readFile(agentDir + "/SOUL.md")
	body, _ := json.Marshal(map[string]string{
		"name":       name,
		"public_key": pubHex,
		"soul_md":    soul,
		"model":      "minimax/minimax-m2.7",
	})

	regResp, err := http.Post(
		fmt.Sprintf("%s/api/governments/%s/registry/register", govURL, govID),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error registering: %v\n", err)
		os.Exit(1)
	}
	defer regResp.Body.Close()

	var result struct {
		ID    string `json:"id"`
		Error string `json:"error"`
	}
	json.NewDecoder(regResp.Body).Decode(&result)
	if result.Error != "" {
		fmt.Fprintf(os.Stderr, "registration error: %s\n", result.Error)
		os.Exit(1)
	}

	fmt.Printf("Registered as %s (id: %s)\n", name, result.ID)
	return result.ID
}

// --- Auth ---

func signRequest(privKey ed25519.PrivateKey, pubHex, method, path string) string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	message := ts + ":" + strings.ToUpper(method) + ":" + path
	sig := ed25519.Sign(privKey, []byte(message))
	return fmt.Sprintf("Signed %s:%s:%s", pubHex, ts, hex.EncodeToString(sig))
}

// --- System prompt ---

func buildSystemPrompt(soul, agents, identity, govID, agentID string) string {
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
	b.WriteString(fmt.Sprintf("Your agent ID is: %s\n", agentID))
	b.WriteString(fmt.Sprintf("Your government ID is: %s\n", govID))
	b.WriteString("You have tools to interact with the government API. Use them to check state before making decisions.\n")
	b.WriteString("Keep responses conversational and concise.\n")
	b.WriteString(fmt.Sprintf("Current time: %s\n", time.Now().Format("2006-01-02 15:04")))
	return b.String()
}

// --- Tools ---

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
	govURL  string
	govID   string
	privKey ed25519.PrivateKey
	pubHex  string
}

func (e *toolExecutor) execute(name, argsJSON string) string {
	var args map[string]any
	json.Unmarshal([]byte(argsJSON), &args)

	switch name {
	case "check_balance":
		return e.authedGet("/bank/balance")
	case "list_citizens":
		return e.publicGet("/registry/agents")
	case "economic_stats":
		return e.publicGet("/bank/supply")
	case "view_ledger":
		return e.publicGet("/bank/ledger")
	case "transfer_money":
		return e.authedPost("/bank/transfer", args)
	case "list_contracts":
		return e.authedGet("/contracts")
	case "create_contract":
		body := map[string]any{
			"type":    "payment",
			"my_role": "payer",
			"terms":   map[string]any{"amount": args["amount"], "description": args["description"]},
			"parties": []map[string]any{{"agent_id": args["payee_id"], "role": "payee"}},
		}
		return e.authedPost("/contracts", body)
	case "list_firms":
		return e.publicGet("/registry/firms")
	case "create_firm":
		return e.authedPost("/registry/firms", args)
	case "get_government_info":
		return e.rawGet(fmt.Sprintf("%s/api/governments/%s", e.govURL, e.govID))
	default:
		return `{"error":"unknown tool"}`
	}
}

func (e *toolExecutor) authedGet(path string) string {
	fullPath := fmt.Sprintf("/api/governments/%s%s", e.govID, path)
	req, _ := http.NewRequest("GET", e.govURL+fullPath, nil)
	req.Header.Set("Authorization", signRequest(e.privKey, e.pubHex, "GET", fullPath))
	return doHTTP(req)
}

func (e *toolExecutor) authedPost(path string, body any) string {
	fullPath := fmt.Sprintf("/api/governments/%s%s", e.govID, path)
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", e.govURL+fullPath, bytes.NewReader(data))
	req.Header.Set("Authorization", signRequest(e.privKey, e.pubHex, "POST", fullPath))
	req.Header.Set("Content-Type", "application/json")
	return doHTTP(req)
}

func (e *toolExecutor) publicGet(path string) string {
	return e.rawGet(fmt.Sprintf("%s/api/governments/%s%s", e.govURL, e.govID, path))
}

func (e *toolExecutor) rawGet(url string) string {
	req, _ := http.NewRequest("GET", url, nil)
	return doHTTP(req)
}

func doHTTP(req *http.Request) string {
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// --- LLM ---

func callLLM(apiKey, model string, messages []message, tools []tool) (*message, error) {
	data, _ := json.Marshal(map[string]any{"model": model, "messages": messages, "tools": tools})
	req, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://omega.gov")
	req.Header.Set("X-Title", "Omega Government")

	resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct{ Message message `json:"message"` } `json:"choices"`
		Error   *struct{ Message string `json:"message"` }  `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("%s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response")
	}
	return &result.Choices[0].Message, nil
}

// --- Helpers ---

func readFile(path string) string {
	data, _ := os.ReadFile(path)
	return string(data)
}

func loadDotEnv(agentDir string) {
	for _, p := range []string{".env", agentDir + "/../../.env"} {
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
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
