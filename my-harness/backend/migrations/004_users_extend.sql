ALTER TABLE users
  ADD COLUMN nickname      VARCHAR(100)                       NOT NULL DEFAULT '' AFTER email,
  ADD COLUMN status        ENUM('active','inactive')          NOT NULL DEFAULT 'active' AFTER role,
  ADD COLUMN deactivated_at DATETIME(3)                       NULL     AFTER status;

ALTER TABLE users DROP COLUMN school_name;
