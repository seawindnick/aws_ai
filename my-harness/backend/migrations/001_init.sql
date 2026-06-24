CREATE TABLE IF NOT EXISTS users (
    id          VARCHAR(36) PRIMARY KEY,
    cognito_sub VARCHAR(128) NOT NULL UNIQUE,
    email       VARCHAR(255) NOT NULL UNIQUE,
    role        ENUM('student', 'teacher', 'admin') NOT NULL,
    school_name VARCHAR(255) NOT NULL DEFAULT '',
    created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS questions (
    id          VARCHAR(36) PRIMARY KEY,
    user_id     VARCHAR(36) NOT NULL,
    image_path  VARCHAR(512) NOT NULL,
    raw_text    TEXT NOT NULL,
    subject     VARCHAR(100) NOT NULL DEFAULT '',
    topic_tags  JSON NOT NULL,
    source      VARCHAR(100) NOT NULL DEFAULT '',
    created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_questions_user_id (user_id),
    CONSTRAINT fk_questions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS errors (
    id            VARCHAR(36) PRIMARY KEY,
    user_id       VARCHAR(36) NOT NULL,
    question_id   VARCHAR(36) NOT NULL,
    wrong_count   INT NOT NULL DEFAULT 1,
    last_wrong_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_errors_user_question (user_id, question_id),
    INDEX idx_errors_user_id (user_id),
    CONSTRAINT fk_errors_user     FOREIGN KEY (user_id)     REFERENCES users(id)     ON DELETE CASCADE,
    CONSTRAINT fk_errors_question FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS review_records (
    id          VARCHAR(36) PRIMARY KEY,
    user_id     VARCHAR(36) NOT NULL,
    question_id VARCHAR(36) NOT NULL,
    reviewed_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    result      ENUM('pass', 'fail') NOT NULL,
    INDEX idx_review_records_user_id (user_id),
    CONSTRAINT fk_rr_user     FOREIGN KEY (user_id)     REFERENCES users(id)     ON DELETE CASCADE,
    CONSTRAINT fk_rr_question FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
