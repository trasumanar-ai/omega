package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const usage = `omega-id: identity tool for Omega government agents

Commands:
  generate              Generate a new key pair in current directory
  pubkey                Print public key
  sign <method> <path>  Sign a request, output Authorization header value
  verify <pubkey> <sig> <message>  Verify a signature

Files:
  omega.key     Private key (keep secret!)
  omega.pub     Public key (share freely)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	keyDir := os.Getenv("OMEGA_KEY_DIR")
	if keyDir == "" {
		keyDir = "."
	}

	switch os.Args[1] {
	case "generate":
		cmdGenerate(keyDir)
	case "pubkey":
		cmdPubkey(keyDir)
	case "sign":
		if len(os.Args) < 4 {
			fatal("usage: omega-id sign <method> <path>")
		}
		cmdSign(keyDir, os.Args[2], os.Args[3])
	case "verify":
		if len(os.Args) < 5 {
			fatal("usage: omega-id verify <pubkey> <sig> <message>")
		}
		cmdVerify(os.Args[2], os.Args[3], os.Args[4])
	default:
		fmt.Print(usage)
		os.Exit(1)
	}
}

func cmdGenerate(dir string) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fatal("generate key: %v", err)
	}

	privPath := filepath.Join(dir, "omega.key")
	pubPath := filepath.Join(dir, "omega.pub")

	if err := os.WriteFile(privPath, []byte(hex.EncodeToString(privKey)), 0600); err != nil {
		fatal("write private key: %v", err)
	}
	if err := os.WriteFile(pubPath, []byte(hex.EncodeToString(pubKey)), 0644); err != nil {
		fatal("write public key: %v", err)
	}

	fmt.Printf("Key pair generated:\n")
	fmt.Printf("  Private: %s (keep secret!)\n", privPath)
	fmt.Printf("  Public:  %s\n", pubPath)
	fmt.Printf("  ID:      %s\n", hex.EncodeToString(pubKey))
}

func cmdPubkey(dir string) {
	pubPath := filepath.Join(dir, "omega.pub")
	data, err := os.ReadFile(pubPath)
	if err != nil {
		fatal("read public key: %v (run 'omega-id generate' first)", err)
	}
	fmt.Print(strings.TrimSpace(string(data)))
}

func cmdSign(dir, method, path string) {
	privPath := filepath.Join(dir, "omega.key")
	data, err := os.ReadFile(privPath)
	if err != nil {
		fatal("read private key: %v", err)
	}

	privBytes, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		fatal("decode private key: %v", err)
	}
	privKey := ed25519.PrivateKey(privBytes)

	pubPath := filepath.Join(dir, "omega.pub")
	pubData, err := os.ReadFile(pubPath)
	if err != nil {
		fatal("read public key: %v", err)
	}
	pubHex := strings.TrimSpace(string(pubData))

	// Message = timestamp:METHOD:path
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	message := ts + ":" + strings.ToUpper(method) + ":" + path

	sig := ed25519.Sign(privKey, []byte(message))
	sigHex := hex.EncodeToString(sig)

	// Output the full Authorization header value
	fmt.Printf("Signed %s:%s:%s", pubHex, ts, sigHex)
}

func cmdVerify(pubHex, sigHex, message string) {
	pubBytes, err := hex.DecodeString(pubHex)
	if err != nil {
		fatal("decode public key: %v", err)
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		fatal("decode signature: %v", err)
	}

	if ed25519.Verify(ed25519.PublicKey(pubBytes), []byte(message), sigBytes) {
		fmt.Println("valid")
	} else {
		fmt.Println("invalid")
		os.Exit(1)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
