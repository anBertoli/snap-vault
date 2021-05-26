package main

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/anBertoli/snap-vault/pkg/tracing"
)

// The env type is a flexible wrapper used to send JSON-formatted data
// using the functions provided in this file.
type env map[string]interface{}

// The sendJSON helper writes the provided JSON-formatted data to response writer,
// after recording some tracing data. Basically it is a wrapper over the
// writeJSON function.
func (app *application) sendJSON(w http.ResponseWriter, r *http.Request, status int, data env, headers http.Header) {
	trace := tracing.TraceFromRequestCtx(r)
	trace.HttpStatus = status
	trace.Err = nil

	err := writeJSON(w, status, data, headers)
	if err != nil {
		app.logger.Errorw("sending json", "id", trace.ID, "err", err)
		trace.HttpStatus = http.StatusInternalServerError
		trace.Err = err
	}
}

// The sendJSONError() method is a helper for sending JSON-formatted error messages
// to the client, after recording some tracing data.
func (app *application) sendJSONError(w http.ResponseWriter, r *http.Request, resp errResponse) {
	trace := tracing.TraceFromRequestCtx(r)
	trace.HttpStatus = resp.status
	trace.Message = resp.message
	trace.Err = resp.err

	err := writeJSON(w, resp.status, env{
		"status_code": resp.status,
		"error":       resp.message,
	}, nil)
	if err != nil {
		app.logger.Errorw("sending json", "id", trace.ID, "err", err)
		trace.HttpStatus = http.StatusInternalServerError
		trace.Err = err
	}
}

// The errResponse struct groups the public message to be provided to the
// client, the HTTP status code and the internal error to be logged.
type errResponse struct {
	message interface{}
	status  int
	err     error
}

// The writeJSON() helper writes the data to the response writer. The data is
// JSON-formatted before being sent. Provided headers are set on the response.
func writeJSON(w http.ResponseWriter, status int, data env, headers http.Header) error {

	// Encode the data to JSON, returning the error if there was one.
	// Avoid indentation and newline in case of performance issues,
	// otherwise it makes the output easier to read for clients.
	js, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	js = append(js, '\n')

	// Loop through the header map and add each header to the
	// http.ResponseWriter header map.
	for key, value := range headers {
		w.Header()[key] = value
	}

	// Add the mandatory "Content-Type: application/json" header, then write the
	// status code and JSON response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(js)
	return err
}

// The streamBytes function is a helper function used to send binary data to the client.
// The data is extracted from a reader, which if it's also a closer, it will be closed
// after reading all the content.
func (app *application) streamBytes(w http.ResponseWriter, r *http.Request, reader io.Reader, headers http.Header) {
	trace := tracing.TraceFromRequestCtx(r)
	trace.HttpStatus = http.StatusOK
	trace.Err = nil

	logger := app.logger.With("id", trace.ID)

	if rc, ok := reader.(io.ReadCloser); ok {
		defer func() {
			// If the reader is also a closer then defer a function that will close it.
			// Generally, we need to close the reader to finalize resources or to
			// signal the end at the data source/generator.
			err := rc.Close()
			if err != nil {
				logger.Errorw("error closing read closer", "err", err)
			}
		}()
	}

	// Set provided headers before start sending data.
	for key, value := range headers {
		w.Header()[key] = value
	}

	_, err := io.Copy(w, reader)
	if err != nil {
		var netErr *net.OpError
		switch {
		case errors.As(err, &netErr):
			// This is a network/client issue. We cannot do a lot here, so we simply
			// log the error. This is not a 'strict' HTTP error, but we still report
			// it with the status 500.
			logger.Errorw("network/client issue streaming bytes", "err", err)
		default:
			// The error is originated internally. Here the status code on the
			// response was already set and we cannot modify it, but we can
			// report this internally.
			logger.Errorw("streaming bytes", "err", err)
		}

		trace.HttpStatus = http.StatusInternalServerError
		trace.Err = err
	}
}
