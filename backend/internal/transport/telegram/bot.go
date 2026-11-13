package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bashkir-school/backend/internal/security"
	"bashkir-school/backend/internal/service"
)

type tgUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Text string `json:"text"`
		From struct {
			ID int64 `json:"id"`
		}
		Chat struct {
			ID int64 `json:"id"`
		}
	} `json:"message"`
	CallbackQuery *struct {
		ID   string `json:"id"`
		Data string `json:"data"`
		From struct {
			ID int64 `json:"id"`
		}
		Message struct {
			Chat struct {
				ID int64 `json:"id"`
			}
		} `json:"message"`
	} `json:"callback_query"`
}

type Bot struct {
	app    *service.Service
	client *http.Client
}

func New(app *service.Service, client *http.Client) *Bot {
	if client == nil {
		client = &http.Client{Timeout: 55 * time.Second}
	}
	return &Bot{app: app, client: client}
}

func (b *Bot) tgCall(token, method string, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", "https://api.telegram.org/bot"+token+"/"+method, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram transport failure")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("telegram %s: %d", method, res.StatusCode)
	}
	return nil
}
func (b *Bot) tgSend(token string, chatID int64, text string, keyboard string) {
	msg := map[string]any{"chat_id": chatID, "text": text}
	if keyboard != "" {
		msg["reply_markup"] = map[string]any{"inline_keyboard": [][]map[string]string{{{"text": "✅ Подтвердить", "callback_data": "confirm:" + keyboard}, {"text": "✏️ Исправить", "callback_data": "correct:" + keyboard}}}}
	}
	if err := b.tgCall(token, "sendMessage", msg); err != nil {
		log.Println("telegram send:", err)
	}
}
func (b *Bot) Handle(ctx context.Context, token string, tg, chat int64, input string, callback bool) {
	input = strings.TrimSpace(input)
	if len(input) > 8000 || len([]rune(input)) > 2000 {
		b.tgSend(token, chat, "Сообщение должно быть не длиннее 2000 символов.", "")
		return
	}
	id, err := b.app.TelegramAccount(ctx, tg)
	if err == service.ErrNotFound {
		if strings.HasPrefix(input, "/start ") {
			code := strings.TrimSpace(strings.TrimPrefix(input, "/start "))
			if err := b.app.LinkTelegram(ctx, tg, code); err == nil {
				b.tgSend(token, chat, "✅ Аккаунт привязан! Напишите /daily для задания.", "")
				return
			}
		}
		b.tgSend(token, chat, "👋 Добро пожаловать! Сначала зарегистрируйтесь на сайте, откройте профиль и отправьте мне команду /start с кодом привязки.", "")
		return
	}
	if err != nil {
		b.tgSend(token, chat, "Ошибка подключения к базе.", "")
		return
	}
	state, task, err := b.app.BotSession(ctx, tg)
	if err != nil {
		log.Println("bot session:", err)
		b.tgSend(token, chat, "Не удалось загрузить состояние. Попробуйте позже.", "")
		return
	}
	saveState := func(state string, task *string) bool {
		if err := b.app.SetBotState(ctx, tg, state, task); err != nil {
			log.Println("bot state:", err)
			b.tgSend(token, chat, "Не удалось сохранить состояние. Попробуйте позже.", "")
			return false
		}
		return true
	}
	if callback {
		action, taskID, ok := strings.Cut(input, ":")
		if !ok || task == nil || taskID != *task {
			b.tgSend(token, chat, "Эта кнопка устарела. Получите задание командой /daily.", "")
			return
		}
		input = action
	}
	switch {
	case input == "/start":
		b.tgSend(token, chat, "Һаумыһығыҙ! 👋\n/daily — следующий перевод\n/quests — цели на сегодня\n/profile — прогресс\n/cancel — отмена", "")
	case input == "/cancel":
		if !saveState("idle", nil) {
			return
		}
		b.tgSend(token, chat, "Задание отменено. /daily — получить задание.", "")
	case input == "/profile":
		xp, streak, err := b.app.BotProfile(ctx, id)
		if err != nil {
			b.tgSend(token, chat, "Не удалось загрузить профиль.", "")
			return
		}
		b.tgSend(token, chat, fmt.Sprintf("🏆 Опыт: %d XP\n🔥 Серия: %d дней", xp, streak), "")
	case input == "/quests":
		total, corrections, err := b.app.BotQuestProgress(ctx, id)
		if err != nil {
			b.tgSend(token, chat, "Не удалось получить ежедневные цели.", "")
			return
		}
		message := "🎯 Ежедневные цели:\n"
		for _, quest := range service.Quests() {
			count := total
			if quest.Code == "one_correction" {
				count = corrections
			}
			if count > quest.Goal {
				count = quest.Goal
			}
			mark := "▫️"
			if count >= quest.Goal {
				mark = "✅"
			}
			message += fmt.Sprintf("%s %s: %d/%d (+%d XP)\n", mark, quest.Title, count, quest.Goal, quest.RewardXP)
		}
		b.tgSend(token, chat, message+"\n/daily — выполнить перевод", "")
	case input == "/daily":
		pair, err := b.app.BotPair(ctx, id)
		if err != nil {
			b.tgSend(token, chat, "Ошибка получения языковой пары.", "")
			return
		}
		if !b.app.ValidPair(ctx, pair) {
			b.tgSend(token, chat, "Языковая пара недоступна. Выберите другую на сайте.", "")
			return
		}
		t, err := b.app.Daily(ctx, id, pair)
		if err != nil {
			b.tgSend(token, chat, "Не удалось найти задание.", "")
			return
		}
		if !saveState("waiting_answer", &t.ID) {
			return
		}
		b.tgSend(token, chat, "📘 Проверьте перевод:\n\n"+t.Text+"\n\nПредложенный вариант:\n"+t.Translation, t.ID)
	case callback && input == "correct" && state == "waiting_answer":
		if !saveState("waiting_correction", task) {
			return
		}
		b.tgSend(token, chat, "✏️ Напишите правильный перевод:", "")
	case callback && input == "confirm" && state == "waiting_answer" && task != nil:
		suggested, err := b.app.BotSuggested(ctx, *task, id)
		if err != nil {
			b.tgSend(token, chat, "Задание не найдено. Получите новое через /daily.", "")
			return
		}
		out, err := b.app.Answer(ctx, id, *task, service.AnswerInput{AnswerType: "confirmation", Translation: suggested, IdempotencyKey: security.UUID()})
		if err == nil {
			if !saveState("idle", nil) {
				return
			}
			b.tgSend(token, chat, fmt.Sprintf("🎉 Ответ сохранён! +%v XP (бонус цели: %v XP). Всего: %v XP\n/daily — следующий перевод", out["xpEarned"], out["questBonusXp"], out["totalXp"]), "")
		} else {
			b.tgSend(token, chat, "Не удалось сохранить ответ. Вернитесь к /daily.", "")
		}
	case !callback && state == "waiting_correction" && task != nil && input != "":
		out, err := b.app.Answer(ctx, id, *task, service.AnswerInput{AnswerType: "correction", Translation: input, IdempotencyKey: security.UUID()})
		if err == nil {
			if !saveState("idle", nil) {
				return
			}
			b.tgSend(token, chat, fmt.Sprintf("✨ Исправление сохранено! +%v XP (бонус цели: %v XP). Всего: %v XP\n/daily — следующий перевод", out["xpEarned"], out["questBonusXp"], out["totalXp"]), "")
		} else {
			b.tgSend(token, chat, "Не удалось сохранить ответ.", "")
		}
	default:
		b.tgSend(token, chat, "Используйте /daily для ежедневного задания или /profile для статистики.", "")
	}
}
func (b *Bot) Run(token string) {
	var offset int64
	for {
		params := url.Values{}
		params.Set("timeout", "35")
		params.Set("offset", fmt.Sprint(offset))
		res, err := b.client.Get("https://api.telegram.org/bot" + token + "/getUpdates?" + params.Encode())
		if err != nil {
			log.Println("telegram poll: transport failure")
			time.Sleep(5 * time.Second)
			continue
		}
		var response struct {
			OK     bool       `json:"ok"`
			Result []tgUpdate `json:"result"`
		}
		err = json.NewDecoder(res.Body).Decode(&response)
		res.Body.Close()
		if err != nil || !response.OK {
			log.Println("telegram response invalid:", err)
			time.Sleep(4 * time.Second)
			continue
		}
		for _, u := range response.Result {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			if u.Message != nil {
				b.Handle(context.Background(), token, u.Message.From.ID, u.Message.Chat.ID, u.Message.Text, false)
			}
			if u.CallbackQuery != nil {
				if err := b.tgCall(token, "answerCallbackQuery", map[string]any{"callback_query_id": u.CallbackQuery.ID}); err != nil {
					log.Println("telegram callback:", err)
				}
				b.Handle(context.Background(), token, u.CallbackQuery.From.ID, u.CallbackQuery.Message.Chat.ID, u.CallbackQuery.Data, true)
			}
		}
	}
}
