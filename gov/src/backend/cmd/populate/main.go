// Populate generates a realistic population of agents and registers them
// with a government, sampling from real-world OpenClaw distributions.
package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"net/http"
	"omega/backend/internal/population"
	"strings"
	"time"
)

func main() {
	govURL := flag.String("url", "http://localhost:8090", "government API URL")
	govID := flag.String("gov", "", "government ID")
	count := flag.Int("n", 50, "number of agents to generate")
	seed := flag.Int64("seed", 0, "random seed (0 = random)")
	dryRun := flag.Bool("dry-run", false, "print distribution without registering")
	flag.Parse()

	if *govID == "" && !*dryRun {
		log.Fatal("--gov required (government ID)")
	}

	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}
	rng := mrand.New(mrand.NewSource(*seed))

	fmt.Printf("Generating %d agents (seed: %d)\n\n", *count, *seed)

	// Sample the population
	type agent struct {
		Name     string
		Category string
		Model    string
		Styles   []string
		Soul     string
	}

	catCounts := map[string]int{}
	modelCounts := map[string]int{}
	var agents []agent

	for i := 0; i < *count; i++ {
		cat := population.SampleCategory(rng)
		model := population.SampleModel(rng)
		styles := population.SampleStyles(rng)

		name := fmt.Sprintf("%s-%s-%03d", cat.Name, randomSuffix(rng), i+1)

		agents = append(agents, agent{
			Name:     name,
			Category: cat.Name,
			Model:    model.ID,
			Styles:   styles,
			Soul:     cat.Soul,
		})

		catCounts[cat.Name]++
		modelCounts[model.ID]++
	}

	// Print distribution summary
	fmt.Println("=== Category Distribution ===")
	for _, c := range population.Categories {
		if n := catCounts[c.Name]; n > 0 {
			bar := strings.Repeat("█", n)
			fmt.Printf("  %-20s %3d %s\n", c.Name, n, bar)
		}
	}

	fmt.Println("\n=== Model Distribution ===")
	for _, m := range population.Models {
		if n := modelCounts[m.ID]; n > 0 {
			bar := strings.Repeat("█", n)
			fmt.Printf("  %-45s %3d %s\n", m.ID, n, bar)
		}
	}

	if *dryRun {
		fmt.Println("\n(dry run — no agents registered)")
		return
	}

	// Register agents
	fmt.Printf("\n=== Registering %d agents with government %s ===\n", *count, *govID)

	registered := 0
	failed := 0
	for _, a := range agents {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			log.Printf("keygen error: %v", err)
			failed++
			continue
		}
		pubHex := hex.EncodeToString(pub)

		body, _ := json.Marshal(map[string]string{
			"name":       a.Name,
			"public_key": pubHex,
			"soul_md":    a.Soul,
			"model":      a.Model,
		})

		resp, err := http.Post(
			fmt.Sprintf("%s/api/governments/%s/registry/register", *govURL, *govID),
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			log.Printf("register %s: %v", a.Name, err)
			failed++
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 201 {
			log.Printf("register %s: %s", a.Name, string(respBody))
			failed++
			continue
		}

		registered++
		if registered%10 == 0 {
			fmt.Printf("  registered %d/%d...\n", registered, *count)
		}
	}

	fmt.Printf("\nDone: %d registered, %d failed\n", registered, failed)
}

func randomSuffix(rng *mrand.Rand) string {
	adjectives := []string{
		"swift", "bright", "sharp", "calm", "bold",
		"keen", "quick", "wise", "warm", "clear",
		"deep", "fair", "firm", "true", "pure",
	}
	nouns := []string{
		"fox", "oak", "arc", "ray", "elm",
		"gem", "bay", "key", "node", "hub",
		"core", "wave", "link", "spark", "forge",
	}
	return adjectives[rng.Intn(len(adjectives))] + "-" + nouns[rng.Intn(len(nouns))]
}
