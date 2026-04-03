package steps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"time"

	"github.com/atotto/clipboard"
	"github.com/go-vgo/robotgo"
)

type PerplexityClient struct {
	APIKey string
}

type PerplexityBrowser struct {
	PerplexityURL string
}

func NewPerplexityBrowser() *PerplexityBrowser {
	return &PerplexityBrowser{
		PerplexityURL: "https://www.perplexity.ai",
	}
}

func (p *PerplexityBrowser) OpenPerplexity() error {
	if err := exec.Command("open", "-a", "Google Chrome", p.PerplexityURL).Run(); err != nil {
		return err
	}
	time.Sleep(3 * time.Second)
	return nil
}

func (p *PerplexityBrowser) TypePrompt(prompt string) error {
	robotgo.KeySleep = 50
	robotgo.Move(500, 300)
	robotgo.Click("left")
	time.Sleep(500 * time.Millisecond)
	robotgo.TypeStr(prompt)
	time.Sleep(1 * time.Second)
	robotgo.KeyTap("return")
	return nil
}

func (p *PerplexityBrowser) WaitForResponse(timeoutMinutes int) (string, error) {
	timeout := time.Duration(timeoutMinutes) * time.Minute
	start := time.Now()

	for {
		if time.Since(start) > timeout {
			return "", fmt.Errorf("no response within timeout")
		}

		img, err := robotgo.CaptureImg()
		if err == nil && img != nil {
			fmt.Println("Response detected")
			break
		}

		time.Sleep(10 * time.Second)
	}

	return "", nil
}

func (p *PerplexityBrowser) CopyResponse() (string, error) {
	robotgo.KeyTap("a", "command")
	time.Sleep(500 * time.Millisecond)
	robotgo.KeyTap("c", "command")
	time.Sleep(300 * time.Millisecond)
	return clipboard.ReadAll()
}

func (p *PerplexityBrowser) GeneratePrompts(prd string) ([]string, error) {
	fullPrompt := fmt.Sprintf(`You are a senior software architect. Read this PRD and generate a JSON array of sequential coding prompts for OpenCode to build this project step by step. Return ONLY a JSON array like: ["prompt1", "prompt2"]. PRD: %s`, prd)

	if err := p.OpenPerplexity(); err != nil {
		return nil, err
	}

	if err := p.TypePrompt(fullPrompt); err != nil {
		return nil, err
	}

	if _, err := p.WaitForResponse(3); err != nil {
		return nil, err
	}

	response, err := p.CopyResponse()
	if err != nil {
		return nil, err
	}

	return parseJSONArrayFromText(response)
}

func (p *PerplexityBrowser) GetNextPrompt(prd, lastOutput string, iteration int) (string, error) {
	fullPrompt := fmt.Sprintf(`PRD: %s | Last code output iteration %d: %s | What is the next OpenCode prompt to continue building? If project is complete, reply exactly: PROJECT_COMPLETE`, prd, iteration, lastOutput)

	if err := p.TypePrompt(fullPrompt); err != nil {
		return "", err
	}

	if _, err := p.WaitForResponse(3); err != nil {
		return "", err
	}

	response, err := p.CopyResponse()
	if err != nil {
		return "", err
	}

	response = removeMarkdownFormatting(response)
	response = trimSpaces(response)

	if response == "PROJECT_COMPLETE" {
		return "PROJECT_COMPLETE", nil
	}

	return response, nil
}

type PerplexityRequest struct {
	Model      string `json:"model"`
	Messages   []Message `json:"messages"`
	MaxTokens  int     `json:"max_tokens,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type PerplexityResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

func (p *PerplexityClient) GeneratePrompts(prd string) ([]string, error) {
	systemPrompt := "You are a senior software architect. Given a PRD, generate a list of sequential coding prompts for OpenCode to build the project step by step. Return as JSON array of strings."

	body := PerplexityRequest{
		Model: "sonar-pro",
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prd},
		},
		MaxTokens: 4096,
	}

	resp, err := p.doRequest(body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited (429)")
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var perplexityResp PerplexityResponse
	if err := json.NewDecoder(resp.Body).Decode(&perplexityResp); err != nil {
		return nil, err
	}

	if len(perplexityResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := perplexityResp.Choices[0].Message.Content
	return parseJSONArray(content)
}

func (p *PerplexityClient) GetNextPrompt(prd, lastCodeOutput string, iteration int) (string, error) {
	userPrompt := fmt.Sprintf(`Given the PRD: %s

And the last output from OpenCode:
%s

Iteration: %d

What should the next OpenCode prompt be? If the project appears complete, respond with exactly: PROJECT_COMPLETE`, prd, lastCodeOutput, iteration)

	body := PerplexityRequest{
		Model: "sonar-pro",
		Messages: []Message{
			{Role: "user", Content: userPrompt},
		},
		MaxTokens: 2048,
	}

	resp, err := p.doRequest(body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("rate limited (429)")
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var perplexityResp PerplexityResponse
	if err := json.NewDecoder(resp.Body).Decode(&perplexityResp); err != nil {
		return "", err
	}

	if len(perplexityResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	content := perplexityResp.Choices[0].Message.Content
	content = removeMarkdownFormatting(content)

	if content == "PROJECT_COMPLETE" {
		return "PROJECT_COMPLETE", nil
	}

	return content, nil
}

func (p *PerplexityClient) doRequest(body PerplexityRequest) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.perplexity.ai/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	return client.Do(req)
}

func parseJSONArray(content string) ([]string, error) {
	content = extractJSONArray(content)
	var prompts []string
	if err := json.Unmarshal([]byte(content), &prompts); err != nil {
		preview := content
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("failed to parse JSON array: %v\nRaw content: %s", err, preview)
	}
	return prompts, nil
}

func parseJSONArrayFromText(text string) ([]string, error) {
	text = extractJSONArray(text)
	var prompts []string
	if err := json.Unmarshal([]byte(text), &prompts); err != nil {
		return nil, fmt.Errorf("no JSON array found in text")
	}
	return prompts, nil
}

// extractJSONArray finds the first balanced [...] block in the text.
func extractJSONArray(content string) string {
	start := -1
	depth := 0
	for i, ch := range content {
		if ch == '[' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if ch == ']' {
			depth--
			if depth == 0 && start >= 0 {
				return content[start : i+1]
			}
		}
	}
	return content
}

func removeMarkdownFormatting(content string) string {
	// Strip ```json ... ``` wrappers
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.+?)\\s*```")
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		return matches[1]
	}
	return content
}

func trimSpaces(s string) string {
	re := regexp.MustCompile(`^\s+|\s+$`)
	return re.ReplaceAllString(s, "")
}
