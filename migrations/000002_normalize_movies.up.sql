-- The golang-migrate postgres driver sends each migration file as one
-- multi-statement exec; PostgreSQL executes that message atomically by default.
CREATE TABLE genres (
    id BIGSERIAL CONSTRAINT pk_genres PRIMARY KEY,
    name TEXT NOT NULL,
    CONSTRAINT uq_genres_name UNIQUE (name)
);

CREATE TABLE directors (
    id BIGSERIAL CONSTRAINT pk_directors PRIMARY KEY,
    full_name TEXT NOT NULL,
    bio TEXT NULL,
    CONSTRAINT uq_directors_full_name UNIQUE (full_name)
);

CREATE TABLE movie_directors (
    movie_id BIGINT NOT NULL,
    director_id BIGINT NOT NULL,
    CONSTRAINT pk_movie_directors PRIMARY KEY (movie_id, director_id),
    CONSTRAINT fk_movie_directors_movie
        FOREIGN KEY (movie_id) REFERENCES movies (id) ON DELETE CASCADE,
    CONSTRAINT fk_movie_directors_director
        FOREIGN KEY (director_id) REFERENCES directors (id) ON DELETE RESTRICT
);

INSERT INTO genres (name)
SELECT DISTINCT genre
FROM movies;

INSERT INTO directors (full_name)
SELECT DISTINCT director
FROM movies;

ALTER TABLE movies
    ADD COLUMN genre_id BIGINT NULL;

UPDATE movies AS m
SET genre_id = g.id
FROM genres AS g
WHERE g.name = m.genre;

INSERT INTO movie_directors (movie_id, director_id)
SELECT m.id, d.id
FROM movies AS m
JOIN directors AS d ON d.full_name = m.director;

ALTER TABLE movies
    ALTER COLUMN genre_id SET NOT NULL,
    ADD CONSTRAINT fk_movies_genre
        FOREIGN KEY (genre_id) REFERENCES genres (id) ON DELETE RESTRICT;

CREATE INDEX idx_movies_genre_id ON movies (genre_id);
CREATE INDEX idx_movie_directors_director_id ON movie_directors (director_id);
-- movie_id is the leading column of pk_movie_directors and needs no separate index.
DROP INDEX idx_movies_genre;

ALTER TABLE movies
    DROP COLUMN director,
    DROP COLUMN genre;
