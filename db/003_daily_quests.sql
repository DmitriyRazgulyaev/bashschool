-- Repeatable migration: daily challenges and multiple translation exercises per day.
-- Preserve completed tasks and old progress when upgrading from previous releases.

ALTER TABLE tasks
DROP CONSTRAINT IF EXISTS tasks_user_id_lang_pair_assigned_date_key;

CREATE UNIQUE INDEX IF NOT EXISTS tasks_one_pending_per_day ON tasks(user_id, lang_pair, assigned_date)
WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS tasks_completed_at_user_idx ON tasks(user_id, completed_at)
WHERE status = 'completed';

CREATE TABLE IF NOT EXISTS daily_quests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    quest_date date NOT NULL DEFAULT CURRENT_DATE,
    code text NOT NULL CHECK (code IN ('three_translations','five_translations','one_correction')),
    rewarded_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, quest_date, code)
);

CREATE INDEX IF NOT EXISTS daily_quests_user_date_idx ON daily_quests(user_id, quest_date);
