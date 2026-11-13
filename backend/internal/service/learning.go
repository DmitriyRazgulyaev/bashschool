package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"bashkir-school/backend/internal/domain"
)

type taskRecord struct {
	ID, FragmentID, Text, Translation, Pair, Status string
	BaseXP                                          int
	Source                                          taskSource `json:"source"`
}

type taskSource struct {
	MaterialID  string `json:"materialId"`
	Title       string `json:"title"`
	Chapter     string `json:"chapter"`
	StartOffset *int   `json:"startOffset"`
	EndOffset   *int   `json:"endOffset"`
	StartTimeMS *int64 `json:"startTimeMs"`
	EndTimeMS   *int64 `json:"endTimeMs"`
	HasAudio    bool   `json:"hasAudio"`
}

func (s *Service) loadTaskSource(ctx context.Context, fragmentID string) (taskSource, error) {
	var source taskSource
	recordGetTaskSource, err := s.store.GetTaskSource(ctx, fragmentID)
	source.MaterialID, source.Title, source.Chapter, source.StartOffset, source.EndOffset, source.StartTimeMS,
		source.EndTimeMS, source.HasAudio = recordGetTaskSource.MaterialID,

		recordGetTaskSource.Title, recordGetTaskSource.Chapter, recordGetTaskSource.StartOffset,
		recordGetTaskSource.EndOffset, recordGetTaskSource.StartTimeMS, recordGetTaskSource.EndTimeMS,
		recordGetTaskSource.HasAudio

	return source, err
}

func (s *Service) getDaily(ctx context.Context, userID, pair string) (taskRecord, error) {
	var t taskRecord
	recordFindPendingTask, err := s.store.FindPendingTask(ctx, userID, pair)
	t.ID, t.FragmentID, t.Text, t.Translation,
		t.Pair, t.Status = recordFindPendingTask.ID, recordFindPendingTask.FragmentID,
		recordFindPendingTask.Text, recordFindPendingTask.Translation,

		recordFindPendingTask.Pair,
		recordFindPendingTask.Status

	if err == nil {
		t.BaseXP = 10
		t.Source, err = s.loadTaskSource(ctx, t.FragmentID)
		return t, err
	}
	if err != ErrNotFound {
		return t, err
	}
	var recordAssignUnseenTask AssignUnseenTaskRecord
	recordAssignUnseenTask, err = s.store.AssignUnseenTask(ctx, userID, pair)
	t.ID = recordAssignUnseenTask.ID

	if err != nil {
		return t, err
	}
	var recordGetAssignedTask GetAssignedTaskRecord
	recordGetAssignedTask, err = s.store.GetAssignedTask(ctx, t.ID)
	t.FragmentID, t.Text, t.Translation, t.Pair, t.Status = recordGetAssignedTask.FragmentID, recordGetAssignedTask.Text, recordGetAssignedTask.Translation, recordGetAssignedTask.Pair,

		recordGetAssignedTask.Status

	t.BaseXP = 10
	if err == nil {
		t.Source, err = s.loadTaskSource(ctx, t.FragmentID)
	}
	return t, err
}
func (s *Service) DailyTask(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	pair := p.Filter("langPair", "ba-ru")
	if !s.validPair(ctx, pair) {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_PAIR", "Пара недоступна")
	}
	t, err := s.getDaily(ctx, p.Actor.ID, pair)
	if err != nil {
		if err == ErrNotFound {
			return domain.Result{}, domain.Failure(domain.NotFound, "NO_TASKS_FOR_PAIR", "Доступные задания закончились. Выберите другую пару или дождитесь новых материалов.")
		}
		return domain.Result{}, domain.Failure(domain.Internal, "TASK_ERROR", "Не удалось получить задание")
	}
	return domain.OK(domain.Object{
		"taskId":               t.ID,
		"fragmentId":           t.FragmentID,
		"sourceText":           t.Text,
		"source":               t.Source,
		"suggestedTranslation": t.Translation,
		"languagePair":         t.Pair,
		"status":               t.Status,
		"taskType":             "translation_review",
		"reward":               domain.Object{"baseXp": t.BaseXP},
	}), nil
}
func (s *Service) History(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	limit := filterLimit(p.Filter("limit", "20"), 20, 100)
	rows1, err := s.store.ListTaskHistory(ctx, p.Actor.ID, limit)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка получения истории")
	}

	items := []domain.Object{}
	for _, row := range rows1 {
		var id, text, tr, status string
		var date time.Time
		{
			id, text, tr, status, date = row.ID, row.Text, row.Translation, row.Status, row.Date
			items = append(items, domain.Object{
				"taskId":               id,
				"sourceText":           text,
				"suggestedTranslation": tr,
				"status":               status,
				"date":                 date,
			})
		}
	}
	return domain.OK(domain.Object{"items": items}), nil
}

