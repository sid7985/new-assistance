package minimax

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

// MiniMax Anthropic-compatible endpoint
const minimaxAPIEndpoint = "https://api.minimax.io/anthropic/v1/messages"
const defaultModel = "MiniMax-M2.7" // Flagship agentic model

type Client struct {
	APIKey  string
	GroupID string
}

func NewClient(apiKey, groupID string) *Client {
	return &Client{APIKey: apiKey, GroupID: groupID}
}

// Anthropic Messages API structures
type MessagePayload struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type   string  `json:"type"`
	Text   string  `json:"text,omitempty"`
	Source *Source `json:"source,omitempty"`
}

type Source struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type Response struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) callAnthropicAPI(payload MessagePayload) (string, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", minimaxAPIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if c.GroupID != "" {
		req.Header.Set("Group-Id", c.GroupID)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("MiniMax API error: status %d, body %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp Response
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse MiniMax response: %v. body: %s", err, string(bodyBytes))
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("MiniMax API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("no content in response. body: %s", string(bodyBytes))
	}

	var fullText strings.Builder
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			fullText.WriteString(block.Text)
		}
	}

	return fullText.String(), nil
}

func (c *Client) AnalyzeScreen(imagePath string, promptContext string) (string, error) {
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %v", err)
	}

	base64Img := base64.StdEncoding.EncodeToString(imgData)

	payload := MessagePayload{
		Model:     defaultModel,
		MaxTokens: 2048, // Increased for M2.7
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{
						Type: "image",
						Source: &Source{
							Type:      "base64",
							MediaType: "image/jpeg",
							Data:      base64Img,
						},
					},
					{Type: "text", Text: "Analyze this screen. " + promptContext},
				},
			},
		},
	}

	return c.callAnthropicAPI(payload)
}

func DecodeFirstJSON(input string, v interface{}) error {
	start := strings.Index(input, "{")
	if start == -1 {
		return fmt.Errorf("no valid JSON object found in input")
	}

	// Manually find the matching closing brace to extract exactly one object
	// This handles "chatty" models that append text after the JSON
	depth := 0
	end := -1
	for i := start; i < len(input); i++ {
		if input[i] == '{' {
			depth++
		} else if input[i] == '}' {
			depth--
			if depth == 0 {
				end = i
				break
			}
		}
	}

	if end == -1 {
		return fmt.Errorf("no balanced JSON object found in input")
	}

	return json.Unmarshal([]byte(input[start:end+1]), v)
}

func (c *Client) GetDesktopAction(promptContext string) (map[string]string, error) {
	systemPrompt := `You are the MINI-MAX 2.7 AGENT CORE. You control a macOS workstation.
Map the Founder's request to the NEXT logical tool.

TOOLS:
- "OpenDocument": {"path": "..."} - Open file/app.
- "CreateDocument": {"path": "...", "content": "..."} - Write file.
- "WebSearch": {"query": "..."} - Search Google.
- "YouTubeSearch": {"query": "..."} - Search YouTube.
- "PlayMusic": {"query": "..."} - Play Spotify.
- "TerminalCommand": {"command": "..."} - Run shell (use python3).
- "OpenCode": {"prompt": "..."} - Write/Edit project code.
- "AgentDesktopControl": {"prompt": "..."} - VISUAL CONTROL loop (Click/Type).
- "ChatResponse": {"reply": "..."} - Conversational reply.
- "WhatsApp": {"phone": "...", "text": "..."} - Send WhatsApp.
- "Paint": {} - Open Freeform/Drawing.

Respond with ONLY one JSON object. Be decisive and autonomous.
Example: {"action": "TerminalCommand", "command": "python3 script.py"}`

	payload := MessagePayload{
		Model:     defaultModel,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages: []Message{
			{Role: "user", Content: []Content{{Type: "text", Text: promptContext}}},
		},
	}

	content, err := c.callAnthropicAPI(payload)
	if err != nil {
		return nil, err
	}

	var result map[string]string
	if err := DecodeFirstJSON(content, &result); err != nil {
		return nil, fmt.Errorf("MiniMax 2.7 error: Invalid JSON. body: %s", content)
	}

	return result, nil
}

func (c *Client) GetVisionDesktopAction(promptContext string, imagePath string) (map[string]interface{}, error) {
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %v", err)
	}

	base64Img := base64.StdEncoding.EncodeToString(imgData)

	systemPrompt := `You are an AI desktop agent. Map the user's intent to one of the following primitive actions based on the screen image.
Actions:
1. "MouseClick" - Args: "x" (int), "y" (int)
2. "MouseDoubleClick" - Args: "x" (int), "y" (int)
3. "KeyboardType" - Args: "text" (string)
4. "KeyboardPress" - Args: "key" (string)
5. "MouseScroll" - Args: "x" (int), "y" (int), "direction" (string, "up" or "down")
6. "TaskComplete" - Args: "summary" (string)

Respond with ONLY ONE valid JSON object representing the NEXT logical step.
Do not include any Markdown, explanations, or multiple objects.
Example:
{"action": "ActionName", "x": 100, "y": 200, "text": "...", "key": "...", "direction": "up", "summary": "..."}`

	payload := MessagePayload{
		Model:     defaultModel,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{
						Type: "image",
						Source: &Source{
							Type:      "base64",
							MediaType: "image/jpeg",
							Data:      base64Img,
						},
					},
					{Type: "text", Text: promptContext},
				},
			},
		},
	}

	content, err := c.callAnthropicAPI(payload)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := DecodeFirstJSON(content, &result); err != nil {
		return nil, fmt.Errorf("AI response was not valid JSON: %s. error: %v", content, err)
	}

	return result, nil
}

func (c *Client) GetManagerPlan(directive string) (string, error) {
	systemPrompt := `You are the PRIMARY MINI-MAX 2.7 ORCHESTRATOR.
Break the objective into a sequence of delegations for specialized sub-agents.

ROLES:
- CODER: File systems, coding, terminal work.
- RESEARCHER: Web intelligence, data gathering.
- DESIGNER: Visuals, App structure, logic.
- TESTER: Validation, bug hunting.

Format:
-> [ROLE]: step description
...
Mission Summary: short summary

Be aggressive and autonomous. Do not ask for permissions. Just execute.`

	payload := MessagePayload{
		Model:     defaultModel,
		MaxTokens: 2048,
		System:    systemPrompt,
		Messages: []Message{
			{Role: "user", Content: []Content{{Type: "text", Text: directive}}},
		},
	}

	return c.callAnthropicAPI(payload)
}
