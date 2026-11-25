package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type envelope map[string]any

var maxCursorUUID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

func (app *application) writeJSON(
	w http.ResponseWriter,
	statusCode int,
	data envelope,
	headers http.Header,
) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_, err = w.Write(js)
	return err
}

func (app *application) readIDParam(req *http.Request) (int32, error) {
	id, err := strconv.ParseInt(req.PathValue("id"), 10, 32)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return int32(id), nil
}

func (app *application) readJSON(w http.ResponseWriter, req *http.Request, dst interface{}) error {
	maxBytes := 1_048_576
	req.Body = http.MaxBytesReader(w, req.Body, int64(maxBytes))
	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf(
				"body contains badly-formed JSON (at character %d)",
				syntaxError.Offset,
			)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf(
					"body contains incorrect JSON type for field %q",
					unmarshalTypeError.Field,
				)
			}
			return fmt.Errorf(
				"body contains incorrect JSON type (at character %d)",
				unmarshalTypeError.Offset,
			)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)

		case errors.As(err, &invalidUnmarshalError):
			// A json.InvalidUnmarshalError means the caller passed an invalid
			// destination, which is a programmer error rather than bad client input.
			panic(err)

		default:
			return err
		}
	}

	// Decode once more into an empty struct so we can reject bodies containing
	// multiple top-level JSON values.
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func (app *application) background(fn func()) {
	app.wg.Add(1)

	go func() {
		defer app.wg.Done()

		defer func() {
			if err := recover(); err != nil {
				app.logger.PrintError(fmt.Errorf("%s", err), nil)
			}
		}()

		fn()
	}()
}

func (app *application) readString(qs url.Values, key, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	return s
}

func (app *application) readInt(qs url.Values, key string, defaultValue int32) int32 {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return defaultValue
	}

	return int32(i)
}

func (app *application) encodeCursor(u uuid.UUID, t time.Time) string {
	s := fmt.Sprintf("%s_%s", t.Format(time.RFC3339Nano), u)
	return base64.URLEncoding.EncodeToString([]byte(s))
}

func (app *application) decodeCursor(cursorStr string) (uuid.UUID, time.Time, error) {
	b, err := base64.URLEncoding.DecodeString(cursorStr)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}

	cursor := string(b)
	parts := strings.Split(cursor, "_")
	if len(parts) != 2 {
		return uuid.Nil, time.Time{}, errors.New("malformed cursor")
	}

	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}

	u, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}

	return u, t, nil
}

func (app *application) encodeCursorHighestRatedMedia(
	ratingValue int32,
	updatedAt time.Time,
	mediaID uuid.UUID,
) string {
	s := fmt.Sprintf("%d_%s_%s", ratingValue, updatedAt.Format(time.RFC3339Nano), mediaID)
	return base64.URLEncoding.EncodeToString([]byte(s))
}

func (app *application) decodeCursorHighestRatedMedia(
	cursorStr string,
) (ratingValue int32, updatedAt time.Time, mediaID uuid.UUID, err error) {
	b, err := base64.URLEncoding.DecodeString(cursorStr)
	if err != nil {
		return 0, time.Time{}, uuid.Nil, err
	}

	parts := strings.Split(string(b), "_")
	if len(parts) != 3 {
		return 0, time.Time{}, uuid.Nil, errors.New("malformed cursor")
	}

	rating, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return 0, time.Time{}, uuid.Nil, err
	}

	updatedAt, err = time.Parse(time.RFC3339Nano, parts[1])
	if err != nil {
		return 0, time.Time{}, uuid.Nil, err
	}

	mediaID, err = uuid.Parse(parts[2])
	if err != nil {
		return 0, time.Time{}, uuid.Nil, err
	}

	return int32(rating), updatedAt, mediaID, nil
}

func (app *application) encodeCursorMatches(
	score float64,
	sharedMediaCount int32,
	recalculatedAt time.Time,
	otherUserID uuid.UUID,
) string {
	s := fmt.Sprintf(
		"%v_%d_%s_%s",
		score,
		sharedMediaCount,
		recalculatedAt.Format(time.RFC3339Nano),
		otherUserID,
	)
	return base64.URLEncoding.EncodeToString([]byte(s))
}

func (app *application) decodeCursorMatches(
	cursorStr string,
) (score float64, sharedMediaCount int32, recalculatedAt time.Time, otherUserID uuid.UUID, err error) {
	b, err := base64.URLEncoding.DecodeString(cursorStr)
	if err != nil {
		return 0, 0, time.Time{}, uuid.Nil, err
	}

	cursor := string(b)
	parts := strings.Split(cursor, "_")
	if len(parts) != 4 {
		return 0.0, 0, time.Time{}, uuid.Nil, errors.New("malformed cursor")
	}

	score, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, time.Time{}, uuid.Nil, err
	}

	sharedMedia, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return 0, 0, time.Time{}, uuid.Nil, err
	}
	sharedMediaCount = int32(sharedMedia)

	recalculatedAt, err = time.Parse(time.RFC3339Nano, parts[2])
	if err != nil {
		return 0, 0, time.Time{}, uuid.Nil, err
	}

	otherUserID, err = uuid.Parse(parts[3])
	if err != nil {
		return 0, 0, time.Time{}, uuid.Nil, err
	}

	return score, sharedMediaCount, recalculatedAt, otherUserID, nil
}

func (app *application) videoURLs(site, key string) (watch, embed string, ok bool) {
	switch site {
	case "YouTube", "youtube":
		return "https://www.youtube.com/watch?v=" + key,
			"https://www.youtube.com/embed/" + key, true
	case "Vimeo", "vimeo":
		return "https://vimeo.com/" + key,
			"https://player.vimeo.com/video/" + key, true
	}
	return "", "", false
}
