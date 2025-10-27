package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// LLMProvider represents different LLM providers
type LLMProvider string

const (
	ProviderNone   LLMProvider = "none"
	ProviderOpenAI LLMProvider = "openai"
	ProviderClaude LLMProvider = "claude"
	ProviderGemini LLMProvider = "gemini"
	ProviderLocal  LLMProvider = "local"
)

// LLMConfig holds LLM integration settings
type LLMConfig struct {
	Enabled  bool
	Provider LLMProvider
	APIKey   string
	APIURL   string
	Model    string
	Timeout  time.Duration
}

// LLMClient handles communication with LLM providers
type LLMClient struct {
	config *LLMConfig
	client *http.Client
}

// NewLLMClient creates a new LLM client
func NewLLMClient(config *LLMConfig) *LLMClient {
	if config == nil || !config.Enabled || config.Provider == ProviderNone {
		return nil
	}

	return &LLMClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// GenerateExecutiveSummary uses LLM to generate executive summary
func (c *LLMClient) GenerateExecutiveSummary(testData map[string]interface{}) (string, error) {
	if c == nil {
		return "", fmt.Errorf("LLM client not initialized")
	}

	prompt := c.buildExecutiveSummaryPrompt(testData)

	switch c.config.Provider {
	case ProviderOpenAI:
		return c.callOpenAI(prompt, 300)
	case ProviderClaude:
		return c.callClaude(prompt, 300)
	case ProviderGemini:
		return c.callGemini(prompt, 300)
	case ProviderLocal:
		return c.callLocal(prompt, 300)
	default:
		return "", fmt.Errorf("unsupported provider: %s", c.config.Provider)
	}
}

// GenerateFixSuggestion uses LLM to suggest fixes for failures
func (c *LLMClient) GenerateFixSuggestion(errorMsg, stackTrace, stepText, specName string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("LLM client not initialized")
	}

	prompt := c.buildFixSuggestionPrompt(errorMsg, stackTrace, stepText, specName)

	switch c.config.Provider {
	case ProviderOpenAI:
		return c.callOpenAI(prompt, 500)
	case ProviderClaude:
		return c.callClaude(prompt, 500)
	case ProviderGemini:
		return c.callGemini(prompt, 500)
	case ProviderLocal:
		return c.callLocal(prompt, 500)
	default:
		return "", fmt.Errorf("unsupported provider: %s", c.config.Provider)
	}
}

// buildExecutiveSummaryPrompt creates the prompt for executive summary
func (c *LLMClient) buildExecutiveSummaryPrompt(data map[string]interface{}) string {
	return fmt.Sprintf(`You are a QA Manager analyzing test automation results. Generate a concise, actionable executive summary (3-4 sentences maximum) for the following test execution:

Test Results:
- Total Scenarios: %v
- Passed: %v
- Failed: %v  
- Skipped: %v
- Success Rate: %.1f%%
- Execution Time: %v

Failed Scenarios:
%v

Requirements:
1. Start with overall health assessment (Excellent/Good/Fair/Poor)
2. Highlight the most critical issues requiring immediate attention
3. Provide one specific, actionable recommendation
4. Use business-friendly language suitable for executives (avoid technical jargon)
5. Be concise but informative

Executive Summary:`,
		data["totalScenarios"],
		data["passed"],
		data["failed"],
		data["skipped"],
		data["successRate"],
		data["duration"],
		data["failedScenarios"],
	)
}

// buildFixSuggestionPrompt creates the prompt for fix suggestions
func (c *LLMClient) buildFixSuggestionPrompt(errorMsg, stackTrace, stepText, specName string) string {
	return fmt.Sprintf(`You are an expert test automation engineer analyzing a test failure. Provide a specific, actionable fix suggestion.

Failed Test:
- Specification: %s
- Step: %s
- Error Message: %s
- Stack Trace:
%s

Provide:
1. Root cause analysis (1-2 sentences)
2. Specific fix recommendation (actionable steps)
3. Code example if applicable (keep it concise)

Focus on the most likely cause and provide practical guidance. Be specific and actionable.

Analysis:`,
		specName,
		stepText,
		errorMsg,
		stackTrace,
	)
}

