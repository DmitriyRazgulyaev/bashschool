package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) CreateUser(ctx context.Context, value string,

	value1 string, displayName string, languagePair string) (service.CreateUserRecord, error) {
	var record service.CreateUserRecord
	err := q.db.QueryRow(ctx, `INSERT INTO users(email, password_hash, display_name, language_pair)
VALUES(lower($1),$2,$3,$4) RETURNING id`, value, value1, displayName, languagePair).Scan(&record.ID)
	return record, persistenceError(err)
}

func (q *Queries) FindUserByEmail(ctx context.Context, value string) (service.FindUserByEmailRecord, error) {
	var record service.FindUserByEmailRecord
	err := q.db.QueryRow(ctx, `SELECT id,
       password_hash,
       ROLE,
       display_name,
       language_pair
FROM users
WHERE email = lower($1)`, value).Scan(&record.ID, &record.PasswordHash, &record.Role, &record.Name, &record.LanguagePair)
	return record, persistenceError(err)
}

func (q *Queries) GetUserProfile(ctx context.Context, id string) (service.GetUserProfileRecord, error) {
	var record service.GetUserProfileRecord
	err := q.db.QueryRow(ctx, `SELECT id,
       display_name,
       email,
       ROLE,
       language_pair,
       xp,
       streak,
       school_id
FROM users
WHERE id = $1`, id).Scan(&record.ID, &record.Name, &record.Email, &record.Role, &record.LanguagePair, &record.XP, &record.Streak, &record.School)
	return record, persistenceError(err)
}

func (q *Queries) UpdateUserProfile(ctx context.Context, displayName *string, languagePair *string, updateSchool bool, schoolID *string, id string) (service.Command, error) {
	return q.exec(ctx, `UPDATE users
SET display_name = COALESCE($1, display_name),
    language_pair = COALESCE($2, language_pair),
    school_id = CASE
                    WHEN $3 THEN $4::UUID
                    ELSE school_id
                END
WHERE id = $5`, displayName, languagePair, updateSchool, schoolID, id)
}

func (q *Queries) CreateTelegramLink(ctx context.Context, code string, id string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO bot_link_codes(code, user_id, expires_at)
VALUES($1,$2,now() + interval '15 minutes')`, code, id)
}

func (q *Queries) FindUserByTelegram(ctx context.Context, tg int64) (service.FindUserByTelegramRecord, error) {
	var record service.FindUserByTelegramRecord
	err := q.db.QueryRow(ctx, `SELECT id
FROM users
WHERE telegram_id = $1`, tg).Scan(&record.ID)
	return record, persistenceError(err)
}

func (q *Queries) ConsumeTelegramLink(ctx context.Context, code string) (service.ConsumeTelegramLinkRecord, error) {
	var record service.ConsumeTelegramLinkRecord
	err := q.db.QueryRow(ctx, `UPDATE bot_link_codes
SET used_at = now()
WHERE code = $1
    AND used_at IS NULL
    AND expires_at > now() RETURNING user_id`, code).Scan(&record.ID)
	return record, persistenceError(err)
}

func (q *Queries) DetachTelegramAccount(ctx context.Context, tg int64) (service.Command, error) {
	return q.exec(ctx, `UPDATE users
SET telegram_id = NULL
WHERE telegram_id = $1`, tg)
}

func (q *Queries) AssignTelegramAccount(ctx context.Context, tg int64, id string) (service.Command, error) {
	return q.exec(ctx, `UPDATE users
SET telegram_id = $1
WHERE id = $2
    AND (telegram_id IS NULL
         OR telegram_id = $1)`, tg, id)
}

func (q *Queries) UserRole(ctx context.Context, userID string) (string, error) {
	var role string
	err := q.db.QueryRow(ctx, `SELECT role FROM users WHERE id = $1`, userID).Scan(&role)
	return role, persistenceError(err)
}
