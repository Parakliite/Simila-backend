-- name: GetMovies :many

SELECT *
FROM movies
ORDER BY created_at;


-- name: GetMovie :one

SELECT *
FROM movies
WHERE $1 = id;
