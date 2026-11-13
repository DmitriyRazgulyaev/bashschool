package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) CreateClassStudent(ctx context.Context, value string,

	value1 string, value2 string, languagePair string, id string) (service.CreateClassStudentRecord, error) {
	var record service.CreateClassStudentRecord
	err := q.db.QueryRow(ctx, `INSERT INTO users(email, password_hash, display_name, language_pair, school_id)
SELECT lower($1),
       $2,
       $3,
       $4,
       school_id
FROM classes
WHERE id = $5 RETURNING id`, value, value1, value2, languagePair, id).Scan(

		&record.ID)
	return record, persistenceError(err)
}

func (q *Queries) EnrollCreatedStudent(ctx context.Context, idArg string, id string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO class_members(class_id, user_id)
VALUES($1,$2)`, idArg, id)
}

func (q *Queries) CreateSchool(ctx context.Context, value string) (service.CreateSchoolRecord, error) {
	var record service.CreateSchoolRecord
	err := q.db.QueryRow(ctx, `INSERT INTO schools(name)
VALUES($1) RETURNING id`, value).Scan(&record.ID)
	return record, persistenceError(err)
}
