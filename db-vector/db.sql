
-- postgresql
DROP TABLE IF EXISTS users;


CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL,
	firstname text NULL,
	lastname text NULL,
	email text NOT NULL,
    data jsonb
);

CREATE INDEX users_organization_id_idx ON users (organization_id);


-- Text search
ALTER TABLE users
ADD COLUMN search_vector_name tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(firstname, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(lastname, '')), 'B')
) STORED;

CREATE INDEX users_search_vector_name_idx ON users USING GIN (search_vector_name);

ALTER TABLE users
ADD COLUMN search_vector_email tsvector GENERATED ALWAYS AS (
    to_tsvector('english', email)
) STORED;

CREATE INDEX users_search_vector_email_idx ON users USING GIN (search_vector_email);

-- Examples of queries

-- name search
SELECT * FROM users WHERE search_vector_name @@ to_tsquery('english', 'John');

-- name search - and
SELECT * FROM users WHERE search_vector_name @@ to_tsquery('english', 'John & Theodore');

-- name search - or
SELECT * FROM users WHERE search_vector_name @@ to_tsquery('english', 'John | Theodore');

-- name substring search
SELECT * FROM users WHERE search_vector_name @@ to_tsquery('english', 'Jo:*');

-- email search
SELECT * FROM users WHERE search_vector_email @@ to_tsquery('english', 'jonh');