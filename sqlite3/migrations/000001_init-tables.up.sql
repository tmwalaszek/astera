CREATE TABLE IF NOT EXISTS module (
    name TEXT,
    version TEXT,
    mod TEXT,
    info TEXT,
    zip_hash TEXT,
    zip BLOB,

    UNIQUE (version, name)
);