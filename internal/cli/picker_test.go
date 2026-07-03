package cli

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestSelectOne_ValidChoice(t *testing.T) {
	var out bytes.Buffer
	r := bufio.NewReader(strings.NewReader("2\n"))

	idx, err := SelectOne(&out, r, "choose: ", []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("SelectOne() error = %v", err)
	}
	if idx != 1 {
		t.Fatalf("idx = %d, want 1", idx)
	}
}

func TestSelectOne_RetriesOnInvalidInput(t *testing.T) {
	var out bytes.Buffer
	r := bufio.NewReader(strings.NewReader("nope\n99\n1\n"))

	idx, err := SelectOne(&out, r, "choose: ", []string{"a", "b"})
	if err != nil {
		t.Fatalf("SelectOne() error = %v", err)
	}
	if idx != 0 {
		t.Fatalf("idx = %d, want 0", idx)
	}
	if !strings.Contains(out.String(), "invalid selection") {
		t.Fatalf("expected a retry message, got: %s", out.String())
	}
}

func TestSelectMany_CommaList(t *testing.T) {
	var out bytes.Buffer
	r := bufio.NewReader(strings.NewReader("1, 3\n"))

	idx, err := SelectMany(&out, r, "choose: ", []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(idx) != 2 || idx[0] != 0 || idx[1] != 2 {
		t.Fatalf("idx = %v, want [0 2]", idx)
	}
}

func TestSelectMany_All(t *testing.T) {
	var out bytes.Buffer
	r := bufio.NewReader(strings.NewReader("all\n"))

	idx, err := SelectMany(&out, r, "choose: ", []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(idx) != 3 {
		t.Fatalf("idx = %v, want all 3 indices", idx)
	}
}
