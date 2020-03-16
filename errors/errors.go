package errors

import (
	"fmt"
	"net/http"
)

type HttpError struct {
	StatusCode int
}

func (err HttpError) Name() string {
	return "http"
}

func (err HttpError) Error() string {
	return fmt.Sprintf("http request failed with %d %v status code", err.StatusCode, http.StatusText(err.StatusCode))
}

type ExtractError struct {
	Caller  string
	Pattern string
}

func (err ExtractError) Name() string {
	return "extract"
}

func (err ExtractError) Error() string {
	return fmt.Sprintf("%v: could not find match for pattern '%v'", err.Caller, err.Pattern)
}

type VideoUnavailableError struct {
	VideoID string
}

func (err VideoUnavailableError) Name() string {
	return "unavailable"
}

func (err VideoUnavailableError) Error() string {
	return fmt.Sprintf("video %v is unavailable", err.VideoID)
}

type VideoUnsupportedError struct {
	VideoID string
}

func (err VideoUnsupportedError) Name() string {
	return "unsupported"
}

func (err VideoUnsupportedError) Error() string {
	return fmt.Sprintf("video %v is unsupported (live content)", err.VideoID)
}

type RequestFailedError struct {
	Reason string
}

func (err RequestFailedError) Name() string {
	return "request"
}

func (err RequestFailedError) Error() string {
	return fmt.Sprintf("youtube request failed due to '%v'", err.Reason)
}
