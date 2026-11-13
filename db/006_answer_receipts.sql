-- Preserve the original response for safe retries after a network interruption.

ALTER TABLE answers ADD COLUMN IF NOT EXISTS RESULT JSONB;
