package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Bot struct {
	Token  string
	ChatID string
	Client *http.Client
}

func NewBot(token, chatID string) *Bot {
	if token == "" || chatID == "" {
		return nil
	}
	return &Bot{
		Token:  token,
		ChatID: chatID,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (b *Bot) SendMessage(text string) error {
	if b == nil {
		return nil
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.Token)
	payload := map[string]string{
		"chat_id": b.ChatID,
		"text":    text,
		"parse_mode": "HTML",
	}

	jsonPayload, _ := json.Marshal(payload)
	resp, err := b.Client.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}
