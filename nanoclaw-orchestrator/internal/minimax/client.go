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
const defaultModel = "MiniMax-M2.7"

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
		MaxTokens: 1024,
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

	return c.callAnthropicAPI(payload)
}

func decodeFirstJSON(input string, v interface{}) error {
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
	systemPrompt := `You are an AI desktop agent. Map the user's request to one of the following tools.
Tools:
1. "OpenDocument" - Args: "path" (the file/folder path)
2. "CreateDocument" - Args: "path", "content" (the file path and the file content)
3. "WebSearch" - Args: "query"
4. "YouTubeSearch" - Args: "query"
5. "PlayMusic" - Args: "query"
6. "TerminalCommand" - Args: "command" (if they just ask to run a generic shell command)
7. "OpenCode" - Args: "prompt" (if the user asks to write code, generate an app, or edit files in the project workspace)
8. "AgentDesktopControl" - Args: "prompt" (if the user asks to control the PC directly via visual/mouse/keyboard steps)
9. "ChatResponse" - Args: "reply" (if the user is just chatting, saying hi, or asking a conversational question)

Respond with ONLY ONE valid JSON object representing the NEXT logical step.
Note: On this system, use 'python3' instead of 'python' for shell commands.
Do not include any Markdown, explanations, or multiple objects.
Example:
{"action": "ActionName", "path": "...", "content": "...", "query": "...", "command": "...", "prompt": "...", "reply": "..."}`

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
	if err := decodeFirstJSON(content, &result); err != nil {
		return nil, fmt.Errorf("AI response was not valid JSON: %s. error: %v", content, err)
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
	if err := decodeFirstJSON(content, &result); err != nil {
		return nil, fmt.Errorf("AI response was not valid JSON: %s. error: %v", content, err)
	}

	return result, nil
}

func (c *Client) GetManagerPlan(directive string) (string, error) {
	systemPrompt := `You are the PURPLE MANAGER AGENT for NanoClaw. 
Your job is to receive the Founder's (User's) directive and break it into a logical sequence of subtasks for your team.

Available specialized roles:
- CODER: Write code, create files, run terminal commands, or handle file operations.
- RESEARCHER: Search the web, find information, or analyze data.
- DESIGNER: Handle visual layouts, logic design, or styling.
- TESTER: Verify work, check for errors, or validate results.

You MUST respond in this EXACT format for each delegation:
-> [ROLE]: detailed task description

Example:
-> [RESEARCHER]: find the latest news about Go 1.25
-> [CODER]: create a summary.md with the findings

After the delegations, provide a one-line mission summary.
Do not include any other conversational text.

IMPORTANT: Each delegation must be a SINGLE ATOMIC ACTION. 
If you need to create a script and then run it, you MUST create TWO delegations (one for CODER: CreateDocument and one for CODER: TerminalCommand).
On this system, always use 'python3' to run scripts.`

	payload := MessagePayload{
		Model:     defaultModel,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages: []Message{
			{Role: "user", Content: []Content{{Type: "text", Text: directive}}},
		},
	}

	return c.callAnthropicAPI(payload)
}
