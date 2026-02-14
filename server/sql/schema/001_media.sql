
-- +goose Up
CREATE TABLE media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    tmdb_id TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL CONSTRAINT min_no_characters_title 
        CHECK (LENGTH(title) BETWEEN 2 AND 500),
    original_title TEXT NOT NULL CONSTRAINT min_no_characters_original_title 
        CHECK (LENGTH(original_title) BETWEEN 2 AND 500),
    poster_path TEXT NOT NULL,
    backdrop_path TEXT NOT NULL,
    overview TEXT NOT NULL,
    release_date DATE NOT NULL,
    runtime INTEGER NOT NULL CHECK (runtime > 0),
    media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'tv'))
);

CREATE TABLE genres (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE CONSTRAINT min_no_characters_genre 
        CHECK (LENGTH(name) BETWEEN 2 AND 100)
);

CREATE TABLE media_genres (
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    genre_id UUID NOT NULL REFERENCES genres (id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES media (id) ON DELETE CASCADE,
    PRIMARY KEY (genre_id, media_id)
);

-- +goose Down
DROP TABLE IF EXISTS media_genres;
DROP TABLE IF EXISTS genres;
DROP TABLE IF EXISTS media;