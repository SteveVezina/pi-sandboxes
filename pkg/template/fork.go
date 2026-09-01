package template

import (
	"fmt"
	"time"
)

// Fork creates an editable local template named newName derived from an
// existing template (builtin, local, snapshot, or imported). The new
// template records its parentage in Source and Lineage and is written to
// the store.
func (s *Store) Fork(source, newName string) (*Template, error) {
	if newName == "" {
		return nil, fmt.Errorf("new template name is required")
	}
	if _, err := s.Get(newName); err == nil {
		return nil, fmt.Errorf("template %q already exists", newName)
	}

	src, err := s.Get(source)
	if err != nil {
		return nil, fmt.Errorf("load source template %q: %w", source, err)
	}

	parentDigest := src.ContentDigest()
	parentGen := 0
	if src.Lineage != nil {
		parentGen = src.Lineage.Generation
	}

	forked := *src
	forked.Name = newName
	forked.Source = &Source{
		Type:       SourceLocal,
		Parent:     source,
		ForkedFrom: source,
	}
	now := time.Now().UTC().Format(time.RFC3339)
	forked.CreatedAt = now
	forked.UpdatedAt = now
	forked.Lineage = &Lineage{
		Generation:   parentGen + 1,
		ParentDigest: parentDigest,
	}
	forked.Lineage.ContentDigest = forked.ContentDigest()

	if err := s.Create(newName, &forked); err != nil {
		return nil, fmt.Errorf("write forked template: %w", err)
	}
	return &forked, nil
}
