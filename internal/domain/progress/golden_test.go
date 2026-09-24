package progress

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The golden suite: every file in testdata/criteria holds criteria, a
// history, a clock and the expected result, written by hand. Adding a case
// is adding a file; see testdata/criteria/README.md.

type goldenObservation struct {
	Set           string   `json:"set"`
	Exercise      string   `json:"exercise"`
	Measure       string   `json:"measure"`
	Value         *float64 `json:"value"`
	LoadKg        float64  `json:"load_kg"`
	Assistance    string   `json:"assistance"`
	FormQuality   *int     `json:"form_quality"`
	Failed        bool     `json:"failed"`
	PartialROM    bool     `json:"partial_rom"`
	EccentricOnly bool     `json:"eccentric_only"`
	At            string   `json:"at"`
}

type goldenCondition struct {
	Met   bool     `json:"met"`
	Count int      `json:"count"`
	Best  *float64 `json:"best"`
}

type goldenCase struct {
	Description string              `json:"description"`
	Now         string              `json:"now"`
	Criteria    json.RawMessage     `json:"criteria"`
	History     []goldenObservation `json:"history"`
	Expect      *struct {
		Met            bool              `json:"met"`
		SelfAttestOnly bool              `json:"self_attest_only"`
		Started        bool              `json:"started"`
		Evidence       *string           `json:"evidence"`
		All            []goldenCondition `json:"all"`
		Any            []goldenCondition `json:"any"`
	} `json:"expect"`
	ExpectError string `json:"expect_error"`
}

// setID maps a golden set name to a stable UUID.
func setID(name string) uuid.UUID { return uuid.NewSHA1(uuid.NameSpaceOID, []byte(name)) }

func TestGoldenCriteria(t *testing.T) {
	files, err := filepath.Glob("../../../testdata/criteria/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 15 {
		t.Fatalf("expected the golden suite, found %d files", len(files))
	}
	for _, f := range files {
		t.Run(strings.TrimSuffix(filepath.Base(f), ".json"), func(t *testing.T) {
			raw, err := os.ReadFile(f) //nolint:gosec // test fixture path from a glob
			if err != nil {
				t.Fatal(err)
			}
			var gc goldenCase
			dec := json.NewDecoder(strings.NewReader(string(raw)))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&gc); err != nil {
				t.Fatalf("fixture: %v", err)
			}
			if gc.Description == "" {
				t.Fatal("fixture has no description")
			}
			now, err := time.Parse(time.RFC3339, gc.Now)
			if err != nil {
				t.Fatalf("now: %v", err)
			}
			var c Criteria
			cdec := json.NewDecoder(strings.NewReader(string(gc.Criteria)))
			cdec.DisallowUnknownFields()
			if err := cdec.Decode(&c); err != nil {
				t.Fatalf("criteria: %v", err)
			}
			var h SetHistory
			for _, o := range gc.History {
				at, err := time.Parse(time.RFC3339, o.At)
				if err != nil {
					t.Fatalf("history at: %v", err)
				}
				assist := o.Assistance
				if assist == "" {
					assist = ClassUnassisted
				}
				h = append(h, Observation{
					SetEntryID: setID(o.Set), Exercise: o.Exercise, Measure: o.Measure, Value: o.Value,
					LoadKg: o.LoadKg, Assistance: assist, FormQuality: o.FormQuality, Failed: o.Failed,
					PartialROM: o.PartialROM, EccentricOnly: o.EccentricOnly, PerformedAt: at,
				})
			}

			got, err := Evaluate(c, h, now)
			if gc.ExpectError != "" {
				if !errors.Is(err, ErrInvalidCriteria) || !strings.Contains(err.Error(), gc.ExpectError) {
					t.Fatalf("expected an error containing %q, got %v", gc.ExpectError, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := gc.Expect
			if want == nil {
				t.Fatal("fixture has neither expect nor expect_error")
			}
			if got.Met != want.Met || got.SelfAttestOnly != want.SelfAttestOnly || got.Started != want.Started {
				t.Errorf("met=%v self_attest_only=%v started=%v; want %v %v %v",
					got.Met, got.SelfAttestOnly, got.Started, want.Met, want.SelfAttestOnly, want.Started)
			}
			switch {
			case want.Evidence == nil && got.Evidence != nil:
				t.Errorf("evidence %v, want none", got.Evidence)
			case want.Evidence != nil && (got.Evidence == nil || *got.Evidence != setID(*want.Evidence)):
				t.Errorf("evidence %v, want set %q", got.Evidence, *want.Evidence)
			}
			compare(t, "all", got.All, want.All)
			compare(t, "any", got.Any, want.Any)
		})
	}
}

func compare(t *testing.T, name string, got []ConditionResult, want []goldenCondition) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: %d conditions, want %d", name, len(got), len(want))
		return
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Met != w.Met || g.Count != w.Count {
			t.Errorf("%s[%d]: met=%v count=%d; want met=%v count=%d", name, i, g.Met, g.Count, w.Met, w.Count)
		}
		switch {
		case w.Best == nil && g.Best != nil:
			t.Errorf("%s[%d]: best %v, want none", name, i, *g.Best)
		case w.Best != nil && (g.Best == nil || *g.Best != *w.Best):
			t.Errorf("%s[%d]: best %v, want %v", name, i, g.Best, *w.Best)
		}
	}
}
