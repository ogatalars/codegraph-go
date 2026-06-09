package store

const schema = `
CREATE TABLE IF NOT EXISTS files (
    id         INTEGER PRIMARY KEY,
    path       TEXT UNIQUE NOT NULL,
    pkg        TEXT NOT NULL,
    indexed_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS symbols (
    id        INTEGER PRIMARY KEY,
    file_id   INTEGER NOT NULL REFERENCES files(id),
    name      TEXT NOT NULL,
    fqn       TEXT UNIQUE NOT NULL,
    kind      TEXT NOT NULL,
    line      INTEGER NOT NULL,
    col       INTEGER NOT NULL,
    signature TEXT,
    docstring TEXT
);

CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_fqn  ON symbols(fqn);
CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_id);

CREATE TABLE IF NOT EXISTS edges (
    id       INTEGER PRIMARY KEY,
    from_fqn TEXT NOT NULL,
    to_fqn   TEXT NOT NULL,
    kind     TEXT NOT NULL,
    file_id  INTEGER NOT NULL REFERENCES files(id),
    line     INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_edges_from ON edges(from_fqn);
CREATE INDEX IF NOT EXISTS idx_edges_to   ON edges(to_fqn);
`
