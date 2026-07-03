// Package cli implements csm's user-facing commands (list, save, bookmark,
// open, clean) on top of the registry, hookcmd-populated data, and
// claudedir lookups.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Prompt writes label to w, reads one line from r, and returns it trimmed.
func Prompt(w io.Writer, r *bufio.Reader, label string) (string, error) {
	fmt.Fprint(w, label)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// SelectOne prints options as a 1-based numbered list and repeatedly prompts
// until the user enters a valid index, returning it zero-based.
func SelectOne(w io.Writer, r *bufio.Reader, prompt string, options []string) (int, error) {
	for i, o := range options {
		fmt.Fprintf(w, "%2d) %s\n", i+1, o)
	}
	for {
		ans, err := Prompt(w, r, prompt)
		if err != nil {
			return 0, err
		}
		n, err := strconv.Atoi(ans)
		if err != nil || n < 1 || n > len(options) {
			fmt.Fprintln(w, "invalid selection, try again")
			continue
		}
		return n - 1, nil
	}
}

// SelectMany prints options as a 1-based numbered list and prompts for a
// comma-separated list of indices (or "a"/"all"), returning them zero-based.
func SelectMany(w io.Writer, r *bufio.Reader, prompt string, options []string) ([]int, error) {
	for i, o := range options {
		fmt.Fprintf(w, "%2d) %s\n", i+1, o)
	}
	for {
		ans, err := Prompt(w, r, prompt)
		if err != nil {
			return nil, err
		}
		if ans == "a" || ans == "all" {
			idx := make([]int, len(options))
			for i := range options {
				idx[i] = i
			}
			return idx, nil
		}
		parts := strings.Split(ans, ",")
		idx := make([]int, 0, len(parts))
		ok := true
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			n, err := strconv.Atoi(part)
			if err != nil || n < 1 || n > len(options) {
				ok = false
				break
			}
			idx = append(idx, n-1)
		}
		if !ok || len(idx) == 0 {
			fmt.Fprintln(w, "invalid selection, try again")
			continue
		}
		return idx, nil
	}
}
