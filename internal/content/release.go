package content

import "fmt"

// Readiness counts how much of the tree is researched content and how much is
// still placeholder (CONTENT_AUTHORING.md, "Do not invent data").
type Readiness struct {
	Skills, PlaceholderSkills       int
	Exercises, PlaceholderExercises int
	Bands                           int
}

// Ready reports whether nothing is a placeholder any more.
func (r Readiness) Ready() bool { return r.PlaceholderSkills == 0 && r.PlaceholderExercises == 0 }

func (r Readiness) String() string {
	return fmt.Sprintf("%d of %d skills and %d of %d exercises are draft placeholders; %d bands",
		r.PlaceholderSkills, r.Skills, r.PlaceholderExercises, r.Exercises, r.Bands)
}

// ReadinessOf counts the tree. Retired entries are not counted.
func ReadinessOf(t Tree) Readiness {
	r := Readiness{Bands: len(t.Bands)}
	for _, s := range t.Skills {
		if s.Status == StatusRetired {
			continue
		}
		r.Skills++
		if s.Status == StatusDraftPlaceholder {
			r.PlaceholderSkills++
		}
	}
	for _, e := range t.Exercises {
		if e.Status == StatusRetired {
			continue
		}
		r.Exercises++
		if e.Status == StatusDraftPlaceholder {
			r.PlaceholderExercises++
		}
	}
	return r
}

// ReleaseIssues are the extra checks for content that ships to athletes
// outside the team, such as an external TestFlight or App Store build: every
// placeholder is an error, because placeholders are not coaching content.
func ReleaseIssues(t Tree) []Issue {
	var out []Issue
	for _, s := range t.Skills {
		if s.Status == StatusDraftPlaceholder {
			out = append(out, errorf(s.File, "skill %q is still a draft placeholder; replace it with researched content or remove it", s.Slug))
		}
	}
	for _, e := range t.Exercises {
		if e.Status == StatusDraftPlaceholder {
			out = append(out, errorf(e.File, "exercise %q is still a draft placeholder; replace it with researched content or remove it", e.Slug))
		}
	}
	SortIssues(out)
	return out
}
