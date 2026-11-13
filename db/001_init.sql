CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS schools (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), name text NOT NULL UNIQUE, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS users (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), email text NOT NULL UNIQUE, password_hash text NOT NULL, display_name text NOT NULL, role text NOT NULL DEFAULT 'student' CHECK(role IN ('student','teacher','moderator','admin')), language_pair text NOT NULL DEFAULT 'ba-ru', school_id uuid REFERENCES schools(id), telegram_id bigint UNIQUE, xp int NOT NULL DEFAULT 0, streak int NOT NULL DEFAULT 0, longest_streak int NOT NULL DEFAULT 0, last_active_date date, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS classes (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), school_id uuid NOT NULL REFERENCES schools(id), teacher_id uuid NOT NULL REFERENCES users(id), name text NOT NULL, UNIQUE(school_id,name));
CREATE TABLE IF NOT EXISTS class_members (class_id uuid NOT NULL REFERENCES classes(id) ON DELETE CASCADE, user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE, PRIMARY KEY(class_id,user_id));
CREATE TABLE IF NOT EXISTS materials (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), title text NOT NULL UNIQUE, kind text NOT NULL DEFAULT 'text', license text NOT NULL DEFAULT 'demo', body text, source_url text, created_at timestamptz DEFAULT now());
CREATE TABLE IF NOT EXISTS fragments (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), material_id uuid NOT NULL REFERENCES materials(id), text_ba text NOT NULL, chapter text, start_offset int, end_offset int, start_ms bigint, end_ms bigint, created_at timestamptz DEFAULT now());
CREATE TABLE IF NOT EXISTS candidates (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), fragment_id uuid NOT NULL REFERENCES fragments(id), lang_pair text NOT NULL DEFAULT 'ba-ru', translation text NOT NULL, is_seed boolean NOT NULL DEFAULT false, created_at timestamptz DEFAULT now(), UNIQUE(fragment_id,lang_pair,translation));
CREATE TABLE IF NOT EXISTS tasks (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users(id), fragment_id uuid NOT NULL REFERENCES fragments(id), candidate_id uuid NOT NULL REFERENCES candidates(id), lang_pair text NOT NULL, assigned_date date NOT NULL DEFAULT CURRENT_DATE, status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','completed')), completed_at timestamptz, UNIQUE(user_id,lang_pair,assigned_date));
CREATE TABLE IF NOT EXISTS answers (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), task_id uuid NOT NULL UNIQUE REFERENCES tasks(id), user_id uuid NOT NULL REFERENCES users(id), candidate_id uuid NOT NULL REFERENCES candidates(id), answer_type text NOT NULL CHECK(answer_type IN ('confirmation','correction')), translation text NOT NULL, idempotency_key uuid NOT NULL UNIQUE, created_at timestamptz DEFAULT now());
CREATE TABLE IF NOT EXISTS translations (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), fragment_id uuid NOT NULL REFERENCES fragments(id), lang_pair text NOT NULL, translation text NOT NULL, confirmed int NOT NULL DEFAULT 0, status text NOT NULL CHECK(status IN ('confirmed','expert_review')), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(fragment_id,lang_pair));
CREATE TABLE IF NOT EXISTS moderation_cases (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), fragment_id uuid NOT NULL REFERENCES fragments(id), lang_pair text NOT NULL, status text NOT NULL DEFAULT 'open', selected_translation text, reviewer_id uuid REFERENCES users(id), created_at timestamptz NOT NULL DEFAULT now(), resolved_at timestamptz, UNIQUE(fragment_id,lang_pair));
CREATE TABLE IF NOT EXISTS xp_events (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users(id), reason text NOT NULL, source_id uuid NOT NULL, amount int NOT NULL, created_at timestamptz DEFAULT now(), UNIQUE(user_id,reason,source_id));
CREATE TABLE IF NOT EXISTS achievements (code text PRIMARY KEY, title text NOT NULL, description text NOT NULL, threshold int NOT NULL);
CREATE TABLE IF NOT EXISTS bot_sessions (telegram_id bigint PRIMARY KEY, user_id uuid REFERENCES users(id), state text NOT NULL DEFAULT 'idle', task_id uuid REFERENCES tasks(id), updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS bot_link_codes (code text PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id), expires_at timestamptz NOT NULL, used_at timestamptz);
CREATE INDEX IF NOT EXISTS idx_answers_candidate ON answers(candidate_id);
CREATE INDEX IF NOT EXISTS idx_tasks_user ON tasks(user_id,assigned_date DESC);
CREATE INDEX IF NOT EXISTS idx_translation_status ON translations(status,lang_pair);
INSERT INTO achievements(code,title,description,threshold) VALUES ('first_step','Первый шаг','Выполнить первое задание',1),('apprentice','Знаток языка','Выполнить 10 заданий',10),('mentor','Хранитель слов','Выполнить 30 заданий',30) ON CONFLICT DO NOTHING;
INSERT INTO schools(name) VALUES ('Демонстрационная школа №1'),('Демонстрационная школа №2') ON CONFLICT DO NOTHING;
INSERT INTO materials(title,kind,license,body) VALUES ('Учебные фразы','text','demo / educational','Примеры коротких башкирских предложений для демонстрации.') ON CONFLICT DO NOTHING;
WITH m AS (SELECT id FROM materials WHERE title='Учебные фразы' LIMIT 1), phrases(ba,ru,wrong) AS (VALUES
 ('Һаумыһығыҙ!','Здравствуйте!','До свидания!'),
 ('Рәхмәт!','Спасибо!','Пожалуйста!'),
 ('Мин мәктәпкә барам.','Я иду в школу.','Я иду домой.'),
 ('Бөгөн көн матур.','Сегодня хорошая погода.','Сегодня идёт дождь.'),
 ('Мин китап уҡыйым.','Я читаю книгу.','Я пишу письмо.'),
 ('Һинең исемең нисек?','Как тебя зовут?','Где ты живёшь?'),
 ('Минең дуҫым бар.','У меня есть друг.','У меня есть собака.'),
 ('Беҙ бергә уҡыйбыҙ.','Мы учимся вместе.','Мы отдыхаем вместе.'),
 ('Хәйерле иртә!','Доброе утро!','Доброй ночи!'),
 ('Һау булығыҙ!','До свидания!','Добро пожаловать!'),
 ('Мин башҡорт телен өйрәнәм.','Я изучаю башкирский язык.','Я изучаю английский язык.'),
 ('Был минең китабым.','Это моя книга.','Это мой дом.')
), inserted AS (INSERT INTO fragments(material_id,text_ba,chapter) SELECT m.id,p.ba,'Базовые фразы' FROM phrases p CROSS JOIN m WHERE NOT EXISTS (SELECT 1 FROM fragments f WHERE f.text_ba=p.ba) RETURNING id,text_ba)
INSERT INTO candidates(fragment_id,lang_pair,translation,is_seed)
SELECT f.id,'ba-ru',v.translation,true FROM fragments f JOIN phrases p ON f.text_ba=p.ba CROSS JOIN LATERAL (VALUES (p.ru),(p.wrong)) v(translation) ON CONFLICT DO NOTHING;
