package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootCommandHelpIncludesCompletion(t *testing.T) {
	cmd := newRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected help to execute cleanly, got %v", err)
	}

	out := buf.String()
	required := []string{
		"Schemagen introspects a database schema",
		"generate",
		"init",
		"completion",
		"schemagen init",
	}
	for _, token := range required {
		if !strings.Contains(out, token) {
			t.Fatalf("expected help output to contain %q, got:\n%s", token, out)
		}
	}
}

func TestRootCommandRejectsInvalidConflictPolicy(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"--dsn", "sqlite::memory:",
		"--driver", "sqlite",
		"--on-conflict", "bad",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid on_conflict") {
		t.Fatalf("expected invalid conflict policy error, got %v", err)
	}
}

func TestRootCommandRejectsInvalidRenderer(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"--dsn", "sqlite::memory:",
		"--driver", "sqlite",
		"--renderer", "ent",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid renderer") {
		t.Fatalf("expected invalid renderer error, got %v", err)
	}
}

func TestRootCommandRejectsInvalidNullableStrategy(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"--dsn", "sqlite::memory:",
		"--driver", "sqlite",
		"--nullable-strategy", "ptr",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid nullable_strategy") {
		t.Fatalf("expected invalid nullable strategy error, got %v", err)
	}
}

func TestGenerateCommandPrintsSummary(t *testing.T) {
	cmd := newRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"generate",
		"--dsn", "file::memory:?cache=shared",
		"--driver", "sqlite",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "INFO  no tables selected for generation") {
		t.Fatalf("expected no tables info, got %q", out)
	}
	if !strings.Contains(out, "INFO  generated=0") {
		t.Fatalf("expected summary info, got %q", out)
	}
}

func TestGenerateCommandQuietSuppressesInfo(t *testing.T) {
	cmd := newRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"generate",
		"--dsn", "file::memory:?cache=shared",
		"--driver", "sqlite",
		"--quiet",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected generate to succeed, got %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected quiet output to be empty, got %q", buf.String())
	}
}

func TestInitCommandWritesDefaultConfig(t *testing.T) {
	cmd := newRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	path := filepath.Join(t.TempDir(), "schemagen.yaml")
	relationsPath := filepath.Join(filepath.Dir(path), "schemagen.relations.yaml")
	cmd.SetArgs([]string{"init", "--path", path, "--relations-path", relationsPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected init to succeed, got %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
	if string(data) != defaultConfigTemplate() {
		t.Fatalf("unexpected config content:\n%s", string(data))
	}
	relationsData, err := os.ReadFile(relationsPath)
	if err != nil {
		t.Fatalf("expected relations config file to exist: %v", err)
	}
	if string(relationsData) != defaultRelationsTemplate() {
		t.Fatalf("unexpected relations config content:\n%s", string(relationsData))
	}
	if !strings.Contains(buf.String(), "wrote "+path) {
		t.Fatalf("expected output to mention path, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "wrote "+relationsPath) {
		t.Fatalf("expected output to mention relations path, got %q", buf.String())
	}
}

func TestInitCommandRejectsExistingWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schemagen.yaml")
	if err := os.WriteFile(path, []byte("dsn: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--path", path})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected init existing file error, got %v", err)
	}
}
