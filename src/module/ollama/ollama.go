package ollama

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	DefaultOllamaURL   = "http://localhost:11434/api/generate"
	DefaultOllamaModel = "gemma4:31b-cloud"
)

type OllamaClient struct {
	URL   string
	Model string
}

type ollamaRequest struct {
	Model  string   `json:"model"`
	System string   `json:"system,omitempty"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images"`
	Stream bool     `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

func (c *OllamaClient) IsRunning() bool {
	base := strings.TrimSuffix(c.URL, "/api/generate")
	resp, err := http.Get(base)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *OllamaClient) Generate(system, prompt, imagePath string) (string, error) {
	imgBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("cannot read image: %w", err)
	}

	model := c.Model
	if model == "" {
		model = DefaultOllamaModel
	}

	req := ollamaRequest{
		Model:  model,
		System: system,
		Prompt: prompt,
		Images: []string{base64.StdEncoding.EncodeToString(imgBytes)},
		Stream: false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(c.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned HTTP %d: %s", resp.StatusCode, raw)
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("cannot decode ollama response: %w", err)
	}

	return result.Response, nil
}
