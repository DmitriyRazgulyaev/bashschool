package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) ListMaterials(ctx context.Context) ([]service.ListMaterialsRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT id,
       title,
       kind,
       license,
       LEFT(COALESCE(BODY, ''), 180),
       created_at
FROM materials
ORDER BY created_at DESC
LIMIT 100`)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListMaterialsRecord{}
	for rows.Next() {
		var record service.ListMaterialsRecord
		if err = rows.Scan(&record.ID, &record.Title, &record.Kind, &record.License, &record.Body, &record.Created); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) ListPublishedFragments(ctx context.Context) ([]service.ListPublishedFragmentsRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT f.id,
       f.text_ba,
       m.title,
       COALESCE(f.chapter, '')
FROM fragments f
JOIN materials m ON m.id = f.material_id
WHERE f.review_status = 'published'
ORDER BY f.created_at DESC
LIMIT 100`)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListPublishedFragmentsRecord{}
	for rows.Next() {
		var record service.ListPublishedFragmentsRecord
		if err = rows.Scan(&record.ID, &record.Text, &record.Title, &record.Chapter); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) CreateMaterial(ctx context.Context, title string, materialType string, license string, description string, sourceURL string) (service.CreateMaterialRecord, error) {
	var record service.CreateMaterialRecord
	err := q.db.QueryRow(ctx, `INSERT INTO materials(title, kind, license, BODY, source_url)
VALUES($1,$2,$3,$4,$5) RETURNING id`, title, materialType, license, description, sourceURL).Scan(&record.ID)
	return record, persistenceError(err)
}

func (q *Queries) CreateFragment(ctx context.Context, materialID string, textBa string, sourceLang string, chapter string, startOffset *int, endOffset *int, startTimeMs *int64, endTimeMs *int64) (service.CreateFragmentRecord, error) {
	var record service.CreateFragmentRecord
	err := q.db.QueryRow(ctx, `INSERT INTO fragments(material_id, text_ba, source_lang, chapter, start_offset, end_offset, start_ms, end_ms, review_status)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,'draft') RETURNING id`, materialID, textBa, sourceLang, chapter, startOffset, endOffset, startTimeMs, endTimeMs).Scan(&record.ID)
	return record, persistenceError(err)
}

func (q *Queries) GetMaterial(ctx context.Context, id string) (service.GetMaterialRecord, error) {
	var record service.GetMaterialRecord
	err := q.db.QueryRow(ctx, `SELECT id,
       title,
       kind,
       license,
       COALESCE(BODY, '')
FROM materials
WHERE id = $1`, id).Scan(&record.ID, &record.Title, &record.Kind, &record.License, &record.Description)
	return record, persistenceError(err)
}

func (q *Queries) GetPublishedFragment(ctx context.Context, id string) (service.GetPublishedFragmentRecord, error) {
	var record service.GetPublishedFragmentRecord
	err := q.db.QueryRow(ctx, `SELECT f.id,
       f.text_ba,
       m.title,
       COALESCE(f.chapter, ''),
       f.start_offset,
       f.end_offset,
       f.start_ms,
       f.end_ms
FROM fragments f
JOIN materials m ON m.id = f.material_id
WHERE f.id = $1
    AND f.review_status = 'published'`, id).Scan(&record.ID, &record.Text, &record.Title, &record.Chapter, &record.Start, &record.End, &record.StartMs, &record.EndMs)
	return record, persistenceError(err)
}
