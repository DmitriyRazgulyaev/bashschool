package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) CreateTextMaterial(ctx context.Context,

	title string, license string, text string, author string, source string, id string, note string) (service.CreateTextMaterialRecord, error) {
	var record service.CreateTextMaterialRecord
	err := q.db.QueryRow(ctx, `INSERT INTO materials(title, kind, license, BODY, author, source_lang, uploaded_by, copyright_note)
VALUES($1,'text',$2,$3,$4,$5,$6,$7) RETURNING id`, title, license, text,

		author, source, id, note).Scan(&record.ID)
	return record, persistenceError(err)
}

func (q *Queries) CreateTextFragment(ctx context.Context, id string, text string, source string, chapter string, start int, end int) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO fragments(material_id, text_ba, source_lang, chapter, start_offset, end_offset, review_status)
VALUES($1,$2,$3,$4,$5,$6,'draft')`, id, text, source, chapter, start, end)
}

func (q *Queries) ListContentDrafts(ctx context.Context, pair string) ([]service.ListContentDraftsRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT f.id,
       f.text_ba,
       m.title,
       COALESCE(f.chapter, ''),
    (SELECT count(*)
     FROM candidates ca
     WHERE ca.fragment_id = f.id
         AND ca.lang_pair = $1),f.review_status,
                                f.source_lang,
                                COALESCE(
                                             (SELECT ca.translation
                                              FROM candidates ca
                                              WHERE ca.fragment_id = f.id
                                                  AND ca.lang_pair = $1
                                                  AND ca.is_seed
                                                  AND ca.is_correct = TRUE
                                              LIMIT 1),ai.reference_translation),
                                COALESCE(
                                             (SELECT ca.translation
                                              FROM candidates ca
                                              WHERE ca.fragment_id = f.id
                                                  AND ca.lang_pair = $1
                                                  AND ca.is_seed
                                                  AND ca.is_correct = FALSE
                                              LIMIT 1),ai.wrong_translation),
                                ai.error_explanation
FROM fragments f
JOIN materials m ON m.id = f.material_id
LEFT JOIN fragment_ai_suggestions ai ON ai.fragment_id = f.id
AND ai.lang_pair = $1
WHERE f.review_status = 'draft'
ORDER BY f.created_at DESC
LIMIT 150`,

		pair)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListContentDraftsRecord{}
	for rows.Next() {
		var record service.ListContentDraftsRecord
		if err = rows.Scan(&record.ID, &record.Text, &record.Title, &record.Chapter, &record.Candidates, &record.Status, &record.SourceLang, &record.Correct, &record.Incorrect, &record.ErrorExplanation); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) GetGenerationSource(ctx context.Context, id string, pair string) (service.GetGenerationSourceRecord, error) {
	var record service.GetGenerationSourceRecord
	err := q.db.QueryRow(ctx, `SELECT f.text_ba,
       f.source_lang,
       lp.target_lang,
       f.review_status
FROM fragments f
JOIN language_pairs lp ON lp.code = $2
AND lp.source_lang = f.source_lang
AND lp.active = TRUE
WHERE f.id = $1`, id, pair).Scan(&record.Text, &record.Source, &record.Target, &record.Status)
	return record, persistenceError(err)
}

func (q *Queries) LockDraftForGeneration(ctx context.Context, id string) (service.LockDraftForGenerationRecord, error) {
	var record service.LockDraftForGenerationRecord
	err := q.db.QueryRow(ctx, `SELECT review_status
FROM fragments
WHERE id = $1
    FOR
    UPDATE`, id).Scan(&record.Current)
	return record, persistenceError(err)
}

func (q *Queries) SaveGeneratedCandidate(ctx context.Context, id string, pair string, candidate string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO candidates(fragment_id, lang_pair, TRANSLATION, is_seed)
VALUES($1,$2,$3,TRUE) ON CONFLICT DO NOTHING`, id, pair, candidate)
}

func (q *Queries) SaveAISuggestion(ctx context.Context, id string, pair string, correct string, incorrect string, value string, errorExplanation string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO fragment_ai_suggestions(fragment_id, lang_pair, reference_translation, wrong_translation, model_name, error_explanation)
VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(fragment_id, lang_pair) DO
UPDATE
SET reference_translation = EXCLUDED.reference_translation,
    wrong_translation = EXCLUDED.wrong_translation,
    error_explanation = EXCLUDED.error_explanation,
    model_name = EXCLUDED.model_name,
    created_at = now()`, id, pair, correct, incorrect, value, errorExplanation)
}

func (q *Queries) LockDraftForPublication(ctx context.Context, id string) (service.LockDraftForPublicationRecord, error) {
	var record service.LockDraftForPublicationRecord
	err := q.db.QueryRow(ctx, `SELECT review_status
FROM fragments
WHERE id = $1
    FOR
    UPDATE`, id).Scan(&record.ReviewStatus)
	return record, persistenceError(err)
}

func (q *Queries) CountReviewedVariants(ctx context.Context, id string, languagePair string) (service.CountReviewedVariantsRecord, error) {
	var record service.CountReviewedVariantsRecord
	err := q.db.QueryRow(ctx, `SELECT count(*)
FROM candidates c
JOIN fragments f ON f.id = c.fragment_id
JOIN language_pairs lp ON lp.code = c.lang_pair
AND lp.source_lang = f.source_lang
AND lp.active
WHERE c.fragment_id = $1
    AND c.lang_pair = $2
    AND c.is_seed
    AND c.is_correct IS NOT NULL
HAVING bool_or(c.is_correct)
AND bool_or(NOT c.is_correct)`, id, languagePair).Scan(&record.Count)
	return record, persistenceError(err)
}

func (q *Queries) PublishReviewedFragment(ctx context.Context, id string, id1 string) (service.Command, error) {
	return q.exec(ctx, `UPDATE fragments
SET review_status = 'published',
    review_author = $2,
    reviewed_at = now()
WHERE id = $1
    AND review_status = 'draft'`, id, id1)
}

func (q *Queries) RejectDraft(ctx context.Context, id string, id1 string) (service.Command, error) {
	return q.exec(ctx, `UPDATE fragments
SET review_status = 'rejected',
    review_author = $2,
    reviewed_at = now()
WHERE id = $1
    AND review_status = 'draft'`, id, id1)
}
