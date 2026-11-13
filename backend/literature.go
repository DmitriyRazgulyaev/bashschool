package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// All these routes MUST be registered under admin.Use(roles("admin")).
const maxLiteratureBytes = 1 << 20 // 1 MiB, including UTF-8 text
var sentencePattern = regexp.MustCompile(`[^.!?…\n]+[.!?…]*`)

type literatureFragment struct {
	Text    string
	Chapter string
	Start   int
	End     int
}

func segmentLiterature(text string, mode string) []literatureFragment {
	if mode == "paragraph" {
		return segmentParagraphs(text)
	}
	var result []literatureFragment
	// Use UTF-8 byte positions so substring offsets can be used with original bytes.
	paragraph := 0
	for _, match := range sentencePattern.FindAllStringIndex(text, -1) {
		start, end := match[0], match[1]
		raw := text[start:end]
		lead := len(raw) - len(strings.TrimLeft(raw, " \t\r\n"))
		tail := len(raw) - len(strings.TrimRight(raw, " \t\r\n"))
		start += lead
		end -= tail
		if start >= end {
			continue
		}
		v := strings.TrimSpace(text[start:end])
		r := utf8.RuneCountInString(v)
		if r < 12 || r > 450 {
			continue
		}
		// Approximate position of paragraph in original work (lines separated by blank lines).
		paragraph = strings.Count(text[:start], "\n\n") + 1
		result = append(result, literatureFragment{v, fmt.Sprintf("Абзац %d", paragraph), start, end})
		if len(result) >= 500 {
			break
		}
	}
	return result
}

// Paragraph mode preserves the original paragraph's exact byte offsets.
func segmentParagraphs(text string) []literatureFragment {
	out := []literatureFragment{}
	cursor := 0
	for _, raw := range strings.SplitAfter(text, "\n\n") {
		start := cursor
		cursor += len(raw)
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		lead := len(raw) - len(strings.TrimLeft(raw, " \t\r\n"))
		start += lead
		end := start + len(part)
		count := utf8.RuneCountInString(part)
		if count < 12 || count > 1500 {
			continue
		}
		out = append(out, literatureFragment{Text: part, Chapter: fmt.Sprintf("Абзац %d", len(out)+1), Start: start, End: end})
		if len(out) >= 500 {
			break
		}
	}
	return out
}

func readLiteratureForm(c *gin.Context) (string, error) {
	if c.Request.ContentLength > maxLiteratureBytes+16384 {
		return "", fmt.Errorf("файл слишком большой")
	}
	fh, err := c.FormFile("file")
	if err == nil {
		if fh.Size > maxLiteratureBytes {
			return "", fmt.Errorf("размер TXT не должен превышать 1 МБ")
		}
		var f multipart.File
		f, err = fh.Open()
		if err != nil {
			return "", err
		}
		defer f.Close()
		b, readErr := io.ReadAll(io.LimitReader(f, maxLiteratureBytes+1))
		if readErr != nil {
			return "", readErr
		}
		if len(b) > maxLiteratureBytes {
			return "", fmt.Errorf("файл слишком большой")
		}
		if !strings.HasSuffix(strings.ToLower(fh.Filename), ".txt") {
			return "", fmt.Errorf("поддерживается только TXT")
		}
		if !utf8.Valid(b) {
			return "", fmt.Errorf("нужен TXT в кодировке UTF-8")
		}
		return string(bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})), nil
	}
	text := c.PostForm("text")
	if len(text) > maxLiteratureBytes || !utf8.ValidString(text) {
		return "", fmt.Errorf("текст слишком большой или не UTF-8")
	}
	return text, nil
}

