package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func (app *application) newTMDBRequest(
	ctx context.Context,
	path []string,
	query url.Values,
) (*http.Request, error) {
	endpoint, err := url.Parse(app.config.tmdbBaseURL)
	if err != nil {
		return nil, err
	}

	if endpoint.Host == "" || endpoint.User != nil {
		return nil, errors.New("invalid TMDB base URL")
	}
	if !allowedTMDBBaseURL(endpoint) {
		return nil, errors.New("invalid TMDB base URL")
	}

	pathSegments := append([]string{endpoint.Path}, path...)
	endpoint.Path, err = url.JoinPath(pathSegments[0], pathSegments[1:]...)
	if err != nil {
		return nil, err
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+app.config.tmdbToken)

	return req, nil
}

func allowedTMDBBaseURL(u *url.URL) bool {
	hostname := u.Hostname()
	if strings.EqualFold(hostname, "api.themoviedb.org") {
		return u.Scheme == "https"
	}

	ip := net.ParseIP(hostname)
	if ip != nil && ip.IsLoopback() {
		return u.Scheme == "http" || u.Scheme == "https"
	}

	return strings.EqualFold(hostname, "localhost") && (u.Scheme == "http" || u.Scheme == "https")
}
