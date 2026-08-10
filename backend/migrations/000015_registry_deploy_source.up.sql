ALTER TABLE projects
    ADD COLUMN image_ref TEXT;

ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS projects_deploy_mode_check;

ALTER TABLE projects
    ADD CONSTRAINT projects_deploy_mode_check
    CHECK (deploy_mode IN ('dockerfile', 'compose', 'static', 'image'));

ALTER TABLE projects
    ADD CONSTRAINT projects_image_source_check
    CHECK (
        (deploy_mode = 'image' AND image_ref IS NOT NULL AND BTRIM(image_ref) <> '')
        OR
        (deploy_mode <> 'image' AND image_ref IS NULL)
    );
