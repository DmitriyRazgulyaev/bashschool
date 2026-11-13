package service

import (
	"context"
	"strings"

	"bashkir-school/backend/internal/domain"
)

func (s *Service) ImportLiterature(ctx context.Context, p domain.Parameters, in LiteratureUpload) (domain.Result, error) {
	title := strings.TrimSpace(in.Field("title"))
	author := strings.TrimSpace(in.Field("author"))
	license := strings.TrimSpace(in.Field("license"))
	note := strings.TrimSpace(in.Field("copyrightNote"))
	source := strings.TrimSpace(in.FieldDefault("sourceLang", "ba"))
	mode := strings.TrimSpace(in.FieldDefault("segmentMode", "sentence"))
	if mode != "sentence" && mode != "paragraph" {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_MODE", "Доступны sentence и paragraph")
	}
	if in.Field("rightsConfirmed") != "true" {
		return domain.Result{}, domain.Failure(domain.Invalid, "RIGHTS_REQUIRED", "Подтвердите наличие прав или разрешения на использование произведения")
	}
	if len(title) < 2 || len(title) > 200 || len(author) > 200 || len(license) < 3 || len(note) > 500 {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_DATA", "Укажите название, лицензию/разрешение и допустимые поля")
	}
	if !langPattern.MatchString(source) {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_LANGUAGE", "Неправильный код языка")
	}
	if license == "unknown" {
		return domain.Result{}, domain.Failure(domain.Invalid, "RIGHTS_REQUIRED", "Необходимо подтвердить право использования текста")
	}
	text, err := in.Text, in.ReadError
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_FILE", err.Error())
	}
	if strings.TrimSpace(text) == "" {
		return domain.Result{}, domain.Failure(domain.Invalid, "EMPTY_TEXT", "Добавьте текст произведения или TXT-файл")
	}
	fragments := domain.SegmentLiterature(text, mode)
	if len(fragments) == 0 {
		return domain.Result{}, domain.Failure(domain.Invalid, "NO_FRAGMENTS", "Не удалось выделить фрагменты подходящей длины")
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось начать транзакцию")
	}
	defer tx.Rollback(ctx)
	var id string
	var recordCreateTextMaterial CreateTextMaterialRecord
	recordCreateTextMaterial, err = tx.CreateTextMaterial(ctx, title, license, text, author, source, p.Actor.ID, note)
	id = recordCreateTextMaterial.ID

	if err != nil {
		return domain.Result{}, domain.Failure(domain.Conflict, "MATERIAL_EXISTS", "Произведение с таким названием уже есть")
	}
	for _, f := range fragments {
		_, err = tx.CreateTextFragment(ctx, id, f.Text, source, f.Chapter, f.Start, f.End)
		if err != nil {
			return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось сохранить фрагменты")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось сохранить произведение")
	}
	return domain.New(domain.Object{
		"materialId":       id,
		"fragmentsCreated": len(fragments),
		"status":           "draft",
		"message":          "Произведение загружено. Фрагменты скрыты от учеников до модерации.",
	}), nil
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

func (s *Service) LiteratureDrafts(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	pair := p.Filter("langPair", "ba-ru")
	rows1, err := s.store.ListContentDrafts(ctx, pair)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось загрузить черновики")
	}

	arr := []literatureDraft{}
	for _, row := range rows1 {
		var f literatureDraft
		{
			f.ID, f.Text, f.Title, f.Chapter, f.Candidates,
				f.Status, f.SourceLang, f.Correct, f.Incorrect, f.ErrorExplanation = row.ID, row.Text,
				row.Title,
				row.Chapter, row.Candidates, row.Status, row.SourceLang, row.Correct, row.Incorrect, row.ErrorExplanation
		}

		arr = append(arr, f)
	}

	return domain.OK(arr), nil
}

