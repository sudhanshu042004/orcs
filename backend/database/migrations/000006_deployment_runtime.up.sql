-- A dynamic deployment keeps its container running, and the site handler needs to know
-- which host port that container was published on.
ALTER TABLE deployment_containers
    ADD COLUMN IF NOT EXISTS host_port INTEGER;
