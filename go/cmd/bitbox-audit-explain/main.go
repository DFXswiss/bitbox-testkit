// Command bitbox-audit-explain turns the structured JSON output of
// bitbox-audit into a plain-language report. Without an Anthropic API key
// it prints the prompt it would send (useful for manual review); with
// ANTHROPIC_API_KEY set, it calls Claude and prints the model's reply.
//
// Pipeline:
//
//	bitbox-audit --repo /path > findings.json
//	bitbox-audit-explain --input findings.json
//
// Or via pipe:
//
//	bitbox-audit --repo /path | bitbox-audit-explain
//
// The prompt template lives in this file so reviewers can audit what
// gets sent to the model.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	anthropicEndpoint = "https://api.anthropic.com/v1/messages"
	anthropicVersion  = "2023-06-01"
	defaultModel      = "claude-opus-4-7"
	defaultMaxTokens  = 2048
)

const promptTemplate = `You are a hardware-wallet integration reviewer. Given a structured BitBox audit report, write a short, actionable narrative for the developer.

For each finding:
  1. Restate the bug class in one sentence (no jargon).
  2. Explain WHY the firmware/protocol behaves this way.
  3. Suggest a concrete fix.

Group output by severity (Critical, Warning, Hint). If there are zero findings, say so plainly and list the quirk classes that were checked.

Audit JSON follows:

%s
`

func main() {
	var (
		input    = flag.String("input", "", "audit JSON file (default: stdin)")
		model    = flag.String("model", defaultModel, "Anthropic model identifier")
		printOnly = flag.Bool("print-prompt", false, "print the prompt and exit (no API call)")
	)
	flag.Parse()

	data, err := readInput(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bitbox-audit-explain: read input: %v\n", err)
		os.Exit(1)
	}

	// Pretty-print the JSON so the prompt is readable to a human inspecting it.
	pretty := bytes.Buffer{}
	if err := json.Indent(&pretty, data, "", "  "); err != nil {
		fmt.Fprintf(os.Stderr, "bitbox-audit-explain: input is not valid JSON: %v\n", err)
		os.Exit(1)
	}

	prompt := fmt.Sprintf(promptTemplate, pretty.String())

	if *printOnly {
		fmt.Println(prompt)
		return
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY not set — printing the prompt instead. Pipe it to your assistant of choice, or run with --print-prompt for explicit no-call mode.")
		fmt.Println(prompt)
		return
	}

	out, err := callClaude(apiKey, *model, prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bitbox-audit-explain: API call failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(out)
}

func readInput(path string) ([]byte, error) {
	if path == "" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

// Anthropic Messages API request shape (minimal subset).
type anthropicReq struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResp struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func callClaude(apiKey, model, prompt string) (string, error) {
	body, err := json.Marshal(anthropicReq{
		Model:     model,
		MaxTokens: defaultMaxTokens,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", anthropicEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %s: %s", resp.Status, string(respBody))
	}

	var parsed anthropicResp
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("api error %s: %s", parsed.Error.Type, parsed.Error.Message)
	}

	var sb bytes.Buffer
	for _, c := range parsed.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return sb.String(), nil
}
