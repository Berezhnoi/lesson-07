CREATE TABLE movies (
    id BIGSERIAL CONSTRAINT pk_movies PRIMARY KEY,
    title TEXT NOT NULL,
    director TEXT NOT NULL,
    country TEXT NOT NULL,
    genre TEXT NOT NULL,
    release_year INT NULL,
    synopsis TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_movies_genre ON movies (genre);
