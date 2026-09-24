package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// maxBodyBytes bounds every request body. The largest legitimate one — a set
// with twenty elements — is a few kilobytes.
const maxBodyBytes = 256 << 10

// errBadRequest marks a body that is not the JSON the endpoint expects.
type errBadRequest struct{ msg string }

func (e errBadRequest) Error() string { return e.msg }

// decode reads a JSON body strictly: unknown fields, trailing data and
// oversized bodies are rejected, so a client typo is an error rather than a
// silently ignored field.
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var tooBig *http.MaxBytesError
		var syntax *json.SyntaxError
		var typ *json.UnmarshalTypeError
		switch {
		case errors.As(err, &tooBig):
			return errBadRequest{fmt.Sprintf("body larger than %d bytes", tooBig.Limit)}
		case errors.As(err, &syntax):
			return errBadRequest{fmt.Sprintf("malformed JSON at byte %d", syntax.Offset)}
		case errors.As(err, &typ):
			return errBadRequest{fmt.Sprintf("field %q must be %s", typ.Field, typ.Type)}
		case errors.Is(err, io.EOF):
			return errBadRequest{"empty body"}
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			return errBadRequest{strings.TrimPrefix(err.Error(), "json: ")}
		default:
			return errBadRequest{err.Error()}
		}
	}
	if dec.More() {
		return errBadRequest{"trailing data after the JSON body"}
	}
	return nil
}

// Optional is a field of a PATCH body: it distinguishes "absent" (leave
// alone) from "null" (clear) from a value.
type Optional[T any] struct {
	Set   bool
	Null  bool
	Value T
}

// UnmarshalJSON is only called when the key is present.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		o.Null = true
		return nil
	}
	if err := json.Unmarshal(data, &o.Value); err != nil {
		return fmt.Errorf("optional field: %w", err)
	}
	return nil
}

// Ptr returns nil for null, the value otherwise. Only meaningful when Set.
func (o Optional[T]) Ptr() *T {
	if o.Null {
		return nil
	}
	v := o.Value
	return &v
}

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }
