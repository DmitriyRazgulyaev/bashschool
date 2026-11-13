package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) ListConfirmedTranslations(ctx context.Context, value string) ([]service.ListConfirmedTranslationsRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT t.id,
       t.fragment_id,
       f.text_ba,
       t.lang_pair,
       t.translation,
       t.confirmed,
       t.status,
       m.title,
       COALESCE(f.chapter, ''),
       t.updated_at
FROM translations t
JOIN fragments f ON f.id = t.fragment_id
JOIN materials m ON m.id = f.material_id
WHERE t.status = 'confirmed'
    AND ($1 = ''
         OR t.lang_pair = $1)
ORDER BY t.updated_at DESC
LIMIT 500`, value)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListConfirmedTranslationsRecord{}
	for rows.Next() {
		var record service.ListConfirmedTranslationsRecord
		if err = rows.Scan(&record.ID, &record.FragmentID, &record.Text, &record.LanguagePair, &record.Translation, &record.Count, &record.Status, &record.Title, &record.Chapter, &record.Updated); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) ExportConfirmedTranslations(ctx context.Context, value string) ([]service.ExportConfirmedTranslationsRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT t.fragment_id::text,
       f.text_ba,
       t.lang_pair,
       t.translation,
       jsonb_build_object('materialId', m.id, 'title', m.title, 'chapter', f.chapter, 'startOffset', f.start_offset, 'endOffset', f.end_offset, 'startTimeMs', f.start_ms, 'endTimeMs', f.end_ms)::text,
       t.confirmed,
       t.status
FROM translations t
JOIN fragments f ON f.id = t.fragment_id
JOIN materials m ON m.id = f.material_id
WHERE t.status = 'confirmed'
    AND ($1 = ''
         OR t.lang_pair = $1)
ORDER BY t.updated_at DESC`, value)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ExportConfirmedTranslationsRecord{}
	for rows.Next() {
		var record service.ExportConfirmedTranslationsRecord
		if err = rows.Scan(&record.FragmentID, &record.TextBa, &record.LangPair, &record.Translation, &record.Source, &record.Confirmed, &record.Status); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records,

			record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) GetTranslation(ctx context.Context, id string) (service.GetTranslationRecord, error) {
	var record service.GetTranslationRecord
	err := q.db.QueryRow(ctx, `SELECT t.id,
       t.fragment_id,
       f.text_ba,
       t.lang_pair,
       t.translation,
       t.confirmed,
       t.status,
       m.title,
       COALESCE(f.chapter, '')
FROM translations t
JOIN fragments f ON f.id = t.fragment_id
JOIN materials m ON m.id = f.material_id
WHERE t.id = $1`, id).Scan(&record.ID, &record.FragmentID, &record.Text, &record.LanguagePair, &record.Translation, &record.Count, &record.Status, &record.Title, &record.Chapter)
	return record, persistenceError(err)
}
