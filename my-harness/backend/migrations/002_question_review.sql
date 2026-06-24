ALTER TABLE questions
  ADD COLUMN status      ENUM('pending_review','approved','rejected') NOT NULL DEFAULT 'approved' AFTER source,
  ADD COLUMN category    VARCHAR(50) NOT NULL DEFAULT 'unknown' AFTER status,
  ADD COLUMN confidence  DECIMAL(5,4) NOT NULL DEFAULT 0.0000 AFTER category,
  ADD COLUMN review_note TEXT NOT NULL DEFAULT '' AFTER confidence,
  ADD COLUMN reviewed_by VARCHAR(36) NULL AFTER review_note,
  ADD COLUMN reviewed_at DATETIME(3) NULL AFTER reviewed_by,
  ADD INDEX idx_questions_status (status);
