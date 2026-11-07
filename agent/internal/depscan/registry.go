package depscan

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrEmptyEcosystem = errors.New(
		"scanner ecosystem cannot be empty",
	)

	ErrDuplicateScanner = errors.New(
		"scanner ecosystem already registered",
	)
)

type Registry struct {
	scanners map[string]Scanner
}

func NewRegistry() *Registry {
	return &Registry{
		scanners: make(map[string]Scanner),
	}
}

// Register adds a scanner to the registry.
func (r *Registry) Register(scanner Scanner) error {
	if r == nil {
		return errors.New("scanner registry is nil")
	}

	if scanner == nil {
		return errors.New("scanner cannot be nil")
	}

	name := strings.ToLower(
		strings.TrimSpace(scanner.Ecosystem()),
	)

	if name == "" {
		return ErrEmptyEcosystem
	}

	if _, exists := r.scanners[name]; exists {
		return fmt.Errorf(
			"%w: %s",
			ErrDuplicateScanner,
			name,
		)
	}

	r.scanners[name] = scanner

	return nil
}

// Get retrieves a scanner by ecosystem name.
func (r *Registry) Get(
	ecosystem string,
) (Scanner, bool) {
	if r == nil {
		return nil, false
	}

	name := strings.ToLower(
		strings.TrimSpace(ecosystem),
	)

	scanner, exists := r.scanners[name]

	return scanner, exists
}

// All returns registered scanners in deterministic order.
func (r *Registry) All() []Scanner {
	if r == nil {
		return nil
	}

	result := make([]Scanner, 0, len(r.scanners))

	for _, scanner := range r.scanners {
		result = append(result, scanner)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Ecosystem() <
			result[j].Ecosystem()
	})

	return result
}

// Count returns the number of registered scanners.
func (r *Registry) Count() int {
	if r == nil {
		return 0
	}

	return len(r.scanners)
}
