package planning

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jaelricco/hefesto/internal/content"
	"github.com/jaelricco/hefesto/internal/domain/planning"
)

// ContentKnowledge loads the knowledge base from a content directory once,
// at start-up. When validation fails the API keeps running and every
// planning call returns ErrUnavailable; logging, sync and the skill map are
// not affected (spec §2.6). It never panics.
type ContentKnowledge struct {
	kb  *planning.Knowledge
	err error
}

// LoadContentKnowledge validates dir/training. With production set, draft
// content (KB-13) is an error too, as long as ENT-10 is open.
func LoadContentKnowledge(dir string, production bool, log *slog.Logger) *ContentKnowledge {
	k, issues, err := content.LoadTraining(dir)
	if err != nil {
		log.Error("planner knowledge base unreadable", "err", err)
		return &ContentKnowledge{err: fmt.Errorf("%w: %w", ErrUnavailable, err)}
	}
	var problems []string
	for _, i := range issues {
		switch {
		case i.Severity == content.SeverityError:
			problems = append(problems, i.String())
		case i.Severity == content.SeverityNote && production:
			problems = append(problems, i.String())
		default:
			log.Warn("planner knowledge base", "issue", i.String())
		}
	}
	if len(problems) > 0 {
		log.Error("planner knowledge base invalid; planning endpoints answer 503", "issues", len(problems))
		return &ContentKnowledge{err: fmt.Errorf("%w: %s", ErrUnavailable, strings.Join(problems, "; "))}
	}
	log.Info("planner knowledge base loaded", "ruleset_version", k.Version)
	return &ContentKnowledge{kb: k}
}

// Current returns the knowledge base or the reason it is unavailable.
func (c *ContentKnowledge) Current(context.Context) (*planning.Knowledge, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.kb, nil
}
