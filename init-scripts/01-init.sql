CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT
);

INSERT INTO users (name)
SELECT CONCAT('user_', num) as name
FROM (
    SELECT generate_series(1, 1000) as num
);

CREATE INDEX idx_id ON users(id);