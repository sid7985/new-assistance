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
}

func NewBot(token, chatID, projectDir string, handler func(string) (string, error)) *Bot {
	return &Bot{
		Token:         token,
		ChatID:        chatID,
		ProjectDir:    projectDir,
		ActionHandler: handler,
	}
}

type telegramResponse struct {
	OK     bool     `json:"ok"`
	Result []update `json:"result"`
}

type update struct {
	UpdateID int     `json:"update_id"`
	Message  message `json:"message"`
}

type message struct {
	Text string `json:"text"`
	Chat chat   `json:"chat"`
}

type chat struct {
	ID int64 `json:"id"`
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

				matches := repoURLPattern.FindAllString(u.Message.Text, -1)
				if len(matches) == 0 {
					fmt.Printf("📨 Received chat request from Telegram: %s\n", u.Message.Text)
					if b.ActionHandler != nil {
						b.SendMessage("⏳ Processing...")
						// Run asynchronously to prevent blocking Telegram loop for long OS tasks
						go func(text string) {
							reply, err := b.ActionHandler(text)
							if err != nil {
								b.SendMessage(fmt.Sprintf("❌ Command error: %v", err))
							} else {
								b.SendMessage(fmt.Sprintf("🤖 %s", reply))
							}
						}(u.Message.Text)
					} else {
						b.SendMessage("🤖 NanoClaw Action Handler offline.")
					}
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
