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
	w.Write(js)

	return nil
}

func (app *application) readIDParam(r *http.Request) (int32, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return int32(id), nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)
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

func (app *application) readInt(qs url.Values, key string, defaultValue int) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return i
}

func (app *application) convertStringToDate(date string) (time.Time, error) {
	layout := "2006-01-02"

	t, err := time.Parse(layout, date)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

func (app *application) decodeCursor(cursorStr string) (uuid.UUID, time.Time, error) {
	b, err := base64.URLEncoding.DecodeString(cursorStr)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}

	cursor := string(b)
	parts := strings.Split(cursor, "_")
	if len(parts) != 2 {
		return uuid.Nil, time.Time{}, fmt.Errorf("malformed cursor")
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

func (app *application) encodeCursor(u uuid.UUID, t time.Time) string {
	s := fmt.Sprintf("%v_%v", t, u)
	return base64.URLEncoding.EncodeToString([]byte(s))
}

func (app *application) translateRatingValueOutbound(value int32) float64 {
	return float64(value) / 2.0
}

func (app *application) translateRatingValueInbound(value float64) int32 {
	return int32(value * 2)
}
