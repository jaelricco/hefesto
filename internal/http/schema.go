package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/yaml.v3"

	"github.com/jaelricco/hefesto/api"
	"github.com/jaelricco/hefesto/internal/domain/training"
)

const openAPIID = "https://hefesto.fit/openapi.json"

var printer = message.NewPrinter(language.English)

// Schemas validates JSON documents against components/schemas of the
// embedded OpenAPI document. OpenAPI 3.1 schemas are JSON Schema 2020-12, so
// the spec's own rules — enums, ranges, required fields, no unknown fields —
// are enforced by the spec, not by a hand-written copy of it.
type Schemas struct {
	mu       sync.Mutex
	compiler *jsonschema.Compiler
	compiled map[string]*jsonschema.Schema
}

// LoadSchemas parses the embedded OpenAPI document.
func LoadSchemas() (*Schemas, error) {
	var doc any
	if err := yaml.Unmarshal(api.OpenAPI, &doc); err != nil {
		return nil, fmt.Errorf("parsing openapi.yaml: %w", err)
	}
	// Round-trip through JSON to get the value types the validator expects.
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("converting openapi.yaml: %w", err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("converting openapi.yaml: %w", err)
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.AssertFormat()
	if err := c.AddResource(openAPIID, inst); err != nil {
		return nil, fmt.Errorf("loading openapi.yaml: %w", err)
	}
	return &Schemas{compiler: c, compiled: map[string]*jsonschema.Schema{}}, nil
}

// Schema returns the compiled components/schemas/<name>.
func (s *Schemas) Schema(name string) (*jsonschema.Schema, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sch, ok := s.compiled[name]; ok {
		return sch, nil
	}
	sch, err := s.compiler.Compile(openAPIID + "#/components/schemas/" + name)
	if err != nil {
		return nil, fmt.Errorf("compiling schema %s: %w", name, err)
	}
	s.compiled[name] = sch
	return sch, nil
}

// Validate checks body against the named schema. A body that is not JSON is
// a bad request; one that breaks the schema is a validation error keyed by
// JSON pointer.
func (s *Schemas) Validate(name string, body []byte) error {
	sch, err := s.Schema(name)
	if err != nil {
		return err
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil {
		return errBadRequest{"malformed JSON"}
	}
	err = sch.Validate(inst)
	if err == nil {
		return nil
	}
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return fmt.Errorf("validating against %s: %w", name, err)
	}
	return &training.ValidationError{Fields: leafErrors(ve)}
}

// leafErrors flattens a validation error to one message per instance
// location, keeping the most specific failure.
func leafErrors(ve *jsonschema.ValidationError) training.FieldErrors {
	out := training.FieldErrors{}
	var walk func(e *jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) == 0 {
			loc := "/" + strings.Join(e.InstanceLocation, "/")
			if loc == "/" {
				loc = ""
			}
			if _, seen := out[loc]; !seen {
				out[loc] = e.ErrorKind.LocalizedString(printer)
			}
			return
		}
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(ve)
	return out
}