// callOpenAI makes API call to OpenAI
func (c *LLMClient) callOpenAI(prompt string, maxTokens int) (string, error) {
	requestBody := map[string]interface{}{
		"model": c.config.Model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are an expert QA and test automation consultant.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  maxTokens,
		"temperature": 0.7,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.config.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close() // Ignore close errors in defer
	}()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("API returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return response.Choices[0].Message.Content, nil
}

// callClaude makes API call to Anthropic Claude
func (c *LLMClient) callClaude(prompt string, maxTokens int) (string, error) {
	requestBody := map[string]interface{}{
		"model":      c.config.Model,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.config.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close() // Ignore close errors in defer
	}()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("API returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no response from Claude")
	}

	return response.Content[0].Text, nil
}

// callGemini makes API call to Google Gemini
func (c *LLMClient) callGemini(prompt string, maxTokens int) (string, error) {
	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": maxTokens + 500, // Add buffer for thinking tokens
			"temperature":     0.7,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Gemini uses API key in URL parameter and model in path
	apiURL := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.config.APIURL, c.config.Model, c.config.APIKey)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close() // Ignore close errors in defer
	}()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("API returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read and parse response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w (body: %s)", err, string(bodyBytes))
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Gemini (body: %s)", string(bodyBytes))
	}

	return response.Candidates[0].Content.Parts[0].Text, nil
}

// callLocal makes API call to local LLM server (Ollama, LM Studio, etc.)
func (c *LLMClient) callLocal(prompt string, maxTokens int) (string, error) {
	requestBody := map[string]interface{}{
		"model":  c.config.Model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"num_predict": maxTokens,
			"temperature": 0.7,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.config.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close() // Ignore close errors in defer
	}()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("API returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Response == "" {
		return "", fmt.Errorf("empty response from local LLM")
	}

	return response.Response, nil
}

// LoadLLMConfigFromEnv loads LLM configuration from environment variables
func LoadLLMConfigFromEnv() *LLMConfig {
	// Check if LLM is enabled
	enabled := os.Getenv("GAUGE_AI_ENABLED") == "true"
	if !enabled {
		return &LLMConfig{
			Enabled:  false,
			Provider: ProviderNone,
		}
	}

	provider := LLMProvider(os.Getenv("GAUGE_AI_PROVIDER"))
	if provider == "" {
		provider = ProviderNone
	}

	config := &LLMConfig{
		Enabled:  true,
		Provider: provider,
		APIKey:   os.Getenv("GAUGE_AI_API_KEY"),
		Timeout:  30 * time.Second,
	}

	// Set provider-specific defaults
	switch provider {
	case ProviderOpenAI:
		config.APIURL = getEnvOrDefault("GAUGE_AI_API_URL", "https://api.openai.com/v1/chat/completions")
		config.Model = getEnvOrDefault("GAUGE_AI_MODEL", "gpt-4-turbo-preview")
	case ProviderClaude:
		config.APIURL = getEnvOrDefault("GAUGE_AI_API_URL", "https://api.anthropic.com/v1/messages")
		config.Model = getEnvOrDefault("GAUGE_AI_MODEL", "claude-3-sonnet-20240229")
	case ProviderGemini:
		config.APIURL = getEnvOrDefault("GAUGE_AI_API_URL", "https://generativelanguage.googleapis.com/v1beta")
		config.Model = getEnvOrDefault("GAUGE_AI_MODEL", "gemini-2.0-flash")
	case ProviderLocal:
		config.APIURL = getEnvOrDefault("GAUGE_AI_API_URL", "http://localhost:11434/api/generate")
		config.Model = getEnvOrDefault("GAUGE_AI_MODEL", "llama2")
	}

	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
