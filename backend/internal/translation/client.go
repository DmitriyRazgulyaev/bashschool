package translation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"bashkir-school/backend/internal/config"
	"bashkir-school/backend/internal/domain"
)

type Client struct{}

func (c *Client) Configured() bool { return config.Env("AI_API_KEY", "") != "" }
func (c *Client) Model() string    { return config.Env("AI_MODEL", "gpt-4.1-mini") }

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *Client) Generate(ctx context.Context, text, source, target string) (domain.AITranslation, error) {
	key := strings.TrimSpace(os.Getenv("AI_API_KEY"))
	if key == "" {
		return domain.AITranslation{}, errors.New("AI_API_KEY не настроен в .env")
	}
	url := config.Env("AI_API_URL", "https://api.openai.com/v1/chat/completions")
	model := config.Env("AI_MODEL", "gpt-4.1-mini")
	request := map[string]any{"model": model, "temperature": 0.4, "response_format": map[string]string{"type": "json_object"}, "messages": []map[string]string{
		{"role": "system", "content": "You are a language teacher. Output ONLY valid JSON with string keys correct, incorrect, errorExplanation. Translate source text to target language accurately, then make a clearly incorrect but plausible alternative with one meaning-changing error. Do not invent source or metadata. Never include explanation in the translation strings. Never include markdown."},
		{"role": "user", "content": fmt.Sprintf("Source language: %s. Target language: %s. Source text: %q. Produce correct and intentionally incorrect translations, plus explanation of the mistake in Russian.", source, target, text)},
	}}
	b, _ := json.Marshal(request)
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return domain.AITranslation{}, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 32 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return domain.AITranslation{}, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 128*1024))
	if err != nil {
		return domain.AITranslation{}, err
	}
	if res.StatusCode != 200 {
		return domain.AITranslation{}, fmt.Errorf("AI HTTP %d (проверьте ключ, модель и лимиты)", res.StatusCode)
	}
	var chat chatResponse
	if err = json.Unmarshal(data, &chat); err != nil {
		return domain.AITranslation{}, err
	}
	if len(chat.Choices) == 0 {
		return domain.AITranslation{}, errors.New("AI вернул пустой ответ")
	}
	var out domain.AITranslation
	if err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &out); err != nil {
		return domain.AITranslation{}, fmt.Errorf("некорректный JSON модели: %w", err)
	}
	out.Correct = strings.TrimSpace(out.Correct)
	out.Incorrect = strings.TrimSpace(out.Incorrect)
	if out.Correct == "" || out.Incorrect == "" || out.Correct == out.Incorrect || utf8.RuneCountInString(out.Correct) > 2000 || utf8.RuneCountInString(out.Incorrect) > 2000 {
		return domain.AITranslation{}, errors.New("модель вернула одинаковые или пустые варианты")
	}
	return out, nil
}
