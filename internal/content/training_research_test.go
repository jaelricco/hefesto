package content

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The research parameters in content/training/parameters.yaml are
// transcriptions of the parameter tables in docs/research/02–07. This test
// keeps them in step: key, value text and sources must match the row of the
// research table, so no number in the catalogue can drift from its source.
func TestTrainingParametersMatchResearch(t *testing.T) {
	files, issues, err := ReadTraining("../../content")
	if err != nil || HasErrors(issues, false) {
		t.Fatalf("reading the knowledge base: %v %v", err, issues)
	}
	rows := researchRows(t, "../../docs/research")
	if len(rows) < 300 {
		t.Fatalf("only %d research rows found", len(rows))
	}
	checked := 0
	for _, p := range files.Parameters {
		if strings.HasPrefix(p.ID, "PAR-S-") {
			continue
		}
		row, ok := rows[p.ID]
		if !ok {
			t.Errorf("%s is not in any research table", p.ID)
			continue
		}
		checked++
		if p.Key != row.key {
			t.Errorf("%s: key %q, research %q", p.ID, p.Key, row.key)
		}
		if p.Text != row.text {
			t.Errorf("%s: text %q, research %q", p.ID, p.Text, row.text)
		}
		for _, s := range p.Sources {
			if !slices.Contains(row.sources, s) {
				t.Errorf("%s: source %s is not cited in the research row", p.ID, s)
			}
		}
		if (p.Value != nil || len(p.Values) > 0) && p.Text == "" {
			t.Errorf("%s: numbers without the research value text", p.ID)
		}
	}
	if checked < 200 {
		t.Errorf("only %d research parameters checked", checked)
	}
}

type researchRow struct {
	key, text string
	sources   []string
}

var (
	paramRow = regexp.MustCompile(`^\| (PAR-[A-F]-\d+) \|`)
	sourceID = regexp.MustCompile(`(?:^|[^\w-])([A-Z])-(\d{2,3})\b`)
	rangeID  = regexp.MustCompile(`(?:^|[^\w-])([A-Z])-(\d{2,3})\s*(?:bis|–)\s*(?:[A-Z]-)?(\d{2,3})\b`)
)

func researchRows(t *testing.T, dir string) map[string]researchRow {
	t.Helper()
	out := map[string]researchRow{}
	paths, err := filepath.Glob(filepath.Join(dir, "0[2-7]_*.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		f, err := os.Open(path) //nolint:gosec // test fixture under docs/research
		if err != nil {
			t.Fatal(err)
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			line := sc.Text()
			m := paramRow.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			if _, dup := out[m[1]]; dup {
				continue // the first table wins, as in the generator
			}
			cells := splitRow(line)
			if len(cells) != 7 {
				t.Errorf("%s: %s has %d cells", filepath.Base(path), m[1], len(cells))
				continue
			}
			out[m[1]] = researchRow{key: strings.Trim(cells[1], "`"), text: cells[2], sources: ids(cells[4])}
		}
		_ = f.Close()
		if err := sc.Err(); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	var cells []string
	var cur strings.Builder
	for i := 0; i < len(line); i++ {
		if line[i] == '\\' && i+1 < len(line) && line[i+1] == '|' {
			cur.WriteByte('|')
			i++
			continue
		}
		if line[i] == '|' {
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(line[i])
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

// ids extracts source IDs, expanding «X-01 bis X-03» ranges.
func ids(text string) []string {
	var out []string
	for _, m := range rangeID.FindAllStringSubmatch(text, -1) {
		lo, hi := atoi(m[2]), atoi(m[3])
		for n := lo; n <= hi && hi-lo < 30; n++ {
			out = append(out, m[1]+"-"+pad(n, len(m[2])))
		}
	}
	for _, m := range sourceID.FindAllStringSubmatch(text, -1) {
		out = append(out, m[1]+"-"+m[2])
	}
	return out
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// pad zero-pads n to the width of the range's lower bound.
func pad(n, width int) string {
	return fmt.Sprintf("%0*d", width, n)
}
