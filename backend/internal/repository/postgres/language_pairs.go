package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) IsPairActive(ctx context.Context, code string) (

	service.IsPairActiveRecord, error) {
	var record service.IsPairActiveRecord
	err := q.db.QueryRow(ctx, `SELECT EXISTS
    (SELECT 1
     FROM language_pairs
     WHERE code = $1
         AND active = TRUE)`, code).Scan(&record.Ok)
	return record, persistenceError(err)
}

func (q *Queries) ListActivePairs(ctx context.Context) ([]service.ListActivePairsRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT code,
       source_lang,
       target_lang,
       name,
       active
FROM language_pairs
WHERE active = TRUE
ORDER BY code`)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListActivePairsRecord{}

	for rows.Next() {
		var record service.ListActivePairsRecord
		if err = rows.Scan(&record.Code, &record.SourceLang, &record.TargetLang, &record.Name, &record.Active); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) ListAllPairs(ctx context.Context) ([]service.ListAllPairsRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT code,
       source_lang,
       target_lang,
       name,
       active
FROM language_pairs
ORDER BY code`)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListAllPairsRecord{}
	for rows.Next() {
		var record service.ListAllPairsRecord
		if err = rows.Scan(&record.Code, &record.SourceLang, &record.TargetLang, &record.Name, &record.Active); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) CreatePair(ctx context.Context, code string, sourceLang string, targetLang string, name string) (service.CreatePairRecord, error) {
	var record service.CreatePairRecord
	err := q.db.QueryRow(ctx, `INSERT INTO language_pairs(code, source_lang, target_lang, name)
VALUES($1,$2,$3,$4) RETURNING active`, code, sourceLang, targetLang, name).Scan(&record.Active)
	return record, persistenceError(err)
}

func (q *Queries) UpdatePair(ctx context.Context, code string, name *string, active *bool) (service.UpdatePairRecord, error) {
	var record service.UpdatePairRecord
	err := q.db.QueryRow(ctx, `UPDATE language_pairs
SET name = COALESCE($2, name),
    active = COALESCE($3, active)
WHERE code = $1 RETURNING code,
                          source_lang,
                          target_lang,
                          name,
                          active`, code, name, active).Scan(&record.Code, &record.SourceLang, &record.TargetLang, &record.Name, &record.Active)
	return record, persistenceError(err)
}
