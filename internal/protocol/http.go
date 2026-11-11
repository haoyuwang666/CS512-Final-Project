// TODO(Haoyu): Maintain HTTP contracts and bounded decoding; add authentication before wider deployment.
package protocol

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
	LeaderURL string `json:"leader_url,omitempty"`
}

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func Fail(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, map[string]any{"error": APIError{Code: code, Message: message, Retryable: status == 503}})
}
func Decode(w http.ResponseWriter, r *http.Request, value any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("expected exactly one JSON value")
	}
	return nil
}
func Health(w http.ResponseWriter, r *http.Request) { JSON(w, 200, map[string]string{"status": "ok"}) }
func NotImplemented(w http.ResponseWriter, r *http.Request) {
	Fail(w, 501, "NOT_IMPLEMENTED", "extension point; not implemented in the starter")
}
