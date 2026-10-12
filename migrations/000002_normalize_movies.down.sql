ALTER TABLE movies
    ADD COLUMN IF NOT EXISTS director TEXT,
    ADD COLUMN IF NOT EXISTS genre TEXT;

UPDATE movies AS m
SET genre = g.name
FROM genres AS g
WHERE g.id = m.genre_id;

-- If a movie has multiple normalized directors, the lowest director id is
-- restored; extra director associations cannot fit in the original flat schema.
-- Movies with no director association cannot be restored to director NOT NULL;
-- the migration fails rather than inventing a director value.
UPDATE movies AS m
SET director = chosen.full_name
FROM (
    SELECT DISTINCT ON (md.movie_id) md.movie_id, d.full_name
    FROM movie_directors AS md
    JOIN directors AS d ON d.id = md.director_id
    ORDER BY md.movie_id, md.director_id
) AS chosen
WHERE chosen.movie_id = m.id;

ALTER TABLE movies
    ALTER COLUMN director SET NOT NULL,
    ALTER COLUMN genre SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_movies_genre ON movies (genre);

DROP INDEX IF EXISTS idx_movie_directors_director_id;
DROP INDEX IF EXISTS idx_movies_genre_id;
ALTER TABLE movies
    DROP CONSTRAINT IF EXISTS fk_movies_genre;
ALTER TABLE movies
    DROP COLUMN IF EXISTS genre_id;
DROP TABLE IF EXISTS movie_directors;
DROP TABLE IF EXISTS directors;
DROP TABLE IF EXISTS genres;
