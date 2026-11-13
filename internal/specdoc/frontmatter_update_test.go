package specdoc

import (
	"strings"
	"testing"
)

func TestReplaceFrontmatterScalarPreservesBodyAndLineEndings(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		body := "## Planning Context\n\n```yaml\nstatus: available\nspec_rev: upstream-4\nlast_updated_at: 2026-11-10T08:00:00Z\n```\n"
		source := "---\nstatus: active\nspec_rev: rev-1\nlast_updated_at: 2026-11-09T08:00:00Z\n---\n" + body
		source = strings.ReplaceAll(source, "\n", newline)
		for key, value := range map[string]string{"status": "done", "spec_rev": "rev-2", "last_updated_at": "2026-11-10T10:00:00Z"} {
			updated, err := ReplaceFrontmatterScalar(source, key, value)
			if err != nil {
				t.Fatalf("%q: %v", newline, err)
			}
			if !strings.HasSuffix(updated, strings.ReplaceAll(body, "\n", newline)) {
				t.Fatal("body changed during metadata update")
			}
			if !strings.Contains(updated, newline+key+": "+value+newline) {
				t.Fatalf("header field %s not updated", key)
			}
			if newline == "\r\n" && strings.Contains(strings.ReplaceAll(updated, "\r\n", ""), "\n") {
				t.Fatal("line endings changed")
			}
		}
	}
}

func TestReplaceFrontmatterScalarRequiresOneHeaderField(t *testing.T) {
	for _, source := range []string{
		"status: active\n",
		"---\nid: cycle-01-orders\n---\nstatus: body-only\n",
		"---\nstatus: active\nstatus: blocked\n---\n",
	} {
		if _, err := ReplaceFrontmatterScalar(source, "status", "done"); err == nil {
			t.Fatalf("accepted invalid header: %q", source)
		}
	}
}

func TestReplaceFrontmatterScalarKeepsReplacementLiteral(t *testing.T) {
	source := "---\nstatus: active\n---\n\nBody.\n"
	updated, err := ReplaceFrontmatterScalar(source, "status", "$1 and $value")
	if err != nil || updated != "---\nstatus: $1 and $value\n---\n\nBody.\n" {
		t.Fatalf("updated=%q, error=%v", updated, err)
	}
	if _, err := ReplaceFrontmatterScalar(source, "status", "active\nnext: injected"); err == nil {
		t.Fatal("accepted a multiline value")
	}
}
