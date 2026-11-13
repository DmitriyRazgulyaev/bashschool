package main

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

const maxAudioBytes = 20 << 20

var vttTiming = regexp.MustCompile(`(?m)^(\d{2,}:\d{2}:\d{2}\.\d{3}|\d{2}:\d{2}\.\d{3})[ \t]+-->[ \t]+(\d{2,}:\d{2}:\d{2}\.\d{3}|\d{2}:\d{2}\.\d{3})[^\r\n]*\r?$`)

type audioCue struct {
	Text                   string
	StartMS, EndMS         int64
	StartOffset, EndOffset int
}

func timestampMS(value string) (int64, error) {
	parts := strings.Split(value, ":")
	var hours int64
	if len(parts) == 3 {
		v, err := strconv.ParseInt(parts[0], 10, 32)
		if err != nil {
			return 0, err
		}
		hours = v
		parts = parts[1:]
	}
	if len(parts) != 2 {
		return 0, fmt.Errorf("неверный таймкод")
	}
	minutes, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil || minutes > 59 {
		return 0, fmt.Errorf("неверные минуты")
	}
	seconds, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || seconds < 0 || seconds >= 60 {
		return 0, fmt.Errorf("неверные секунды")
	}
	return hours*3600000 + minutes*60000 + int64(seconds*1000+0.5), nil
}

// WebVTT supplies meaningful, human-aligned speech segments. Audio is played
// directly from the preserved file using these exact millisecond coordinates.
func parseWebVTT(body string) ([]audioCue, error) {
	if !utf8.ValidString(body) || !strings.HasPrefix(strings.TrimPrefix(body, "\ufeff"), "WEBVTT") {
		return nil, fmt.Errorf("нужна расшифровка WebVTT в UTF-8")
	}
	matches := vttTiming.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 || len(matches) > 500 || strings.Count(body, "-->") != len(matches) {
		return nil, fmt.Errorf("нужно от 1 до 500 отрезков WebVTT")
	}
	cues := make([]audioCue, 0, len(matches))
	var previousEnd int64
	for _, match := range matches {
		start, err := timestampMS(body[match[2]:match[3]])
		if err != nil {
			return nil, err
		}
		end, err := timestampMS(body[match[4]:match[5]])
		if err != nil || end <= start || start < previousEnd || end > 24*3600000 {
			return nil, fmt.Errorf("таймкоды должны идти по порядку, не пересекаться и укладываться в 24 часа")
		}
		textStart := match[1]
		for textStart < len(body) && (body[textStart] == '\r' || body[textStart] == '\n') {
			textStart++
		}
		textEnd := len(body)
		if i := strings.Index(body[textStart:], "\n\n"); i >= 0 {
			textEnd = textStart + i
		}
		if i := strings.Index(body[textStart:textEnd], "\r\n\r\n"); i >= 0 {
			textEnd = textStart + i
		}
		raw := body[textStart:textEnd]
		text := strings.TrimSpace(raw)
		if text == "" || utf8.RuneCountInString(text) > 2000 {
			return nil, fmt.Errorf("текст отрезка должен содержать 1–2000 символов")
		}
		textStart += len(raw) - len(strings.TrimLeft(raw, " \t\r\n"))
		cues = append(cues, audioCue{text, start, end, textStart, textStart + len(text)})
		previousEnd = end
	}
	return cues, nil
}

func importAudio(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAudioBytes+(2<<20))
	if c.PostForm("rightsConfirmed") != "true" {
		fail(c, 422, "RIGHTS_REQUIRED", "Подтвердите права на аудио и расшифровку")
		return
	}
	title, license, source := strings.TrimSpace(c.PostForm("title")), strings.TrimSpace(c.PostForm("license")), c.DefaultPostForm("sourceLang", "ba")
	if len(title) < 2 || len(title) > 200 || len(license) < 3 || len(license) > 500 || !langPattern.MatchString(source) {
		fail(c, 422, "INVALID_DATA", "Укажите название, лицензию и код языка")
		return
	}
	transcript := c.PostForm("transcript")
	if len(transcript) > maxLiteratureBytes {
		fail(c, 422, "INVALID_TRANSCRIPT", "Расшифровка должна быть до 1 МБ")
		return
	}
	cues, err := parseWebVTT(transcript)
	if err != nil {
		fail(c, 422, "INVALID_TRANSCRIPT", err.Error())
		return
	}
	file, err := c.FormFile("audio")
	if err != nil || file.Size > maxAudioBytes {
		fail(c, 422, "INVALID_AUDIO", "Загрузите MP3, WAV или OGG до 20 МБ")
		return
	}
	reader, err := file.Open()
	if err != nil {
		fail(c, 422, "INVALID_AUDIO", "Не удалось прочитать файл")
		return
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, maxAudioBytes+1))
	if err != nil || len(data) > maxAudioBytes {
		fail(c, 422, "INVALID_AUDIO", "Аудиофайл слишком большой или повреждён")
		return
	}
	contentType := http.DetectContentType(data)
	allowed := map[string]bool{"audio/mpeg": true, "audio/wave": true, "audio/wav": true, "audio/x-wav": true, "application/ogg": true, "audio/ogg": true}
	if !allowed[contentType] {
		fail(c, 422, "INVALID_AUDIO", "Поддерживаются MP3, WAV и OGG с корректным заголовком файла")
		return
	}
	ctx := c.Request.Context()
	tx, err := db.Begin(ctx)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка транзакции")
		return
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO materials(title,kind,license,body,source_lang,uploaded_by,audio_data,audio_mime,audio_filename)
 VALUES($1,'audio',$2,$3,$4,$5,$6,$7,$8) RETURNING id`, title, license, transcript, source, uid(c), data, contentType, filepath.Base(file.Filename)).Scan(&id)
	if err != nil {
		fail(c, 409, "IMPORT_FAILED", "Не удалось создать материал. Проверьте уникальность названия.")
		return
	}
	for i, cue := range cues {
		_, err = tx.Exec(ctx, `INSERT INTO fragments(material_id,text_ba,source_lang,chapter,start_offset,end_offset,start_ms,end_ms,review_status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,'draft')`, id, cue.Text, source, fmt.Sprintf("Отрезок %d", i+1), cue.StartOffset, cue.EndOffset, cue.StartMS, cue.EndMS)
		if err != nil {
			fail(c, 500, "DB_ERROR", "Не удалось сохранить отрезки")
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось сохранить аудио")
		return
	}
	c.JSON(201, gin.H{"materialId": id, "fragmentsCreated": len(cues), "status": "draft"})
}

func materialAudio(c *gin.Context) {
	var data []byte
	var contentType, name string
	// Students can only read audio belonging to a published fragment.
	err := db.QueryRow(c, `SELECT audio_data,audio_mime,audio_filename FROM materials m WHERE m.id=$1 AND m.audio_data IS NOT NULL AND
 ($2 IN ('admin','moderator') OR EXISTS(SELECT 1 FROM fragments f WHERE f.material_id=m.id AND f.review_status='published'))`, c.Param("id"), c.GetString("role")).Scan(&data, &contentType, &name)
	if err != nil {
		fail(c, 404, "NOT_FOUND", "Аудио не найдено")
		return
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": name}))
	c.Header("Cache-Control", "private, max-age=300")
	http.ServeContent(c.Writer, c.Request, name, time.Time{}, bytes.NewReader(data))
}
