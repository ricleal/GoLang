
-- postgresql
DROP TABLE IF EXISTS users;

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    bio TEXT,
    data jsonb
)


-- Text search
ALTER TABLE users
ADD COLUMN ts_bio tsvector
GENERATED ALWAYS AS (to_tsvector('english', bio)) STORED;

CREATE INDEX ts_bio_idx ON users USING GIN (ts_bio);


-- SELECT *
-- FROM users
-- WHERE ts_bio @@ to_tsquery('english', 'tornado');