func importLiterature(c *gin.Context) {
	title := strings.TrimSpace(c.PostForm("title"))
	author := strings.TrimSpace(c.PostForm("author"))
	license := strings.TrimSpace(c.PostForm("license"))
	note := strings.TrimSpace(c.PostForm("copyrightNote"))
	source := strings.TrimSpace(c.DefaultPostForm("sourceLang", "ba"))
	mode := strings.TrimSpace(c.DefaultPostForm("segmentMode", "sentence"))
	if mode != "sentence" && mode != "paragraph" {
		fail(c, 422, "INVALID_MODE", "Доступны sentence и paragraph")
		return
	}
	if c.PostForm("rightsConfirmed") != "true" {
		fail(c, 422, "RIGHTS_REQUIRED", "Подтвердите наличие прав или разрешения на использование произведения")
		return
	}
	if len(title) < 2 || len(title) > 200 || len(author) > 200 || len(license) < 3 || len(note) > 500 {
		fail(c, 422, "INVALID_DATA", "Укажите название, лицензию/разрешение и допустимые поля")
		return
	}
	if !langPattern.MatchString(source) {
		fail(c, 422, "INVALID_LANGUAGE", "Неправильный код языка")
		return
	}
	if license == "unknown" {
		fail(c, 422, "RIGHTS_REQUIRED", "Необходимо подтвердить право использования текста")
		return
	}
	text, err := readLiteratureForm(c)
	if err != nil {
		fail(c, 422, "INVALID_FILE", err.Error())
		return
	}
	if strings.TrimSpace(text) == "" {
		fail(c, 422, "EMPTY_TEXT", "Добавьте текст произведения или TXT-файл")
		return
	}
	fragments := segmentLiterature(text, mode)
	if len(fragments) == 0 {
		fail(c, 422, "NO_FRAGMENTS", "Не удалось выделить фрагменты подходящей длины")
		return
	}
	ctx := c.Request.Context()
	tx, err := db.Begin(ctx)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось начать транзакцию")
		return
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO materials(title,kind,license,body,author,source_lang,uploaded_by,copyright_note)
 VALUES($1,'text',$2,$3,$4,$5,$6,$7) RETURNING id`, title, license, text, author, source, uid(c), note).Scan(&id)
	if err != nil {
		fail(c, 409, "MATERIAL_EXISTS", "Произведение с таким названием уже есть")
		return
	}
	for _, f := range fragments {
		_, err = tx.Exec(ctx, `INSERT INTO fragments(material_id,text_ba,source_lang,chapter,start_offset,end_offset,review_status)
  VALUES($1,$2,$3,$4,$5,$6,'draft')`, id, f.Text, source, f.Chapter, f.Start, f.End)
		if err != nil {
			fail(c, 500, "DB_ERROR", "Не удалось сохранить фрагменты")
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось сохранить произведение")
		return
	}
	c.JSON(201, gin.H{"materialId": id, "fragmentsCreated": len(fragments), "status": "draft", "message": "Произведение загружено. Фрагменты скрыты от учеников до модерации."})
}

type literatureDraft struct {
	ID               string  `json:"id"`
	Text             string  `json:"text"`
	Title            string  `json:"title"`
	Chapter          string  `json:"chapter"`
	Candidates       int     `json:"candidates"`
	Status           string  `json:"status"`
	SourceLang       string  `json:"sourceLang"`
	Correct          *string `json:"correct"`
	Incorrect        *string `json:"incorrect"`
	ErrorExplanation *string `json:"errorExplanation"`
}

func literatureDrafts(c *gin.Context) {
	pair := c.DefaultQuery("langPair", "ba-ru")
	rows, err := db.Query(c.Request.Context(), `SELECT f.id,f.text_ba,m.title,COALESCE(f.chapter,''),
 (SELECT count(*) FROM candidates ca WHERE ca.fragment_id=f.id AND ca.lang_pair=$1),f.review_status,f.source_lang,
 ai.reference_translation,ai.wrong_translation,ai.error_explanation
 FROM fragments f JOIN materials m ON m.id=f.material_id
 LEFT JOIN fragment_ai_suggestions ai ON ai.fragment_id=f.id AND ai.lang_pair=$1
 WHERE f.review_status='draft'
 ORDER BY f.created_at DESC LIMIT 150`, pair)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось загрузить черновики")
		return
	}
	defer rows.Close()
	arr := []literatureDraft{}
	for rows.Next() {
		var f literatureDraft
		if err := rows.Scan(&f.ID, &f.Text, &f.Title, &f.Chapter, &f.Candidates, &f.Status, &f.SourceLang, &f.Correct, &f.Incorrect, &f.ErrorExplanation); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения черновиков")
			return
		}
		arr = append(arr, f)
	}
	if err = rows.Err(); err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения черновиков")
		return
	}
	jsonOK(c, arr)
}

type AITranslation struct {
	Correct          string `json:"correct"`
	Incorrect        string `json:"incorrect"`
	ErrorExplanation string `json:"errorExplanation"`
}
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func generateAI(ctx context.Context, text, source, target string) (AITranslation, error) {
	key := strings.TrimSpace(os.Getenv("AI_API_KEY"))
	if key == "" {
		return AITranslation{}, errors.New("AI_API_KEY не настроен в .env")
	}
	url := env("AI_API_URL", "https://api.openai.com/v1/chat/completions")
	model := env("AI_MODEL", "gpt-4.1-mini")
	request := map[string]any{"model": model, "temperature": 0.4, "response_format": map[string]string{"type": "json_object"}, "messages": []map[string]string{
		{"role": "system", "content": "You are a language teacher. Output ONLY valid JSON with string keys correct, incorrect, errorExplanation. Translate source text to target language accurately, then make a clearly incorrect but plausible alternative with one meaning-changing error. Do not invent source or metadata. Never include explanation in the translation strings. Never include markdown."},
		{"role": "user", "content": fmt.Sprintf("Source language: %s. Target language: %s. Source text: %q. Produce correct and intentionally incorrect translations, plus explanation of the mistake in Russian.", source, target, text)},
	}}
	b, _ := json.Marshal(request)
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return AITranslation{}, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 32 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return AITranslation{}, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 128*1024))
	if err != nil {
		return AITranslation{}, err
	}
	if res.StatusCode != 200 {
		return AITranslation{}, fmt.Errorf("AI HTTP %d (проверьте ключ, модель и лимиты)", res.StatusCode)
	}
	var chat chatResponse
	if err = json.Unmarshal(data, &chat); err != nil {
		return AITranslation{}, err
	}
	if len(chat.Choices) == 0 {
		return AITranslation{}, errors.New("AI вернул пустой ответ")
	}
	var out AITranslation
	if err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &out); err != nil {
		return AITranslation{}, fmt.Errorf("некорректный JSON модели: %w", err)
	}
	out.Correct = strings.TrimSpace(out.Correct)
	out.Incorrect = strings.TrimSpace(out.Incorrect)
	if out.Correct == "" || out.Incorrect == "" || out.Correct == out.Incorrect || utf8.RuneCountInString(out.Correct) > 2000 || utf8.RuneCountInString(out.Incorrect) > 2000 {
		return AITranslation{}, errors.New("модель вернула одинаковые или пустые варианты")
	}
	return out, nil
}

func generateLiteratureCandidates(c *gin.Context) {
	pair := strings.TrimSpace(c.Query("langPair"))
	if pair == "" {
		pair = "ba-ru"
	}
	ctx := c.Request.Context()
	var text, source, target, status string
	err := db.QueryRow(ctx, `SELECT f.text_ba,f.source_lang,lp.target_lang,f.review_status FROM fragments f JOIN language_pairs lp ON lp.code=$2 AND lp.source_lang=f.source_lang AND lp.active=true WHERE f.id=$1`, c.Param("id"), pair).Scan(&text, &source, &target, &status)
	if err != nil {
		fail(c, 404, "NOT_FOUND", "Фрагмент или языковая пара не найдены")
		return
	}
	if status != "draft" {
		fail(c, 409, "ALREADY_REVIEWED", "Можно генерировать только для черновиков")
		return
	}
	if env("AI_API_KEY", "") == "" {
		fail(c, 503, "AI_NOT_CONFIGURED", "Добавьте AI_API_KEY в .env и перезапустите api")
		return
	}
	result, err := generateAI(ctx, text, source, target)
	if err != nil {
		fail(c, 502, "AI_FAILED", err.Error())
		return
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка базы данных")
		return
	}
	defer tx.Rollback(ctx)
	// Rechecking status under transaction locks prevents published content being modified by a concurrent request.
	var current string
	if err = tx.QueryRow(ctx, `SELECT review_status FROM fragments WHERE id=$1 FOR UPDATE`, c.Param("id")).Scan(&current); err != nil || current != "draft" {
		fail(c, 409, "NOT_DRAFT", "Фрагмент уже модерирован")
		return
	}
	for _, candidate := range []string{result.Correct, result.Incorrect} {
		if _, err = tx.Exec(ctx, `INSERT INTO candidates(fragment_id,lang_pair,translation,is_seed) VALUES($1,$2,$3,true) ON CONFLICT DO NOTHING`, c.Param("id"), pair, candidate); err != nil {
			fail(c, 500, "DB_ERROR", "Не удалось сохранить варианты")
			return
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO fragment_ai_suggestions(fragment_id,lang_pair,reference_translation,wrong_translation,model_name,error_explanation)
 VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(fragment_id,lang_pair) DO UPDATE SET reference_translation=EXCLUDED.reference_translation,wrong_translation=EXCLUDED.wrong_translation,error_explanation=EXCLUDED.error_explanation,model_name=EXCLUDED.model_name,created_at=now()`, c.Param("id"), pair, result.Correct, result.Incorrect, env("AI_MODEL", "gpt-4.1-mini"), result.ErrorExplanation); err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось сохранить предложения ИИ")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка фиксации")
		return
	}
	jsonOK(c, gin.H{"correct": result.Correct, "incorrect": result.Incorrect, "errorExplanation": result.ErrorExplanation, "status": "draft", "warning": "Варианты ИИ требуют проверки администратором до публикации"})
}

