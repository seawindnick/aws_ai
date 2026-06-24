-- 标签表：支持 suggested / confirmed 两种状态
CREATE TABLE IF NOT EXISTS question_tags (
    id          VARCHAR(36) PRIMARY KEY,
    question_id VARCHAR(36) NOT NULL,
    user_id     VARCHAR(36) NOT NULL,
    name        VARCHAR(100) NOT NULL,
    status      ENUM('suggested','confirmed') NOT NULL DEFAULT 'suggested',
    created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_question_tag (question_id, name),
    INDEX idx_qt_question (question_id),
    INDEX idx_qt_user (user_id),
    CONSTRAINT fk_qt_question FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE,
    CONSTRAINT fk_qt_user     FOREIGN KEY (user_id)     REFERENCES users(id)     ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 复习试卷
CREATE TABLE IF NOT EXISTS papers (
    id         VARCHAR(36) PRIMARY KEY,
    user_id    VARCHAR(36) NOT NULL,
    title      VARCHAR(255) NOT NULL,
    status     ENUM('draft','published') NOT NULL DEFAULT 'draft',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_papers_user_id (user_id),
    CONSTRAINT fk_papers_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 试卷题目（含顺序）
CREATE TABLE IF NOT EXISTS paper_questions (
    id          VARCHAR(36) PRIMARY KEY,
    paper_id    VARCHAR(36) NOT NULL,
    question_id VARCHAR(36) NOT NULL,
    position    INT NOT NULL DEFAULT 0,
    created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_paper_question (paper_id, question_id),
    INDEX idx_pq_paper (paper_id),
    CONSTRAINT fk_pq_paper    FOREIGN KEY (paper_id)    REFERENCES papers(id)    ON DELETE CASCADE,
    CONSTRAINT fk_pq_question FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 站内通知
CREATE TABLE IF NOT EXISTS notifications (
    id         VARCHAR(36) PRIMARY KEY,
    user_id    VARCHAR(36) NOT NULL,
    type       VARCHAR(50) NOT NULL,
    title      VARCHAR(255) NOT NULL,
    body       TEXT NOT NULL,
    ref_id     VARCHAR(36) NOT NULL DEFAULT '',
    is_read    TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_notif_user_read (user_id, is_read),
    CONSTRAINT fk_notif_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
