package content

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jaelricco/hefesto/internal/domain/planning"
)

// Layout of the planner's knowledge base inside a content directory.
const (
	TrainingDir       = "training"
	TrainingSchemaDir = "schema/training"
)

const trainingSchemaBase = schemaBase + "training/"

// trainingFiles maps each knowledge-base file to its schema name.
var trainingFiles = []string{"manifest", "parameters", "rules", "sources", "body", "exercises", "skills", "sessions", "onboarding"}

// LoadTraining reads, schema-validates (KB-01) and builds the planner's
// knowledge base under dir/training. Problems in the files come back as
// issues; the error is reserved for an unreadable tree or broken schemas.
// When any issue is an error the knowledge must not be used for planning.
func LoadTraining(dir string) (*planning.Knowledge, []Issue, error) {
	root := filepath.Join(dir, TrainingDir)
	if st, err := os.Stat(root); err != nil {
		return nil, nil, fmt.Errorf("training directory: %w", err)
	} else if !st.IsDir() {
		return nil, nil, fmt.Errorf("training directory: %s is not a directory", root)
	}
	schemas, err := compileSchemas(filepath.Join(dir, TrainingSchemaDir), trainingSchemaBase, trainingFiles)
	if err != nil {
		return nil, nil, err
	}
	l := loader{dir: dir, schemas: schemas}
	var f planning.Files
	rel := func(name string) string { return filepath.Join(TrainingDir, name+".yaml") }

	l.decode(rel("manifest"), "manifest", &f.Manifest)
	var params struct {
		Parameters []planning.Param `yaml:"parameters"`
	}
	if l.decode(rel("parameters"), "parameters", &params) {
		f.Parameters = params.Parameters
	}
	var rules struct {
		Rules []planning.Rule `yaml:"rules"`
	}
	if l.decode(rel("rules"), "rules", &rules) {
		f.Rules = rules.Rules
	}
	var sources struct {
		Sources []planning.Source `yaml:"sources"`
	}
	if l.decode(rel("sources"), "sources", &sources) {
		f.Sources = sources.Sources
	}
	l.decode(rel("body"), "body", &f.Body)
	var exercises struct {
		Exercises []planning.Exercise `yaml:"exercises"`
	}
	if l.decode(rel("exercises"), "exercises", &exercises) {
		f.Exercises = exercises.Exercises
	}
	var skills struct {
		Skills []planning.Skill `yaml:"skills"`
	}
	if l.decode(rel("skills"), "skills", &skills) {
		f.Skills = skills.Skills
	}
	l.decode(rel("sessions"), "sessions", &f.Sessions)
	l.decode(rel("onboarding"), "onboarding", &f.Onboarding)
	if l.err != nil {
		return nil, nil, l.err
	}
	// Cross-file checks on files that failed their schema only produce noise.
	if HasErrors(l.issues, false) {
		return nil, l.issues, nil
	}
	k, kbIssues := planning.Build(f)
	issues := l.issues
	drafts := 0
	for _, i := range kbIssues {
		// KB-13: draft content is the expected state until the review of
		// ENT-10; it is reported once as a note, like draft_placeholder in
		// the rest of the tree, which contentlint does not flag either.
		if i.Check == "KB-13" {
			drafts++
			continue
		}
		sev := SeverityError
		if i.Warn {
			sev = SeverityWarning
		}
		issues = append(issues, Issue{Severity: sev, File: TrainingDir, Msg: fmt.Sprintf("%s: %s: %s", i.Check, i.Where, i.Msg)})
	}
	if drafts > 0 {
		issues = append(issues, Issue{Severity: SeverityNote, File: TrainingDir,
			Msg: fmt.Sprintf("KB-13: %d exercises are draft content awaiting review (ENT-10)", drafts)})
	}
	return k, issues, nil
}
