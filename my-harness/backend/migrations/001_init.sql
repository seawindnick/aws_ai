CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    cognito_sub TEXT NOT NULL UNIQUE,
    email       TEXT NOT NULL UNIQUE,
    role        TEXT NOT NULL CHECK (role IN ('student', 'teacher', 'admin')),
    school_name TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS questions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    image_path  TEXT NOT NULL,
    raw_text    TEXT NOT NULL,
    subject     TEXT NOT NULL DEFAULT '',
    topic_tags  TEXT[] NOT NULL DEFAULT '{}',
    source      TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_questions_user_id ON questions(user_id);

CREATE TABLE IF NOT EXISTS errors (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question_id   TEXT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    wrong_count   INT NOT NULL DEFAULT 1,
    last_wrong_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, question_id)
);
CREATE INDEX IF NOT EXISTS idx_errors_user_id ON errors(user_id);

CREATE TABLE IF NOT EXISTS review_records (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question_id TEXT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    reviewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    result      TEXT NOT NULL CHECK (result IN ('pass', 'fail'))
);
CREATE INDEX IF NOT EXISTS idx_review_records_user_id ON review_records(user_id);
