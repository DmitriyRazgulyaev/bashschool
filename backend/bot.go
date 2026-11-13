package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
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

var tgClient = &http.Client{Timeout: 55 * time.Second}

func tgCall(token, method string, body any) error {
	b, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", "https://api.telegram.org/bot"+token+"/"+method, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := tgClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("telegram %s: %d", method, res.StatusCode)
	}
	return nil
}
func tgSend(token string, chatID int64, text string, keyboard bool) {
	msg := map[string]any{"chat_id": chatID, "text": text}
	if keyboard {
		msg["reply_markup"] = map[string]any{"inline_keyboard": [][]map[string]string{{{"text": "✅ Подтвердить", "callback_data": "confirm"}, {"text": "✏️ Исправить", "callback_data": "correct"}}}}
	}
	if err := tgCall(token, "sendMessage", msg); err != nil {
		log.Println("telegram send:", err)
	}
}
func setBotState(ctx context.Context, tg int64, state string, taskID *string) {
	_, _ = db.Exec(ctx, `INSERT INTO bot_sessions(telegram_id,state,task_id) VALUES($1,$2,$3) ON CONFLICT(telegram_id) DO UPDATE SET state=EXCLUDED.state,task_id=EXCLUDED.task_id,updated_at=now()`, tg, state, taskID)
}
func botSession(ctx context.Context, tg int64) (string, *string) {
	var state string
	var taskID *string
	err := db.QueryRow(ctx, `SELECT state,task_id FROM bot_sessions WHERE telegram_id=$1`, tg).Scan(&state, &taskID)
	if err != nil {
		return "idle", nil
	}
	return state, taskID
}
func handleTelegram(ctx context.Context, token string, tg, chat int64, input string, callback bool) {
	input = strings.TrimSpace(input)
	id, err := accountByTelegram(ctx, tg)
	if err == pgx.ErrNoRows {
		if strings.HasPrefix(input, "/start ") {
			code := strings.TrimSpace(strings.TrimPrefix(input, "/start "))
			if err := bindTelegram(ctx, tg, code); err == nil {
				tgSend(token, chat, "✅ Аккаунт привязан! Напишите /daily для задания.", false)
				return
			}
		}
		tgSend(token, chat, "👋 Добро пожаловать! Сначала зарегистрируйтесь на сайте, откройте профиль и отправьте мне команду /start с кодом привязки.", false)
		return
	}
	if err != nil {
		tgSend(token, chat, "Ошибка подключения к базе.", false)
		return
	}
	state, task := botSession(ctx, tg)
	switch {
	case input == "/start":
		tgSend(token, chat, "Һаумыһығыҙ! 👋\n/daily — следующий перевод\n/quests — цели на сегодня\n/profile — прогресс\n/cancel — отмена", false)
	case input == "/cancel":
		setBotState(ctx, tg, "idle", nil)
		tgSend(token, chat, "Задание отменено. /daily — получить задание.", false)
	case input == "/profile":
		var xp, streak int
		_ = db.QueryRow(ctx, `SELECT xp,streak FROM users WHERE id=$1`, id).Scan(&xp, &streak)
		tgSend(token, chat, fmt.Sprintf("🏆 Опыт: %d XP\n🔥 Серия: %d дней", xp, streak), false)
	case input == "/quests":
		var total, corrections int
		err := db.QueryRow(ctx, `SELECT COUNT(*)::int, COUNT(*) FILTER (WHERE a.answer_type='correction')::int FROM answers a JOIN tasks t ON t.id=a.task_id WHERE a.user_id=$1 AND t.status='completed' AND t.completed_at::date=CURRENT_DATE`, id).Scan(&total, &corrections)
		if err != nil {
			tgSend(token, chat, "Не удалось получить ежедневные цели.", false)
			return
		}
		message := "🎯 Ежедневные цели:\n"
		for _, quest := range dailyQuestDefinitions {
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
		tgSend(token, chat, message+"\n/daily — выполнить перевод", false)
	case input == "/daily":
		var pair string
		if err := db.QueryRow(ctx, `SELECT language_pair FROM users WHERE id=$1`, id).Scan(&pair); err != nil {
			tgSend(token, chat, "Ошибка получения языковой пары.", false)
			return
		}
		if !validPair(ctx, pair) {
			tgSend(token, chat, "Языковая пара недоступна. Выберите другую на сайте.", false)
			return
		}
		t, err := getDaily(ctx, id, pair)
		if err != nil {
			tgSend(token, chat, "Не удалось найти задание.", false)
			return
		}
		setBotState(ctx, tg, "waiting_answer", &t.ID)
		tgSend(token, chat, "📘 Проверьте перевод:\n\n"+t.Text+"\n\nПредложенный вариант:\n"+t.Translation, true)
	case callback && input == "correct" && state == "waiting_answer":
		setBotState(ctx, tg, "waiting_correction", task)
		tgSend(token, chat, "✏️ Напишите правильный перевод:", false)
	case callback && input == "confirm" && state == "waiting_answer" && task != nil:
		var suggested string
		_ = db.QueryRow(ctx, `SELECT c.translation FROM tasks t JOIN candidates c ON c.id=t.candidate_id WHERE t.id=$1`, *task).Scan(&suggested)
		out, status, _, _ := answerCore(ctx, id, *task, answerInput{"confirmation", suggested, randomUUID()})
		if status == 200 {
			setBotState(ctx, tg, "idle", nil)
			tgSend(token, chat, fmt.Sprintf("🎉 Ответ сохранён! +%v XP (бонус цели: %v XP). Всего: %v XP\n/daily — следующий перевод", out["xpEarned"], out["questBonusXp"], out["totalXp"]), false)
		} else {
			tgSend(token, chat, "Не удалось сохранить ответ. Вернитесь к /daily.", false)
		}
	case !callback && state == "waiting_correction" && task != nil && input != "":
		out, status, _, _ := answerCore(ctx, id, *task, answerInput{"correction", input, randomUUID()})
		if status == 200 {
			setBotState(ctx, tg, "idle", nil)
			tgSend(token, chat, fmt.Sprintf("✨ Исправление сохранено! +%v XP (бонус цели: %v XP). Всего: %v XP\n/daily — следующий перевод", out["xpEarned"], out["questBonusXp"], out["totalXp"]), false)
		} else {
			tgSend(token, chat, "Не удалось сохранить ответ.", false)
		}
	default:
		tgSend(token, chat, "Используйте /daily для ежедневного задания или /profile для статистики.", false)
	}
}
func randomUUID() string {
	s := randomCode()
	return fmt.Sprintf("%s-%s-4%s-a%s-%s", s[:8], s[8:12], s[13:16], s[17:20], s[20:32])
}
func runTelegram(token string) {
	var offset int64
	for {
		params := url.Values{}
		params.Set("timeout", "35")
		params.Set("offset", fmt.Sprint(offset))
		res, err := tgClient.Get("https://api.telegram.org/bot" + token + "/getUpdates?" + params.Encode())
		if err != nil {
			log.Println("telegram poll:", err)
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
				handleTelegram(context.Background(), token, u.Message.From.ID, u.Message.Chat.ID, u.Message.Text, false)
			}
			if u.CallbackQuery != nil {
				_ = tgCall(token, "answerCallbackQuery", map[string]any{"callback_query_id": u.CallbackQuery.ID})
				handleTelegram(context.Background(), token, u.CallbackQuery.From.ID, u.CallbackQuery.Message.Chat.ID, u.CallbackQuery.Data, true)
			}
		}
	}
}
