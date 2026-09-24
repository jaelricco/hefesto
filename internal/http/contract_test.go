package http

import (
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"

	"github.com/jaelricco/hefesto/api"
	"github.com/jaelricco/hefesto/internal/auth"
	"github.com/jaelricco/hefesto/internal/store"
)

// TestRoutesMatchTheSpec keeps the router and api/openapi.yaml in lockstep:
// every served route is documented and every documented operation is served.
func TestRoutesMatchTheSpec(t *testing.T) {
	schemas, err := LoadSchemas()
	if err != nil {
		t.Fatal(err)
	}
	h := NewRouter(RouterDeps{Auth: &auth.Service{}, Store: store.New(nil), Schemas: schemas})

	served := map[string]bool{}
	err = chi.Walk(h.(chi.Routes), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		served[method+" "+strings.TrimSuffix(route, "/")] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(api.OpenAPI, &doc); err != nil {
		t.Fatal(err)
	}
	documented := map[string]bool{}
	for path, ops := range doc.Paths {
		for method := range ops {
			if method == "parameters" {
				continue
			}
			documented[strings.ToUpper(method)+" "+path] = true
		}
	}

	if len(documented) < 20 || len(served) < 20 {
		t.Fatalf("suspiciously few operations: %d documented, %d served", len(documented), len(served))
	}

	var missing, undocumented []string
	for op := range documented {
		if !served[op] {
			missing = append(missing, op)
		}
	}
	for op := range served {
		if !documented[op] {
			undocumented = append(undocumented, op)
		}
	}
	sort.Strings(missing)
	sort.Strings(undocumented)
	if len(missing) > 0 {
		t.Errorf("documented but not served:\n  %s", strings.Join(missing, "\n  "))
	}
	if len(undocumented) > 0 {
		t.Errorf("served but not documented:\n  %s", strings.Join(undocumented, "\n  "))
	}
}

// TestEveryRequestSchemaCompiles catches a broken $ref in the spec before a
// request does.
func TestEveryRequestSchemaCompiles(t *testing.T) {
	schemas, err := LoadSchemas()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"RegisterRequest", "LoginRequest", "AppleSignInRequest", "RefreshRequest", "UserUpdate", "BandCreate",
		"SessionCreate", "SessionUpdate", "BlockWrite", "SetEntryWrite", "ReorderRequest",
		"AuthResponse", "User", "DeletionScheduled", "ExerciseList", "Exercise", "BandList", "Band",
		"Session", "SessionPage", "Block", "SetEntry", "LastSet", "Problem",
		"SessionComplete", "AttestRequest", "CompletionResult", "AttestResult", "SkillGraph", "SkillDetail", "SkillMap", "Progress",
	} {
		if _, err := schemas.Schema(name); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestOptionalDistinguishesAbsentNullAndValue(t *testing.T) {
	var in struct {
		A Optional[int] `json:"a"`
		B Optional[int] `json:"b"`
		C Optional[int] `json:"c"`
	}
	r, _ := http.NewRequest("PATCH", "/", strings.NewReader(`{"b": null, "c": 3}`))
	if err := decode(nil, r, &in); err != nil {
		t.Fatal(err)
	}
	if in.A.Set || !in.B.Set || !in.B.Null || in.B.Ptr() != nil || !in.C.Set || *in.C.Ptr() != 3 {
		t.Fatalf("got %+v", in)
	}
}
