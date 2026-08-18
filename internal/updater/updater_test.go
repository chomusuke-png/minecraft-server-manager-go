package updater

import (
	"bufio"
	"strings"
	"testing"

	"minecraft-manager/internal/instance"
)

func readerFor(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}

func TestPreferredJavaPrefersInstanceOwn(t *testing.T) {
	meta := &instance.InstanceMeta{JavaPath: "runtimes/17/bin/java"}
	if got := preferredJava(meta, "java"); got != "runtimes/17/bin/java" {
		t.Errorf("got %q", got)
	}
}

func TestPreferredJavaFallsBackToGlobal(t *testing.T) {
	meta := &instance.InstanceMeta{}
	if got := preferredJava(meta, "java"); got != "java" {
		t.Errorf("got %q", got)
	}
}

func TestPromptLoaderTypeKeepsCurrentOnEmpty(t *testing.T) {
	if got := promptLoaderType(readerFor("\n"), "forge"); got != "forge" {
		t.Errorf("got %q", got)
	}
}

func TestPromptLoaderTypeSelectsByNumber(t *testing.T) {
	cases := map[string]string{
		"1\n": "paper",
		"2\n": "fabric",
		"3\n": "forge",
		"4\n": "neoforge",
		"5\n": "vanilla",
	}
	for input, want := range cases {
		if got := promptLoaderType(readerFor(input), "paper"); got != want {
			t.Errorf("promptLoaderType(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPromptLoaderTypeRetriesOnInvalidInput(t *testing.T) {
	if got := promptLoaderType(readerFor("9\n2\n"), "paper"); got != "fabric" {
		t.Errorf("got %q", got)
	}
}
