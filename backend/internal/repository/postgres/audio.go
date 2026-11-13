package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) CreateAudioMaterial(ctx context.Context, title string,

	license string, transcript string, source string, id string, data []byte, contentType string, filename string) (service.CreateAudioMaterialRecord, error) {
	var record service.CreateAudioMaterialRecord
	err := q.db.QueryRow(ctx, `INSERT INTO materials(title, kind, license, BODY, source_lang, uploaded_by, audio_data, audio_mime, audio_filename)
VALUES($1,'audio',$2,$3,$4,$5,$6,$7,$8) RETURNING id`,

		title, license, transcript, source, id, data, contentType, filename).Scan(&record.ID)
	return record, persistenceError(err)
}

func (q *Queries) CreateAudioFragment(ctx context.Context, id string, text string, source string, value string, startOffset int, endOffset int, startMS int64, endMS int64) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO fragments(material_id, text_ba, source_lang, chapter, start_offset, end_offset, start_ms, end_ms, review_status)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,'draft')`, id, text, source, value, startOffset, endOffset, startMS, endMS)
}

func (q *Queries) GetAccessibleAudio(ctx context.Context, id string, role string) (service.GetAccessibleAudioRecord, error) {
	var record service.GetAccessibleAudioRecord
	err := q.db.QueryRow(ctx, `SELECT audio_data,
       audio_mime,
       audio_filename
FROM materials m
WHERE m.id = $1
    AND m.audio_data IS NOT NULL
    AND ($2 IN ('admin',
                'moderator')
         OR EXISTS
             (SELECT 1
              FROM fragments f
              WHERE f.material_id = m.id
                  AND f.review_status = 'published'))`, id, role).Scan(&record.Data, &record.ContentType, &record.Name)
	return record, persistenceError(err)
}
