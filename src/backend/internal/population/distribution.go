// Package population provides real-world distributions of OpenClaw agent
// configurations, sourced from mergisi/awesome-openclaw-agents (196 agents),
// souls.directory (4,631 templates), and OpenRouter usage data.
package population

import (
	"math/rand"
)

// AgentCategory represents a type of agent with its real-world frequency.
type AgentCategory struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"` // relative frequency (sums to 1.0)
	Soul   string  `json:"soul"`   // template SOUL.md content
}

// ModelChoice represents an LLM model with its usage frequency.
type ModelChoice struct {
	ID     string  `json:"id"`     // OpenRouter model ID
	Weight float64 `json:"weight"` // relative frequency (sums to 1.0)
}

// CommunicationStyle represents a personality trait with its frequency.
type CommunicationStyle struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"` // fraction of agents with this trait
}

// --- Real data from awesome-openclaw-agents (196 SOUL.md files) ---

var Categories = []AgentCategory{
	{"marketing", 0.143, soulMarketing},
	{"development", 0.092, soulDevelopment},
	{"business", 0.071, soulBusiness},
	{"creative", 0.066, soulCreative},
	{"devops", 0.051, soulDevOps},
	{"finance", 0.051, soulFinance},
	{"data", 0.046, soulData},
	{"productivity", 0.046, soulProductivity},
	{"education", 0.041, soulEducation},
	{"hr", 0.041, soulHR},
	{"ecommerce", 0.036, soulEcommerce},
	{"healthcare", 0.036, soulHealthcare},
	{"personal", 0.036, soulPersonal},
	{"automation", 0.031, soulAutomation},
	{"legal", 0.031, soulLegal},
	{"saas", 0.031, soulSaaS},
	{"security", 0.031, soulSecurity},
	{"real-estate", 0.026, soulRealEstate},
	{"compliance", 0.020, soulCompliance},
	{"freelance", 0.020, soulFreelance},
	{"supply-chain", 0.015, soulSupplyChain},
	{"customer-success", 0.010, soulCustomerSuccess},
}

// --- Real data from OpenRouter weekly rankings + OpenClaw community ---

var Models = []ModelChoice{
	// Top models by actual usage on OpenRouter (normalized to OpenClaw-relevant subset)
	{"anthropic/claude-sonnet-4-5", 0.25},         // Most popular for agents
	{"google/gemini-2.5-flash", 0.18},              // Fast, cheap, popular
	{"deepseek/deepseek-chat-v3-0324", 0.15},       // Very popular, good value
	{"openai/gpt-4o", 0.12},                        // Widely used
	{"anthropic/claude-haiku-4-5-20251001", 0.10},   // Fast + cheap Anthropic
	{"google/gemini-2.0-flash-001", 0.08},           // Budget option
	{"stepfun/step-3.5-flash", 0.05},                // Emerging
	{"minimax/minimax-m2.7", 0.04},                  // Niche but capable
	{"meta-llama/llama-3.1-70b-instruct", 0.03},     // Open-weight
}

// --- Real data from SOUL.md personality analysis (196 files) ---

var Styles = []CommunicationStyle{
	{"concise", 0.567},        // 111/196 — most common
	{"analytical", 0.357},     // 70/196
	{"professional", 0.270},   // 53/196
	{"thorough", 0.260},       // 51/196
	{"friendly", 0.240},       // 47/196
	{"structured", 0.230},     // 45/196
	{"empathetic", 0.153},     // 30/196
	{"creative", 0.071},       // 14/196
}

// --- Sampling functions ---

// SampleCategory returns a random category weighted by real-world frequency.
func SampleCategory(rng *rand.Rand) AgentCategory {
	return weightedSample(rng, Categories)
}

// SampleModel returns a random model weighted by real-world usage.
func SampleModel(rng *rand.Rand) ModelChoice {
	return weightedSample(rng, Models)
}

// SampleStyles returns a set of personality traits for an agent,
// where each trait is included with its real-world probability.
func SampleStyles(rng *rand.Rand) []string {
	var styles []string
	for _, s := range Styles {
		if rng.Float64() < s.Weight {
			styles = append(styles, s.Name)
		}
	}
	if len(styles) == 0 {
		styles = append(styles, "concise") // fallback
	}
	return styles
}

func weightedSample[T any](rng *rand.Rand, items []T, weights ...float64) T {
	// Use Weight field via interface
	type weighted interface{ getWeight() float64 }
	// Direct implementation for our types
	var totalWeight float64
	switch v := any(items).(type) {
	case []AgentCategory:
		for _, item := range v {
			totalWeight += item.Weight
		}
		r := rng.Float64() * totalWeight
		for _, item := range v {
			r -= item.Weight
			if r <= 0 {
				return any(item).(T)
			}
		}
		return any(v[len(v)-1]).(T)
	case []ModelChoice:
		for _, item := range v {
			totalWeight += item.Weight
		}
		r := rng.Float64() * totalWeight
		for _, item := range v {
			r -= item.Weight
			if r <= 0 {
				return any(item).(T)
			}
		}
		return any(v[len(v)-1]).(T)
	default:
		return items[rng.Intn(len(items))]
	}
}
