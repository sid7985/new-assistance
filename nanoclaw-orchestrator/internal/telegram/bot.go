package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"nanoclaw-orchestrator/internal/computer"
)

const telegramAPI = "https://api.telegram.org/bot"

type Bot struct {
	Token      string
	ChatID     string
	ProjectDir    string
	lastUpdate    int
	ActionHandler func(string) (string, error)
	MachineName   string
}

func NewBot(token, chatID, projectDir, machineName string, handler func(string) (string, error)) *Bot {
	return &Bot{
		Token:         token,
		ChatID:        chatID,
		ProjectDir:    projectDir,
		ActionHandler: handler,
		MachineName:   machineName,
	}
}

type telegramResponse struct {
	OK     bool     `json:"ok"`
	Result []update `json:"result"`
}

type update struct {
	UpdateID      int           `json:"update_id"`
	Message       *message      `json:"message,omitempty"`
	CallbackQuery *callbackQuery `json:"callback_query,omitempty"`
}

type message struct {
	MessageID int    `json:"message_id"`
	Text      string `json:"text"`
	Chat      chat   `json:"chat"`
}

type callbackQuery struct {
	ID      string   `json:"id"`
	From    user     `json:"from"`
	Message *message `json:"message,omitempty"`
	Data    string   `json:"data"` // This will store the MachineName selected
}

type user struct {
	ID int64 `json:"id"`
}

type chat struct {
	ID int64 `json:"id"`
}

type inlineKeyboardMarkup struct {
	InlineKeyboard [][]inlineKeyboardButton `json:"inline_keyboard"`
}

type inlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

var repoURLPattern = regexp.MustCompile(`https?://[^\s]+\.git|https?://github\.com/[^\s]+`)

// SendMessage sends a text message to the configured Telegram chat.
func (b *Bot) SendMessage(text string) error {
	encodedText := url.QueryEscape(text)
	requestURL := fmt.Sprintf("%s%s/sendMessage?chat_id=%s&text=%s",
		telegramAPI, b.Token, b.ChatID, encodedText)
	resp, err := http.Get(requestURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SendMessageWithKeyboard sends a message with inline buttons for machine selection.
func (b *Bot) SendMessageWithKeyboard(text string, machines []string) error {
	var buttons []inlineKeyboardButton
	for _, m := range machines {
		buttons = append(buttons, inlineKeyboardButton{
			Text:         m,
			CallbackData: m + "::" + text, // "Mac::Take a screenshot"
		})
	}

	markup := inlineKeyboardMarkup{
		InlineKeyboard: [][]inlineKeyboardButton{buttons},
	}

	payload := map[string]interface{}{
		"chat_id":      b.ChatID,
		"text":         text,
		"reply_markup": markup,
	}

	body, _ := json.Marshal(payload)
	requestURL := fmt.Sprintf("%s%s/sendMessage", telegramAPI, b.Token)
	resp, err := http.Post(requestURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SendScreenshot sends an image file to the configured Telegram chat.
func (b *Bot) SendScreenshot(imagePath string, caption string) error {
	file, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("photo", filepath.Base(imagePath))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	_ = writer.WriteField("chat_id", b.ChatID)
	_ = writer.WriteField("caption", caption)
	err = writer.Close()
	if err != nil {
		return err
	}

	requestURL := fmt.Sprintf("%s%s/sendPhoto", telegramAPI, b.Token)
	req, err := http.NewRequest("POST", requestURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send photo: %s", string(respBody))
	}

	return nil
}

// PollForRepoURLs checks Telegram for new messages containing repo URLs.
// When found, it clones the repo into the project directory and notifies.
func (b *Bot) PollForRepoURLs(stopChan <-chan struct{}) {
	fmt.Println("📡 Telegram bot listening for repo URLs...")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			fmt.Println("Telegram bot stopped.")
			return
		case <-ticker.C:
			updates, err := b.getUpdates()
			if err != nil {
				continue
			}

			for _, u := range updates {
				if u.UpdateID <= b.lastUpdate {
					continue
				}
				b.lastUpdate = u.UpdateID

				// 1. Handle Callback Queries (Button Clicks)
				if u.CallbackQuery != nil {
					data := u.CallbackQuery.Data // e.g. "Mac::Take a screenshot"
					parts := strings.SplitN(data, "::", 2)
					if len(parts) == 2 {
						targetMachine := parts[0]
						commandText := parts[1]

						// Each instance checks if it is the target
						if targetMachine == b.MachineName {
							fmt.Printf("🎯 [Machine %s] executing: %s\n", b.MachineName, commandText)
							b.SendMessage(fmt.Sprintf("⚡️ [Machine %s] Proceeding: %s", b.MachineName, commandText))
							
							if b.ActionHandler != nil {
								go func(text string) {
									reply, err := b.ActionHandler(text)
									if err != nil {
										b.SendMessage(fmt.Sprintf("❌ [%s] Error: %v", b.MachineName, err))
									} else {
										b.SendMessage(fmt.Sprintf("🤖 [%s] %s", b.MachineName, reply))
									}
								}(commandText)
							}
						}
					}
					continue
				}

				// 2. Handle standard messages (Command Input)
				if u.Message == nil || u.Message.Text == "" {
					continue
				}

				matches := repoURLPattern.FindAllString(u.Message.Text, -1)
				if len(matches) == 0 {
					fmt.Printf("📨 Received chat request from Telegram: %s\n", u.Message.Text)
					
					// Instead of immediate execution, ask which machine to use
					machines := []string{"Mac", "Windows"}
					b.SendMessageWithKeyboard("Which machine should handle this?", machines)
					
					// Store the command context in the button's callback data
					// Note: Telegram callback data has a 64-byte limit. 
					// If command is long, we'd need a DB-backed session ID instead.
					// For now, let's just send the machines prompt.

					continue
				}

				for _, repoURL := range matches {
					repoURL = strings.TrimSpace(repoURL)
					fmt.Printf("📨 Received repo URL from Telegram: %s\n", repoURL)

					b.SendMessage(fmt.Sprintf("⏳ Cloning repository: %s...", repoURL))
					if err := computer.CloneRepo(repoURL, b.ProjectDir); err != nil {
						b.SendMessage(fmt.Sprintf("❌ Failed to clone %s: %v", repoURL, err))
					} else {
						b.SendMessage(fmt.Sprintf("✅ Successfully cloned %s into project folder!", repoURL))
					}
				}
			}
		}
	}
}

func (b *Bot) getUpdates() ([]update, error) {
	url := fmt.Sprintf("%s%s/getUpdates?offset=%d&timeout=1",
		telegramAPI, b.Token, b.lastUpdate+1)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tgResp telegramResponse
	if err := json.Unmarshal(body, &tgResp); err != nil {
		return nil, err
	}

	return tgResp.Result, nil
}
