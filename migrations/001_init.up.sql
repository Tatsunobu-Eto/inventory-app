CREATE TABLE IF NOT EXISTS departments (
    id   SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    department_id INTEGER REFERENCES departments(id),
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL DEFAULT '',
    role          TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('sysadmin','admin','user'))
);

CREATE TABLE IF NOT EXISTS items (
    id            SERIAL PRIMARY KEY,
    department_id INTEGER NOT NULL REFERENCES departments(id),
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    owner_id      INTEGER NOT NULL REFERENCES users(id),
    created_by    INTEGER NOT NULL REFERENCES users(id),
    status        TEXT NOT NULL DEFAULT 'private'
                    CHECK (status IN ('private','market','applying','deleted')),
    market_at     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS item_images (
    id        SERIAL PRIMARY KEY,
    item_id   INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS transactions (
    id             SERIAL PRIMARY KEY,
    item_id        INTEGER NOT NULL REFERENCES items(id),
    from_user_id   INTEGER NOT NULL REFERENCES users(id),
    to_user_id     INTEGER NOT NULL REFERENCES users(id),
    from_user_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_items_department_status ON items(department_id, status);
CREATE INDEX idx_items_owner ON items(owner_id);
CREATE INDEX idx_items_market_at ON items(market_at) WHERE status = 'market';
