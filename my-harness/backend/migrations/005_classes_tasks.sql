CREATE TABLE IF NOT EXISTS classes (
    id          VARCHAR(36) PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    teacher_id  VARCHAR(36)  NOT NULL,
    invite_code VARCHAR(6)   NOT NULL,
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_classes_invite_code (invite_code),
    INDEX idx_classes_teacher (teacher_id),
    CONSTRAINT fk_classes_teacher FOREIGN KEY (teacher_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS class_members (
    id        VARCHAR(36) PRIMARY KEY,
    class_id  VARCHAR(36)  NOT NULL,
    user_id   VARCHAR(36)  NOT NULL,
    joined_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_class_member (class_id, user_id),
    INDEX idx_cm_class (class_id),
    CONSTRAINT fk_cm_class FOREIGN KEY (class_id) REFERENCES classes(id) ON DELETE CASCADE,
    CONSTRAINT fk_cm_user  FOREIGN KEY (user_id)  REFERENCES users(id)   ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS class_tasks (
    id          VARCHAR(36) PRIMARY KEY,
    class_id    VARCHAR(36)                    NOT NULL,
    paper_id    VARCHAR(36)                    NOT NULL,
    title       VARCHAR(200)                   NOT NULL,
    assigned_by VARCHAR(36)                    NOT NULL,
    due_at      DATETIME(3)                    NULL,
    status      ENUM('active','closed')        NOT NULL DEFAULT 'active',
    created_at  DATETIME(3)                    NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_ct_class (class_id),
    CONSTRAINT fk_ct_class FOREIGN KEY (class_id) REFERENCES classes(id)  ON DELETE CASCADE,
    CONSTRAINT fk_ct_paper FOREIGN KEY (paper_id) REFERENCES papers(id)   ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_submissions (
    id           VARCHAR(36) PRIMARY KEY,
    task_id      VARCHAR(36)           NOT NULL,
    user_id      VARCHAR(36)           NOT NULL,
    question_id  VARCHAR(36)           NOT NULL,
    result       ENUM('pass','fail')   NOT NULL,
    submitted_at DATETIME(3)           NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_ts (task_id, user_id, question_id),
    INDEX idx_ts_task (task_id),
    CONSTRAINT fk_ts_task     FOREIGN KEY (task_id)     REFERENCES class_tasks(id) ON DELETE CASCADE,
    CONSTRAINT fk_ts_user     FOREIGN KEY (user_id)     REFERENCES users(id)       ON DELETE CASCADE,
    CONSTRAINT fk_ts_question FOREIGN KEY (question_id) REFERENCES questions(id)   ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
