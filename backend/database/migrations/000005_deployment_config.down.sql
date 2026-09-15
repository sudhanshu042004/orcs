ALTER TABLE deployments DROP CONSTRAINT IF EXISTS deployments_status_check;

UPDATE deployments SET status = 'success' WHERE status = 'deployed';
UPDATE deployments SET status = 'building' WHERE status IN ('queued', 'pending');

ALTER TABLE deployments
    ADD CONSTRAINT deployments_status_check CHECK (status IN ('draft', 'building', 'failed', 'success'));

ALTER TABLE deployments
    DROP COLUMN IF EXISTS stack,
    DROP COLUMN IF EXISTS install_cmd,
    DROP COLUMN IF EXISTS build_cmd,
    DROP COLUMN IF EXISTS run_cmd;
