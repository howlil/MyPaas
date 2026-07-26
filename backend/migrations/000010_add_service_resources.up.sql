ALTER TABLE projects ADD COLUMN service_resources JSONB DEFAULT '{}'::jsonb NOT NULL;
