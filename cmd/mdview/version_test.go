package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/codewandler/markdown/terminal"
)

func TestVersionStringIncludesBuildMetadata(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	defer func() {
		version, commit, date = oldVersion, oldCommit, oldDate
	}()
	version = "v1.2.3"
	commit = "abc123"
	date = "2026-04-30"

	got := versionString()
	for _, want := range []string{"mdview v1.2.3", "commit abc123", "built 2026-04-30"} {
		if !strings.Contains(got, want) {
			t.Fatalf("versionString() = %q, missing %q", got, want)
		}
	}
}

func TestLiveModeFallsBackForNonTerminalWriter(t *testing.T) {
	enabled, fallback := liveMode(true, &bytes.Buffer{})
	if enabled || !fallback {
		t.Fatalf("liveMode(true, buffer) = (%v, %v), want (false, true)", enabled, fallback)
	}
}

func TestLiveModeDisabledWhenNotRequested(t *testing.T) {
	enabled, fallback := liveMode(false, &bytes.Buffer{})
	if enabled || fallback {
		t.Fatalf("liveMode(false, buffer) = (%v, %v), want (false, false)", enabled, fallback)
	}
}

func TestNewMarkdownRendererLiveFallbackUsesStreamRenderer(t *testing.T) {
	var out bytes.Buffer
	r := newMarkdownRenderer(&out, false)
	if _, ok := r.(*terminal.StreamRenderer); ok {
		return
	}
	t.Fatalf("renderer type = %T, want *terminal.StreamRenderer", r)
}

func TestRootCommandVersionFlag(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	defer func() {
		version, commit, date = oldVersion, oldCommit, oldDate
	}()
	version = "v9.9.9"
	commit = "deadbeef"
	date = "2026-04-30"

	var out, errOut bytes.Buffer
	cmd := newRootCommand(&out, &errOut)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(out.String())
	if !strings.Contains(got, "mdview v9.9.9") || !strings.Contains(got, "deadbeef") {
		t.Fatalf("version output = %q", got)
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}
}
