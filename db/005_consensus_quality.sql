-- Equivalence is curated per fragment and language pair. The application normalizes
-- typography, while moderators explicitly approve semantic equivalence.
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS normalized_text text;
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS canonical_id uuid REFERENCES candidates(id);
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS is_correct boolean;
ALTER TABLE translations ADD COLUMN IF NOT EXISTS expert_confirmed boolean NOT NULL DEFAULT false;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS audio_data bytea;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS audio_mime text;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS audio_filename text;
CREATE INDEX IF NOT EXISTS idx_candidate_canonical ON candidates(canonical_id);
CREATE INDEX IF NOT EXISTS idx_answers_latest ON answers(user_id,created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tasks_fragment_pair ON tasks(fragment_id,lang_pair);
-- Preserve existing expert decisions when upgrading.
UPDATE translations tr SET expert_confirmed=true
WHERE EXISTS (SELECT 1 FROM moderation_cases mc WHERE mc.fragment_id=tr.fragment_id
 AND mc.lang_pair=tr.lang_pair AND mc.status='resolved');
-- Mark the demonstration references; imported works require human review.
WITH refs(ba,correct) AS (VALUES
 ('Һаумыһығыҙ!','Здравствуйте!'),('Рәхмәт!','Спасибо!'),
 ('Мин мәктәпкә барам.','Я иду в школу.'),('Бөгөн көн матур.','Сегодня хорошая погода.'),
 ('Мин китап уҡыйым.','Я читаю книгу.'),('Һинең исемең нисек?','Как тебя зовут?'),
 ('Минең дуҫым бар.','У меня есть друг.'),('Беҙ бергә уҡыйбыҙ.','Мы учимся вместе.'),
 ('Хәйерле иртә!','Доброе утро!'),('Һау булығыҙ!','До свидания!'),
 ('Мин башҡорт телен өйрәнәм.','Я изучаю башкирский язык.'),('Был минең китабым.','Это моя книга.'))
UPDATE candidates c SET is_correct=(c.translation=r.correct)
FROM fragments f JOIN materials m ON m.id=f.material_id JOIN refs r ON r.ba=f.text_ba
WHERE c.fragment_id=f.id AND c.lang_pair='ba-ru' AND m.title='Учебные фразы';
UPDATE candidates c SET is_correct=(c.translation=s.reference_translation)
FROM fragment_ai_suggestions s JOIN fragments f ON f.id=s.fragment_id
WHERE c.fragment_id=s.fragment_id AND c.lang_pair=s.lang_pair
 AND c.translation IN (s.reference_translation,s.wrong_translation) AND f.review_status='published';
CREATE OR REPLACE FUNCTION normalize_translation(value text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE AS $$
 SELECT trim(regexp_replace(replace(lower(value),'ё','е'),'[^[:alnum:]]+',' ','g'))
$$;