func publishLiteratureFragment(c *gin.Context) {
	var in struct {
		LanguagePair string `json:"languagePair" binding:"required"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	ctx := c.Request.Context()
	// Require a human-reviewed AI pair (not merely two arbitrary candidates).
	var count int
	err := db.QueryRow(ctx, `SELECT count(*) FROM fragment_ai_suggestions s
 JOIN candidates correct ON correct.fragment_id=s.fragment_id AND correct.lang_pair=s.lang_pair AND correct.translation=s.reference_translation
 JOIN candidates wrong ON wrong.fragment_id=s.fragment_id AND wrong.lang_pair=s.lang_pair AND wrong.translation=s.wrong_translation
 WHERE s.fragment_id=$1 AND s.lang_pair=$2 AND s.reference_translation<>s.wrong_translation`, c.Param("id"), in.LanguagePair).Scan(&count)
	if err != nil || count == 0 {
		fail(c, 409, "NEEDS_AI_REVIEW", "Сначала сгенерируйте варианты перевода и проверьте их вручную")
		return
	}
	tag, err := db.Exec(ctx, `UPDATE fragments SET review_status='published',review_author=$2,reviewed_at=now() WHERE id=$1 AND review_status='draft'`, c.Param("id"), uid(c))
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось опубликовать")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(c, 409, "NOT_DRAFT", "Фрагмент уже обработан")
		return
	}
	jsonOK(c, gin.H{"status": "published", "message": "Фрагмент опубликован. Перевод считается предложенным, а не экспертно подтверждённым."})
}
func rejectLiteratureFragment(c *gin.Context) {
	tag, err := db.Exec(c.Request.Context(), `UPDATE fragments SET review_status='rejected',review_author=$2,reviewed_at=now() WHERE id=$1 AND review_status='draft'`, c.Param("id"), uid(c))
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось отклонить")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(c, 409, "NOT_DRAFT", "Фрагмент уже обработан")
		return
	}
	jsonOK(c, gin.H{"status": "rejected"})
}
