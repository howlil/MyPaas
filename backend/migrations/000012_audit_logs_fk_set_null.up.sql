ALTER TABLE audit_logs
DROP CONSTRAINT audit_logs_user_id_fkey,
ADD CONSTRAINT audit_logs_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE deployments
DROP CONSTRAINT deployments_triggered_by_user_id_fkey,
ADD CONSTRAINT deployments_triggered_by_user_id_fkey
    FOREIGN KEY (triggered_by_user_id) REFERENCES users(id) ON DELETE SET NULL;

