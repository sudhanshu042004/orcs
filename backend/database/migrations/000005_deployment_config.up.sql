-- Deployments are now configured up front (stack + commands) and move through a job queue.
ALTER TABLE deployments
    ADD COLUMN IF NOT EXISTS stack TEXT NOT NULL DEFAULT 'react',
    ADD COLUMN IF NOT EXISTS install_cmd TEXT NOT NULL DEFAULT 'npm i',
    ADD COLUMN IF NOT EXISTS build_cmd TEXT NOT NULL DEFAULT 'npm run build',
    ADD COLUMN IF NOT EXISTS run_cmd TEXT NOT NULL DEFAULT '';

-- Status lifecycle: queued -> pending -> deployed | failed
ALTER TABLE deployments DROP CONSTRAINT IF EXISTS deployments_status_check;

UPDATE deployments SET status = 'deployed' WHERE status = 'success';
-- Builds that were in flight (or never started) can no longer be resumed
UPDATE deployments SET status = 'failed' WHERE status IN ('building', 'draft');

ALTER TABLE deployments
    ADD CONSTRAINT deployments_status_check CHECK (status IN ('queued', 'pending', 'deployed', 'failed'));
