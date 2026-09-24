package content

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// Layout of a content directory. Paths are relative to its root.
const (
	SchemaDir    = "schema"
	FamiliesFile = "families.yaml"
	ExercisesDir = "exercises"
	BandsDir     = "bands"
	SkillsDir    = "skills"
)

const schemaBase = "https://hefesto.fit/schema/"

// Load reads and schema-validates every file under dir.
//
// Content problems come back as issues, so a single run reports all of them.
// The error is reserved for problems with the tree itself: a missing
// directory, unreadable files, or schemas that do not compile. When any issue
// is an error the returned Tree is partial and must not be seeded.
func Load(dir string) (Tree, []Issue, error) {
	if st, err := os.Stat(dir); err != nil {
		return Tree{}, nil, fmt.Errorf("content directory: %w", err)
	} else if !st.IsDir() {
		return Tree{}, nil, fmt.Errorf("content directory: %s is not a directory", dir)
	}

	schemas, err := compileSchemas(filepath.Join(dir, SchemaDir))
	if err != nil {
		return Tree{}, nil, err
	}

	l := loader{dir: dir, schemas: schemas}
	var t Tree

	var fams struct {
		Families []Family `yaml:"families"`
	}
	if l.decode(FamiliesFile, "families", &fams) {
		t.Families = fams.Families
	}

	for _, f := range l.glob(ExercisesDir) {
		var e Exercise
		if l.decode(f, "exercise", &e) {
			e.File = f
			t.Exercises = append(t.Exercises, e)
		}
	}
	for _, f := range l.glob(BandsDir) {
		var bs struct {
			Bands []Band `yaml:"bands"`
		}
		if l.decode(f, "bands", &bs) {
			for _, b := range bs.Bands {
				b.File = f
				t.Bands = append(t.Bands, b)
			}
		}
	}
	for _, f := range l.glob(SkillsDir) {
		var s Skill
		if l.decode(f, "skill", &s) {
			s.File = f
			t.Skills = append(t.Skills, s)
		}
	}
	if l.err != nil {
		return Tree{}, nil, l.err
	}

	normalise(&t)
	return t, l.issues, nil
}

type loader struct {
	dir     string
	schemas map[string]*jsonschema.Schema
	issues  []Issue
	err     error
}

func (l *loader) glob(sub string) []string {
	var out []string
	root := filepath.Join(l.dir, sub)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root {
				return fs.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(l.dir, path)
		switch filepath.Ext(path) {
		case ".yaml":
			out = append(out, rel)
		case ".yml":
			l.issues = append(l.issues, errorf(rel, "use the .yaml extension"))
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) && l.err == nil {
		l.err = fmt.Errorf("reading %s: %w", sub, err)
	}
	slices.Sort(out)
	return out
}

// decode validates rel against the named schema and, if clean, decodes it
// into v. It reports whether v was filled.
func (l *loader) decode(rel, schema string, v any) bool {
	raw, err := os.ReadFile(filepath.Join(l.dir, rel))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			l.issues = append(l.issues, errorf(rel, "file is missing"))
			return false
		}
		if l.err == nil {
			l.err = fmt.Errorf("reading %s: %w", rel, err)
		}
		return false
	}

	var generic any
	if err := yaml.Unmarshal(raw, &generic); err != nil {
		l.issues = append(l.issues, errorf(rel, "invalid YAML: %v", err))
		return false
	}
	// Round-trip through JSON: the schema validator wants JSON values, and
	// this also rejects YAML-only constructs such as non-string keys.
	asJSON, err := json.Marshal(generic)
	if err != nil {
		l.issues = append(l.issues, errorf(rel, "not representable as JSON: %v", err))
		return false
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(asJSON))
	if err != nil {
		l.issues = append(l.issues, errorf(rel, "not representable as JSON: %v", err))
		return false
	}
	if err := l.schemas[schema].Validate(inst); err != nil {
		var ve *jsonschema.ValidationError
		if !errors.As(err, &ve) {
			l.issues = append(l.issues, errorf(rel, "schema: %v", err))
			return false
		}
		for _, msg := range flatten(ve) {
			l.issues = append(l.issues, errorf(rel, "schema: %s", msg))
		}
		return false
	}

	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		// The schema passed, so this is a mismatch between the schema and the
		// Go types — a bug here, not in the content.
		l.issues = append(l.issues, errorf(rel, "decoding after schema passed (schema and Go types disagree): %v", err))
		return false
	}
	return true
}

