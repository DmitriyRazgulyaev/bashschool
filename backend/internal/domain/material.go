package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const MaxLiteratureBytes = 1 << 20 // 1 MiB, including UTF-8 text
var sentencePattern = regexp.MustCompile(`[^.!?…\n]+[.!?…]*`)

type LiteratureFragment struct {
	Text    string
	Chapter string
	Start   int
	End     int
}

func SegmentLiterature(text string, mode string) []LiteratureFragment {
	if mode == "paragraph" {
		return SegmentParagraphs(text)
	}
	var result []LiteratureFragment
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
		result = append(result, LiteratureFragment{v, fmt.Sprintf("Абзац %d", paragraph), start, end})
		if len(result) >= 500 {
			break
		}
	}
	return result
}

// Paragraph mode preserves the original paragraph's exact byte offsets.
func SegmentParagraphs(text string) []LiteratureFragment {
	out := []LiteratureFragment{}
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
		out = append(out, LiteratureFragment{Text: part, Chapter: fmt.Sprintf("Абзац %d", len(out)+1), Start: start, End: end})
		if len(out) >= 500 {
			break
		}
	}
	return out
}

const MaxAudioBytes = 20 << 20

var vttTiming = regexp.MustCompile(`(?m)^(\d{2,}:\d{2}:\d{2}\.\d{3}|\d{2}:\d{2}\.\d{3})[ \t]+-->[ \t]+(\d{2,}:\d{2}:\d{2}\.\d{3}|\d{2}:\d{2}\.\d{3})[^\r\n]*\r?$`)

type AudioCue struct {
	Text                   string
	StartMS, EndMS         int64
	StartOffset, EndOffset int
}

func TimestampMS(value string) (int64, error) {
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
func ParseWebVTT(body string) ([]AudioCue, error) {
	if !utf8.ValidString(body) || !strings.HasPrefix(strings.TrimPrefix(body, "\ufeff"), "WEBVTT") {
		return nil, fmt.Errorf("нужна расшифровка WebVTT в UTF-8")
	}
	matches := vttTiming.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 || len(matches) > 500 || strings.Count(body, "-->") != len(matches) {
		return nil, fmt.Errorf("нужно от 1 до 500 отрезков WebVTT")
	}
	cues := make([]AudioCue, 0, len(matches))
	var previousEnd int64
	for _, match := range matches {
		start, err := TimestampMS(body[match[2]:match[3]])
		if err != nil {
			return nil, err
		}
		end, err := TimestampMS(body[match[4]:match[5]])
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
		cues = append(cues, AudioCue{text, start, end, textStart, textStart + len(text)})
		previousEnd = end
	}
	return cues, nil
}
