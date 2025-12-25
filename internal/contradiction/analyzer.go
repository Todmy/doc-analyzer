package contradiction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Analyzer detects contradictions between statement pairs using Claude API
type Analyzer struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// Config holds analyzer configuration
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		BaseURL: "https://api.anthropic.com/v1",
		Model:   "claude-3-haiku-20240307",
		Timeout: 30 * time.Second,
	}
}

// NewAnalyzer creates a new contradiction analyzer
func NewAnalyzer(config Config) *Analyzer {
	if config.BaseURL == "" {
		config.BaseURL = DefaultConfig().BaseURL
	}
	if config.Model == "" {
		config.Model = DefaultConfig().Model
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultConfig().Timeout
	}

	return &Analyzer{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		model:   config.Model,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// AnalyzePair analyzes a single pair for contradictions
func (a *Analyzer) AnalyzePair(ctx context.Context, pair StatementPair) (*ContradictionResult, error) {
	prompt := buildPrompt(pair)

	response, err := a.callClaude(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("call claude: %w", err)
	}

	result, err := parseResponse(response, pair)
	if err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return result, nil
}

// AnalyzePairs analyzes multiple pairs concurrently
func (a *Analyzer) AnalyzePairs(ctx context.Context, pairs []StatementPair, maxConcurrent int) ([]ContradictionResult, error) {
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	results := make([]ContradictionResult, 0)
	sem := make(chan struct{}, maxConcurrent)

	type result struct {
		contradiction *ContradictionResult
		err           error
	}
	resultChan := make(chan result, len(pairs))

	for _, pair := range pairs {
		sem <- struct{}{}
		go func(p StatementPair) {
			defer func() { <-sem }()

			cr, err := a.AnalyzePair(ctx, p)
			resultChan <- result{contradiction: cr, err: err}
		}(pair)
	}

	for range pairs {
		r := <-resultChan
		if r.err != nil {
			continue // Skip errors, log them in production
		}
		if r.contradiction != nil && r.contradiction.Type != "" {
			results = append(results, *r.contradiction)
		}
	}

	return results, nil
}

func buildPrompt(pair StatementPair) string {
	return fmt.Sprintf(`Analyze these two statements for contradictions:

Statement 1: "%s"
Statement 2: "%s"

Determine if they contradict each other. If yes, respond with JSON:
{
  "is_contradiction": true,
  "type": "direct|numerical|temporal|implicit",
  "severity": "high|medium|low",
  "explanation": "brief explanation",
  "confidence": 0.0-1.0
}

If no contradiction, respond:
{"is_contradiction": false}

Respond ONLY with valid JSON.`, pair.Statement1, pair.Statement2)
}

type claudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (a *Analyzer) callClaude(ctx context.Context, prompt string) (string, error) {
	reqBody := claudeRequest{
		Model:     a.model,
		MaxTokens: 500,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	var cr claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", err
	}

	if len(cr.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return cr.Content[0].Text, nil
}

type analysisResponse struct {
	IsContradiction bool    `json:"is_contradiction"`
	Type            string  `json:"type"`
	Severity        string  `json:"severity"`
	Explanation     string  `json:"explanation"`
	Confidence      float64 `json:"confidence"`
}

func parseResponse(response string, pair StatementPair) (*ContradictionResult, error) {
	var ar analysisResponse
	if err := json.Unmarshal([]byte(response), &ar); err != nil {
		return nil, err
	}

	if !ar.IsContradiction {
		return nil, nil
	}

	return &ContradictionResult{
		Statement1:   pair.Statement1,
		Statement2:   pair.Statement2,
		Statement1ID: pair.Statement1ID,
		Statement2ID: pair.Statement2ID,
		File1:        pair.File1,
		File2:        pair.File2,
		Type:         ContradictionType(ar.Type),
		Severity:     Severity(ar.Severity),
		Explanation:  ar.Explanation,
		Confidence:   ar.Confidence,
	}, nil
}
