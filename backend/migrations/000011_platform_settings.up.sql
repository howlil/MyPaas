-- Platform-level settings (overrides env var defaults at runtime).
CREATE TABLE platform_settings (
    key        VARCHAR(100) PRIMARY KEY,
    value      JSONB NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
