package content

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// Checksum fingerprints the normalised tree. Two trees with the same meaning
// have the same checksum regardless of file order, key order or YAML style,
// because it is computed over the loaded, defaulted, sorted structure rather
// than the bytes on disk. The /v1/skills ETag derives from it.
func Checksum(t Tree) ([]byte, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("encoding content tree: %w", err)
	}
	sum := sha256.Sum256(b)
	return sum[:], nil
}

// Counts is the item_counts summary recorded on a content version.
func (t Tree) Counts() map[string]int {
	levels, injuries := 0, 0
	for _, s := range t.Skills {
		levels += len(s.Levels)
		injuries += len(s.Injuries)
	}
	return map[string]int{
		"families":  len(t.Families),
		"exercises": len(t.Exercises),
		"bands":     len(t.Bands),
		"skills":    len(t.Skills),
		"levels":    levels,
		"edges":     len(t.Edges()),
		"injuries":  injuries,
	}
}
