package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) ListSchools(ctx context.Context) ([]service.ListSchoolsRecord,

	error) {
	rows, err := q.db.Query(ctx, `SELECT id,
       name
FROM schools
ORDER BY name`)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListSchoolsRecord{}
	for rows.Next() {
		var record service.ListSchoolsRecord
		if err = rows.Scan(&record.ID, &record.Name); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) ListUserClasses(ctx context.Context, id string) ([]service.ListUserClassesRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT cl.id,
       cl.name,
       s.name,
       cl.teacher_id
FROM classes cl
JOIN schools s ON s.id = cl.school_id
WHERE cl.teacher_id = $1
    OR EXISTS
        (SELECT 1
         FROM class_members cm
         WHERE cm.class_id = cl.id
             AND cm.user_id = $1)
ORDER BY cl.name`, id)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListUserClassesRecord{}
	for rows.Next() {
		var record service.ListUserClassesRecord
		if err = rows.Scan(&record.ID, &record.Name, &record.School, &record.Teacher); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) CreateClass(ctx context.Context, schoolID string, id string, name string) (service.CreateClassRecord, error) {
	var record service.CreateClassRecord
	err := q.db.QueryRow(ctx, `INSERT INTO classes(school_id, teacher_id, name)
VALUES($1,$2,$3) RETURNING id`, schoolID, id, name).Scan(&record.ID)
	return record, persistenceError(err)
}

func (q *Queries) CanManageClass(ctx context.Context, id string, id1 string, role string) (service.CanManageClassRecord, error) {
	var record service.CanManageClassRecord
	err := q.db.QueryRow(ctx, `SELECT EXISTS
    (SELECT 1
     FROM classes
     WHERE id = $1
         AND (teacher_id = $2
              OR $3 = 'admin'))`, id, id1, role).Scan(&record.Exists)
	return record, persistenceError(err)
}

func (q *Queries) AddClassMember(ctx context.Context, id string, studentID string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO class_members(class_id, user_id)
SELECT $1,
       id
FROM users
WHERE id = $2
    AND ROLE = 'student' ON CONFLICT DO
    UPDATE
    SET user_id = EXCLUDED.user_id`, id, studentID)
}

func (q *Queries) ListClassStudents(ctx context.Context, id string) ([]service.ListClassStudentsRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT u.id,
       u.display_name,
       u.xp,
       u.streak,
    (SELECT count(*)
     FROM tasks t
     WHERE t.user_id = u.id
         AND t.status = 'completed')
FROM class_members cm
JOIN users u ON u.id = cm.user_id
WHERE cm.class_id = $1
ORDER BY u.xp DESC`, id)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListClassStudentsRecord{}
	for rows.Next() {
		var record service.ListClassStudentsRecord
		if err = rows.Scan(&record.ID, &record.Name, &record.XP, &record.Streak, &record.Completed); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) GetClassStatistics(ctx context.Context, id string) (service.GetClassStatisticsRecord, error) {
	var record service.GetClassStatisticsRecord
	err := q.db.QueryRow(ctx, `SELECT count(*),
       count(*) FILTER(
                       WHERE u.last_active_date >= CURRENT_DATE-7),
                COALESCE(sum(
                                 (SELECT count(*)
                                  FROM tasks t
                                  WHERE t.user_id = u.id
                                      AND t.status = 'completed')), 0),
                COALESCE(sum(u.xp), 0)
FROM class_members cm
JOIN users u ON u.id = cm.user_id
WHERE cm.class_id = $1`, id).Scan(&record.N, &record.Active, &record.Completed, &record.XP)
	return record, persistenceError(err)
}