// flatten turns a validation error into one line per failure. The library's
// own rendering is the most readable (its structured output drops messages
// behind a $ref), so this reuses it and strips the indentation.
func flatten(ve *jsonschema.ValidationError) []string {
	var out []string
	for _, line := range strings.Split(ve.Error(), "\n") {
		line = strings.TrimSpace(line)
		if msg, ok := strings.CutPrefix(line, "- "); ok {
			out = append(out, msg)
		}
	}
	if len(out) == 0 {
		out = append(out, ve.Error())
	}
	return out
}

func compileSchemas(dir string) (map[string]*jsonschema.Schema, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.schema.json"))
	if err != nil {
		return nil, fmt.Errorf("listing schemas: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no schemas found in %s", dir)
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	for _, f := range files {
		fh, err := os.Open(f) //nolint:gosec // path comes from a glob of the content tree
		if err != nil {
			return nil, fmt.Errorf("opening schema: %w", err)
		}
		doc, err := jsonschema.UnmarshalJSON(fh)
		_ = fh.Close()
		if err != nil {
			return nil, fmt.Errorf("parsing schema %s: %w", f, err)
		}
		if err := c.AddResource(schemaBase+filepath.Base(f), doc); err != nil {
			return nil, fmt.Errorf("adding schema %s: %w", f, err)
		}
	}
	out := map[string]*jsonschema.Schema{}
	for _, name := range []string{"families", "exercise", "bands", "skill"} {
		s, err := c.Compile(schemaBase + name + ".schema.json")
		if err != nil {
			return nil, fmt.Errorf("compiling %s schema: %w", name, err)
		}
		out[name] = s
	}
	return out, nil
}

// normalise applies defaults, replaces nil slices with empty ones, and sorts
// everything into a canonical order (families keep file order, which is their
// display order), so that the checksum depends only on
// meaning and the store never writes NULL into a NOT NULL array column.
func normalise(t *Tree) {
	for i := range t.Exercises {
		e := &t.Exercises[i]
		e.Status = cmp.Or(e.Status, StatusActive)
		e.LoadSemantics = cmp.Or(e.LoadSemantics, "added")
		e.IsBodyweight = boolOr(e.IsBodyweight, true)
		e.TempoApplicable = boolOr(e.TempoApplicable, true)
		e.Aka, e.Equipment, e.Cues, e.CommonFaults = nonNil(e.Aka), nonNil(e.Equipment), nonNil(e.Cues), nonNil(e.CommonFaults)
		e.Summary = strings.TrimSpace(e.Summary)
	}
	for i := range t.Skills {
		s := &t.Skills[i]
		s.Status = cmp.Or(s.Status, StatusActive)
		s.Aka, s.PrimaryMuscles, s.CommonFaults = nonNil(s.Aka), nonNil(s.PrimaryMuscles), nonNil(s.CommonFaults)
		s.Summary = strings.TrimSpace(s.Summary)
		for j := range s.Levels {
			l := &s.Levels[j]
			l.Description = strings.TrimSpace(l.Description)
			l.UnlockCriteria = l.UnlockCriteria.WithDefaults()
		}
		slices.SortStableFunc(s.Levels, func(a, b Level) int { return cmp.Compare(a.Order, b.Order) })
		for j := range s.Injuries {
			in := &s.Injuries[j]
			in.Description = strings.TrimSpace(in.Description)
			in.RiskFactors, in.EarlySigns = nonNil(in.RiskFactors), nonNil(in.EarlySigns)
		}
	}
	slices.SortStableFunc(t.Exercises, func(a, b Exercise) int { return cmp.Compare(a.Slug, b.Slug) })
	slices.SortStableFunc(t.Skills, func(a, b Skill) int { return cmp.Compare(a.Slug, b.Slug) })
	slices.SortStableFunc(t.Bands, func(a, b Band) int {
		return cmp.Or(cmp.Compare(a.Brand, b.Brand), cmp.Compare(a.ColourLabel, b.ColourLabel))
	})
}

func boolOr(b *bool, def bool) *bool {
	if b != nil {
		return b
	}
	return &def
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
