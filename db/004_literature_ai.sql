-- Existing material and exercises are not changed by this migration.
ALTER TABLE materials ADD COLUMN IF NOT EXISTS author text NOT NULL DEFAULT '';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS source_lang text NOT NULL DEFAULT 'ba';
ALTER TABLE materials ADD COLUMN IF NOT EXISTS uploaded_by uuid REFERENCES users(id);
ALTER TABLE materials ADD COLUMN IF NOT EXISTS copyright_note text NOT NULL DEFAULT '';
ALTER TABLE fragments ADD COLUMN IF NOT EXISTS review_status text NOT NULL DEFAULT 'published'
    CHECK (review_status IN ('draft','published','rejected'));
ALTER TABLE fragments ADD COLUMN IF NOT EXISTS review_author uuid REFERENCES users(id);
ALTER TABLE fragments ADD COLUMN IF NOT EXISTS reviewed_at timestamptz;
CREATE INDEX IF NOT EXISTS idx_fragment_review ON fragments(review_status,source_lang);
CREATE TABLE IF NOT EXISTS fragment_ai_suggestions (
    fragment_id uuid NOT NULL REFERENCES fragments(id) ON DELETE CASCADE,
    lang_pair text NOT NULL REFERENCES language_pairs(code),
    reference_translation text NOT NULL,
    wrong_translation text NOT NULL,
    model_name text NOT NULL,
    error_explanation text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY(fragment_id,lang_pair)
);
