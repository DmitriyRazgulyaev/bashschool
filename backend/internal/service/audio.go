package service

import (
	"fmt"
	"strings"

	"context"

	"bashkir-school/backend/internal/domain"
)

func (s *Service) ImportAudio(ctx context.Context, p domain.Parameters, in AudioUpload) (domain.Result, error) {
	if in.Field("rightsConfirmed") != "true" {
		return domain.Result{}, domain.Failure(domain.Invalid, "RIGHTS_REQUIRED", "Подтвердите права на аудио и расшифровку")
	}
	title, license, source := strings.TrimSpace(in.Field("title")), strings.TrimSpace(in.Field("license")), in.FieldDefault("sourceLang", "ba")
	if len(title) < 2 || len(title) > 200 || len(license) < 3 || len(license) > 500 || !langPattern.MatchString(source) {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_DATA", "Укажите название, лицензию и код языка")
	}
	transcript := in.Field("transcript")
	if len(transcript) > domain.MaxLiteratureBytes {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_TRANSCRIPT", "Расшифровка должна быть до 1 МБ")
	}
	cues, err := domain.ParseWebVTT(transcript)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_TRANSCRIPT", err.Error())
	}
	file, err := in.Audio, in.ReadError
	if err != nil || len(file) > domain.MaxAudioBytes {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_AUDIO", "Загрузите MP3, WAV или OGG до 20 МБ")
	}
	data := file
	contentType := in.ContentType
	allowed := map[string]bool{"audio/mpeg": true, "audio/wave": true, "audio/wav": true, "audio/x-wav": true, "application/ogg": true, "audio/ogg": true}
	if !allowed[contentType] {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_AUDIO", "Поддерживаются MP3, WAV и OGG с корректным заголовком файла")
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка транзакции")
	}
	defer tx.Rollback(ctx)
	var id string
	var recordCreateAudioMaterial CreateAudioMaterialRecord
	recordCreateAudioMaterial, err = tx.CreateAudioMaterial(ctx, title, license, transcript, source, p.Actor.ID, data, contentType, in.Filename)
	id = recordCreateAudioMaterial.ID

	if err != nil {
		return domain.Result{}, domain.Failure(domain.Conflict, "IMPORT_FAILED", "Не удалось создать материал. Проверьте уникальность названия.")
	}
	for i, cue := range cues {
		_, err = tx.CreateAudioFragment(ctx, id, cue.Text, source, fmt.Sprintf("Отрезок %d", i+1), cue.StartOffset, cue.EndOffset, cue.StartMS, cue.EndMS)
		if err != nil {
			return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось сохранить отрезки")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось сохранить аудио")
	}
	return domain.New(domain.Object{
		"materialId":       id,
		"fragmentsCreated": len(cues),
		"status":           "draft",
	}), nil
}

func (s *Service) MaterialAudio(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	var data []byte
	var contentType, name string
	// Students can only read audio belonging to a published fragment.
	recordGetAccessibleAudio, err := s.store.GetAccessibleAudio(ctx, p.ID, p.Actor.Role)
	data, contentType, name = recordGetAccessibleAudio.Data, recordGetAccessibleAudio.ContentType, recordGetAccessibleAudio.Name

	if err != nil {
		return domain.Result{}, domain.Failure(domain.NotFound, "NOT_FOUND", "Аудио не найдено")
	}
	return domain.Download(data, name, contentType), nil
}
