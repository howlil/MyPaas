ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS projects_image_source_check;

ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS projects_deploy_mode_check;

-- A downgrade cannot preserve registry runtime semantics. Convert image
-- projects into stopped legacy-mode records so the old deploy-mode constraint
-- can be restored without making the migration itself fail.
UPDATE projects
SET deploy_mode = 'dockerfile',
    status = 'stopped',
    active_deployment_id = NULL
WHERE deploy_mode = 'image';

ALTER TABLE projects
    ADD CONSTRAINT projects_deploy_mode_check
    CHECK (deploy_mode IN ('dockerfile', 'compose', 'static'));

ALTER TABLE projects
    DROP COLUMN image_ref;
