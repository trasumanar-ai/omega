// Signing proxy: runs inside each agent container on localhost:9999.
// Forwards requests to the government API with Ed25519 signatures.
// Agents just call curl http://localhost:9999/bank/balance — no signing needed.
package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	govURL := os.Getenv("OMEGA_URL")
	govID := os.Getenv("OMEGA_GOV_ID")
	keyDir := os.Getenv("OMEGA_KEY_DIR")
	port := os.Getenv("PROXY_PORT")

	if govURL == "" || govID == "" {
		log.Fatal("OMEGA_URL and OMEGA_GOV_ID required")
	}
	if keyDir == "" {
		keyDir = "/home/node/work/keys"
	}
	if port == "" {
		port = "9999"
	}

	privKey, pubHex, err := loadKeys(keyDir)
	if err != nil {
		log.Fatalf("load keys from %s: %v", keyDir, err)
	}

	proxy := &signingProxy{
		govURL: strings.TrimRight(govURL, "/"),
		govID:  govID,
		priv:   privKey,
		pubHex: pubHex,
	}

	log.Printf("signing proxy on :%s → %s/api/governments/%s (pubkey: %s...)", port, govURL, govID, pubHex[:16])
	log.Fatal(http.ListenAndServe(":"+port, proxy))
}

type signingProxy struct {
	govURL string
	govID  string
	priv   ed25519.PrivateKey
	pubHex string
}

func (p *signingProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Map: localhost:9999/bank/balance → govURL/api/governments/{id}/bank/balance
	targetPath := fmt.Sprintf("/api/governments/%s%s", p.govID, r.URL.Path)
	targetURL := p.govURL + targetPath

	req, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Copy headers
	for k, v := range r.Header {
		req.Header[k] = v
	}

	// Sign the request
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	msg := ts + ":" + strings.ToUpper(r.Method) + ":" + targetPath
	sig := ed25519.Sign(p.priv, []byte(msg))
	req.Header.Set("Authorization", fmt.Sprintf("Signed %s:%s:%s", p.pubHex, ts, hex.EncodeToString(sig)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func loadKeys(dir string) (ed25519.PrivateKey, string, error) {
	privData, err := os.ReadFile(filepath.Join(dir, "omega.key"))
	if err != nil {
		return nil, "", err
	}
	pubData, err := os.ReadFile(filepath.Join(dir, "omega.pub"))
	if err != nil {
		return nil, "", err
	}

	privBytes, err := hex.DecodeString(strings.TrimSpace(string(privData)))
	if err != nil || len(privBytes) != ed25519.PrivateKeySize {
		return nil, "", fmt.Errorf("invalid private key")
	}
	return ed25519.PrivateKey(privBytes), strings.TrimSpace(string(pubData)), nil
}
