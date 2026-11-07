// Package depscan provides a registry for dependency scanners.
package depscan

import (
	"context"
	"errors"
	"testing"
)

type mockScanner struct {
	name string
}

func (m mockScanner) Ecosystem() string {
	return m.name
}

func (m mockScanner) Detect(Source) Detection {
	return Detection{}
}

func (m mockScanner) RequiredExecutable() string {
	return "mock"
}

func (m mockScanner) Scan(
	context.Context,
	Source,
	Runtime,
	Executor,
	Detection,
) (EcosystemResult, error) {
	return EcosystemResult{
		Ecosystem: m.name,
		Status:    StatusCompleted,
	}, nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	registry := NewRegistry()

	scanner := mockScanner{
		name: "composer",
	}

	if err := registry.Register(scanner); err != nil {
		t.Fatalf("register scanner: %v", err)
	}

	if registry.Count() != 1 {
		t.Fatalf(
			"expected 1 scanner, got %d",
			registry.Count(),
		)
	}

	found, exists := registry.Get("composer")

	if !exists {
		t.Fatal("expected composer scanner to exist")
	}

	if found.Ecosystem() != "composer" {
		t.Errorf(
			"expected composer, got %s",
			found.Ecosystem(),
		)
	}
}

func TestRegistryDuplicateScanner(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Register(
		mockScanner{name: "composer"},
	); err != nil {
		t.Fatalf("register first scanner: %v", err)
	}

	err := registry.Register(
		mockScanner{name: "COMPOSER"},
	)

	if !errors.Is(err, ErrDuplicateScanner) {
		t.Fatalf(
			"expected ErrDuplicateScanner, got %v",
			err,
		)
	}
}

func TestRegistryRejectsEmptyEcosystem(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register(
		mockScanner{name: "  "},
	)

	if !errors.Is(err, ErrEmptyEcosystem) {
		t.Fatalf(
			"expected ErrEmptyEcosystem, got %v",
			err,
		)
	}
}

func TestRegistryReturnsSortedScanners(t *testing.T) {
	registry := NewRegistry()

	names := []string{
		"npm",
		"composer",
		"golang",
	}

	for _, name := range names {
		if err := registry.Register(
			mockScanner{name: name},
		); err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
	}

	scanners := registry.All()

	expected := []string{
		"composer",
		"golang",
		"npm",
	}

	if len(scanners) != len(expected) {
		t.Fatalf(
			"expected %d scanners, got %d",
			len(expected),
			len(scanners),
		)
	}

	for i, scanner := range scanners {
		if scanner.Ecosystem() != expected[i] {
			t.Errorf(
				"index %d: expected %s, got %s",
				i,
				expected[i],
				scanner.Ecosystem(),
			)
		}
	}
}