type AnswerInput struct {
	AnswerType     string `json:"answerType" binding:"required,oneof=confirmation correction"`
	Translation    string `json:"translation" binding:"required,min=1,max=2000"`
	IdempotencyKey string `json:"idempotencyKey" binding:"required,uuid"`
}

func (s *Service) SubmitAnswer(ctx context.Context, p domain.Parameters, in AnswerInput) (domain.Result, error) {
	out, err := s.Answer(ctx, p.Actor.ID, p.ID, in)
	if err != nil {
		return domain.Result{}, err
	}
	return domain.OK(out), nil
}
func (s *Service) Answer(ctx context.Context, userID, taskID string, in AnswerInput) (domain.Object, error) {
	tx, err := s.store.Begin(ctx)
	if err != nil {
		return nil, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка транзакции")
	}
	defer tx.Rollback(ctx)
	var status, candidateID, fragmentID, pair, suggested string
	var recordLockOwnedTask LockOwnedTaskRecord
	recordLockOwnedTask, err = tx.LockOwnedTask(ctx, taskID, userID)
	status, candidateID, fragmentID, pair, suggested = recordLockOwnedTask.Status, recordLockOwnedTask.CandidateID,
		recordLockOwnedTask.FragmentID, recordLockOwnedTask.LanguagePair,

		recordLockOwnedTask.Suggested

	if err != nil {
		return nil, domain.Failure(domain.NotFound, "TASK_NOT_FOUND", "Задание не найдено")
	}
	// Serialize all XP mutations for this user, even if multiple requests arrive together.
	{
		var recordLockUserProgress LockUserProgressRecord
		recordLockUserProgress, err = tx.LockUserProgress(ctx, userID)
		_ = recordLockUserProgress.LockedUser

		if err != nil {
			return nil, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка блокировки прогресса")
		}
	}
	if status == "completed" {
		var key string
		var receipt []byte
		var recordGetAnswerReceipt GetAnswerReceiptRecord
		recordGetAnswerReceipt, err = tx.GetAnswerReceipt(ctx, taskID)
		key, receipt = recordGetAnswerReceipt.Key, recordGetAnswerReceipt.Receipt

		if err == nil && key == in.IdempotencyKey {
			out := domain.Object{}
			if len(receipt) > 0 {
				if err = json.Unmarshal(receipt, &out); err != nil {
					return nil, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось прочитать результат")
				}
			} else {
				var xp, streak int
				{
					var recordGetUserProgress GetUserProgressRecord
					recordGetUserProgress, err = tx.GetUserProgress(ctx, userID)
					xp, streak = recordGetUserProgress.XP, recordGetUserProgress.Streak

					if err != nil {
						return nil, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка чтения прогресса")
					}
				}
				out = domain.Object{
					"status":         "accepted",
					"xpEarned":       0,
					"baseXp":         0,
					"questBonusXp":   0,
					"totalXp":        xp,
					"streak":         streak,
					"unlockedQuests": []string{},
				}
			}
			out["alreadyProcessed"] = true
			return out, nil
		}
		return nil, domain.Failure(domain.Conflict, "TASK_ALREADY_COMPLETED", "Задание уже выполнено")
	}
	if in.AnswerType != "confirmation" && in.AnswerType != "correction" {
		return nil, domain.Failure(domain.Invalid, "INVALID_ANSWER", "Неизвестный тип ответа")
	}
	if len(in.Translation) > 8000 || len([]rune(in.Translation)) > 2000 {
		return nil, domain.Failure(domain.Invalid, "TRANSLATION_TOO_LONG", "Перевод должен содержать не более 2000 символов")
	}
	{
		var recordLockAnswerFragment LockAnswerFragmentRecord
		recordLockAnswerFragment, err = tx.LockAnswerFragment(ctx, fragmentID)
		_ = recordLockAnswerFragment.LockedFragment

		if err != nil {
			return nil, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось заблокировать фрагмент")
		}
	}
	translation := strings.TrimSpace(in.Translation)
	if translation == "" {
		return nil, domain.Failure(domain.Invalid, "EMPTY_TRANSLATION", "Пустой перевод")
	}
	if in.AnswerType == "confirmation" && translation != suggested {
		return nil, domain.Failure(domain.Invalid, "INVALID_CONFIRMATION", "При подтверждении текст должен совпадать с предложенным")
	}
	if in.AnswerType == "correction" {
		var recordUpsertCorrection UpsertCorrectionRecord
		recordUpsertCorrection, err = tx.UpsertCorrection(ctx, fragmentID, pair, translation)
		candidateID = recordUpsertCorrection.CandidateID

		if err != nil {
			return nil, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка сохранения варианта")
		}
	}
	_, err = tx.SaveAnswer(ctx, taskID, userID, candidateID, in.AnswerType, translation, in.IdempotencyKey)
	if err != nil {
		return nil, domain.Failure(domain.Conflict, "DUPLICATE_ANSWER", "Повторный ответ")
	}
	_, err = tx.CompleteTask(ctx, taskID)
	if err != nil {
		return nil, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка обновления задания")
	}
	_, err = tx.RecordTaskReward(ctx, userID, taskID)
	if err != nil {
		return nil, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка начисления XP")
	}
	var xp, streak int
	var recordAdvanceUserProgress AdvanceUserProgressRecord
	recordAdvanceUserProgress, err = tx.AdvanceUserProgress(ctx, userID)
	xp, streak = recordAdvanceUserProgress.XP, recordAdvanceUserProgress.Streak

	if err != nil {
		return nil, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка прогресса")
	}
	if err = s.reconcileTranslation(ctx, tx, fragmentID, pair); err != nil {
		return nil, domain.Failure(domain.Internal, "CONSENSUS_ERROR", "Не удалось согласовать перевод")
	}
	bonusXP, unlockedQuests, err := s.awardDailyQuestBonuses(ctx, tx, userID)
	if err != nil {
		return nil, domain.Failure(domain.Internal, "QUEST_ERROR", "Не удалось обновить ежедневные цели")
	}
	xp += bonusXP
	out := domain.Object{
		"status":         "accepted",
		"xpEarned":       10 + bonusXP,
		"baseXp":         10,
		"questBonusXp":   bonusXP,
		"unlockedQuests": unlockedQuests,
		"totalXp":        xp,
		"streak":         streak,
	}
	receipt, err := json.Marshal(out)
	if err != nil {
		return nil, domain.Failure(domain.Internal, "RECEIPT_ERROR", "Не удалось сохранить результат")
	}
	if _, err = tx.SaveAnswerReceipt(ctx, taskID, receipt); err != nil {
		return nil, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось сохранить результат")
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка фиксации")
	}
	return out, nil
}

func (s *Service) TaskDetail(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	var taskID, fragmentID, text, tr, pair, status string
	recordGetOwnedTask, err := s.store.GetOwnedTask(ctx, p.ID, p.Actor.ID)
	taskID, fragmentID, text, tr, pair, status = recordGetOwnedTask.TaskID, recordGetOwnedTask.FragmentID, recordGetOwnedTask.Text, recordGetOwnedTask.Translation, recordGetOwnedTask.LanguagePair,
		recordGetOwnedTask.Status

	if err != nil {
		return domain.Result{}, domain.Failure(domain.NotFound, "NOT_FOUND", "Задание не найдено")
	}
	return domain.OK(domain.Object{
		"taskId":               taskID,
		"fragmentId":           fragmentID,
		"sourceText":           text,
		"suggestedTranslation": tr,
		"languagePair":         pair,
		"status":               status,
		"reward":               domain.Object{"baseXp": 10},
	}), nil
}
