-- Idempotent upgrade for existing PostgreSQL volumes.
-- Existing fragment text_ba stays intact for backward compatibility.
CREATE TABLE IF NOT EXISTS language_pairs (
    code text PRIMARY KEY CHECK (code ~ '^[a-z]{2,3}-[a-z]{2,3}$'),
    source_lang text NOT NULL CHECK(source_lang ~ '^[a-z]{2,3}$'),
    target_lang text NOT NULL CHECK(target_lang ~ '^[a-z]{2,3}$'),
    name text NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK(source_lang <> target_lang),
    CHECK(code = source_lang || '-' || target_lang)
);

INSERT INTO language_pairs(code, source_lang, target_lang, name, active)
VALUES ('ba-ru','ba','ru','Башкирский → Русский',TRUE) ON CONFLICT(code) DO NOTHING;

ALTER TABLE fragments ADD COLUMN IF NOT EXISTS source_lang text NOT NULL DEFAULT 'ba';

CREATE INDEX IF NOT EXISTS idx_fragments_source_lang ON fragments(source_lang);

CREATE INDEX IF NOT EXISTS idx_candidates_pair ON candidates(lang_pair, fragment_id);

-- Existing text_ba column stores original fragment text, regardless of its source language.
-- Pair/fragment consistency is checked by the Go application on writes and task selection.
-- Tie translations, tasks and candidates to configured directions.
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='users_language_pair_fk') THEN
   ALTER TABLE users ADD CONSTRAINT users_language_pair_fk FOREIGN KEY (language_pair) REFERENCES language_pairs(code);
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='candidates_language_pair_fk') THEN
   ALTER TABLE candidates ADD CONSTRAINT candidates_language_pair_fk FOREIGN KEY (lang_pair) REFERENCES language_pairs(code);
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='tasks_language_pair_fk') THEN
   ALTER TABLE tasks ADD CONSTRAINT tasks_language_pair_fk FOREIGN KEY (lang_pair) REFERENCES language_pairs(code);
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='translations_language_pair_fk') THEN
   ALTER TABLE translations ADD CONSTRAINT translations_language_pair_fk FOREIGN KEY (lang_pair) REFERENCES language_pairs(code);
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='moderation_language_pair_fk') THEN
   ALTER TABLE moderation_cases ADD CONSTRAINT moderation_language_pair_fk FOREIGN KEY (lang_pair) REFERENCES language_pairs(code);
 END IF;
END $$;
