package content

import "testing"

// The knowledge base in the repository must load without errors or
// warnings; draft status is reported as a note (KB-13).
func TestLoadTrainingRepositoryContent(t *testing.T) {
	k, issues, err := LoadTraining("../../content")
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range issues {
		if i.Severity != SeverityNote {
			t.Errorf("%s", i)
		}
	}
	if k == nil || k.Version == "" {
		t.Fatal("no knowledge base")
	}
}
