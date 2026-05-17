package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// SanitizeInput removes potential XSS and injection characters
func sanitizeInput(input string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(input, "")
}

// ValidateChatID ensures chat ID is safe
func validateChatID(chatID string) error {
	// Chat ID should be numeric only
	re := regexp.MustCompile(`^\d+$`)
	if !re.MatchString(chatID) {
		return fmt.Errorf("invalid chat ID format")
	}
	return nil
}

// ValidateToken ensures bot token is safe
func validateToken(token string) error {
	// Basic format validation (numbers:numbers)
	re := regexp.MustCompile(`^\d+:[\w-]+$`)
	if !re.MatchString(token) {
		return fmt.Errorf("invalid bot token format")
	}
	return nil
}

type Bot struct {
	Token  string
	ChatID string
	Client *http.Client
}

func NewBot(token, chatID string) *Bot {
	// Validate inputs before creating bot
	if err := validateToken(token); err != nil {
		return nil
	}
	if err := validateChatID(chatID); err != nil {
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

	// Sanitize message content to prevent XSS
	sanitizedText := sanitizeInput(text)
	if len(sanitizedText) > 4000 { // Limit message length
		sanitizedText = sanitizedText[:4000]
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.Token)
	payload := map[string]string{
		"chat_id": b.ChatID,
		"text":    sanitizedText,
		"parse_mode": "HTML",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}
	
	// Add rate limiting
	time.Sleep(100 * time.Millisecond)
	
	resp, err := b.Client.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}