func (s *Service) GenerateLiteratureCandidates(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	pair := strings.TrimSpace(p.Filter("langPair", ""))
	if pair == "" {
		pair = "ba-ru"
	}

	var text, source, target, status string
	recordGetGenerationSource, err := s.store.GetGenerationSource(ctx, p.ID, pair)
	text, source, target, status = recordGetGenerationSource.Text, recordGetGenerationSource.Source, recordGetGenerationSource.Target, recordGetGenerationSource.Status

	if err != nil {
		return domain.Result{}, domain.Failure(domain.NotFound, "NOT_FOUND", "Фрагмент или языковая пара не найдены")
	}
	if status != "draft" {
		return domain.Result{}, domain.Failure(domain.Conflict, "ALREADY_REVIEWED", "Можно генерировать только для черновиков")
	}
	if s.translator == nil || !s.translator.Configured() {
		return domain.Result{}, domain.Failure(domain.Unavailable, "AI_NOT_CONFIGURED", "Добавьте AI_API_KEY в .env и перезапустите api")
	}
	result, err := s.translator.Generate(ctx, text, source, target)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Unavailable, "AI_FAILED", err.Error())
	}
	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка базы данных")
	}
	defer tx.Rollback(ctx)
	// Rechecking status under transaction locks prevents published content being modified by a concurrent request.
	var current string
	{
		var recordLockDraftForGeneration LockDraftForGenerationRecord
		recordLockDraftForGeneration, err = tx.LockDraftForGeneration(ctx, p.ID)
		current = recordLockDraftForGeneration.Current

		if err != nil || current != "draft" {
			return domain.Result{}, domain.Failure(domain.Conflict, "NOT_DRAFT", "Фрагмент уже модерирован")
		}
	}
	for _, candidate := range []string{result.Correct, result.Incorrect} {
		if _, err = tx.SaveGeneratedCandidate(ctx, p.ID, pair, candidate); err != nil {
			return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось сохранить варианты")
		}
	}
	if _, err = tx.SaveAISuggestion(ctx, p.ID, pair, result.Correct, result.Incorrect, s.translator.Model(), result.ErrorExplanation); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось сохранить предложения ИИ")
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка фиксации")
	}
	return domain.OK(domain.Object{
		"correct":          result.Correct,
		"incorrect":        result.Incorrect,
		"errorExplanation": result.ErrorExplanation,
		"status":           "draft",
		"warning":          "Варианты ИИ требуют проверки администратором до публикации",
	}), nil
}

func (s *Service) PublishLiteratureFragment(ctx context.Context, p domain.Parameters, in PublishLiteratureFragmentInput) (domain.Result, error) {

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка транзакции")
	}
	defer tx.Rollback(ctx)
	var reviewStatus string
	{
		var recordLockDraftForPublication LockDraftForPublicationRecord
		recordLockDraftForPublication, err = tx.LockDraftForPublication(ctx, p.ID)
		reviewStatus = recordLockDraftForPublication.ReviewStatus

		if err != nil {
			return domain.Result{}, domain.Failure(domain.NotFound, "NOT_FOUND", "Фрагмент не найден")
		}
	}
	if reviewStatus != "draft" {
		return domain.Result{}, domain.Failure(domain.Conflict, "NOT_DRAFT", "Фрагмент уже обработан")
	}
	var count int
	var recordCountReviewedVariants CountReviewedVariantsRecord
	recordCountReviewedVariants, err = tx.CountReviewedVariants(ctx, p.ID, in.LanguagePair)
	count = recordCountReviewedVariants.Count

	if err != nil || count < 2 {
		return domain.Result{}, domain.Failure(domain.Conflict, "NEEDS_REVIEW", "Сохраните и проверьте правильный и ошибочный варианты перевода перед публикацией")
	}
	tag, err := tx.PublishReviewedFragment(ctx, p.ID, p.Actor.ID)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось опубликовать")
	}
	if tag.RowsAffected() == 0 {
		return domain.Result{}, domain.Failure(domain.Conflict, "NOT_DRAFT", "Фрагмент уже обработан")
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось опубликовать")
	}
	return domain.OK(domain.Object{"status": "published", "message": "Фрагмент опубликован. Перевод считается предложенным, а не экспертно подтверждённым."}), nil
}
func (s *Service) RejectLiteratureFragment(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	tag, err := s.store.RejectDraft(ctx, p.ID, p.Actor.ID)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось отклонить")
	}
	if tag.RowsAffected() == 0 {
		return domain.Result{}, domain.Failure(domain.Conflict, "NOT_DRAFT", "Фрагмент уже обработан")
	}
	return domain.OK(domain.Object{"status": "rejected"}), nil
}
