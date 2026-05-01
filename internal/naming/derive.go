// Package naming provides pure functions for deriving and validating
// worktree directory names from git branch names.
package naming

import (
	"strings"
	"unicode"
)

// Derive converts a git branch name into a kebab-case worktree directory
// name.
//
// The transformation is:
//  1. Replace every '/' with '-'.
//  2. Insert '-' between a lowercase-letter-or-digit and an uppercase
//     letter (camelCase to kebab-case).
//  3. Lowercase the result.
//
// The function is pure and idempotent: applying it twice produces the same
// result as applying it once for any already-kebab input.
func Derive(branch string) string {
	if branch == "" {
		return ""
	}

	// Step 1: '/' -> '-'.
	s := strings.ReplaceAll(branch, "/", "-")

	// Step 2: insert '-' between [a-z0-9] and [A-Z], left-to-right,
	// non-overlapping (matching sed's default behavior).
	var b strings.Builder
	b.Grow(len(s) + 4)
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 {
			prev := runes[i-1]
			if (unicode.IsLower(prev) || unicode.IsDigit(prev)) && unicode.IsUpper(r) {
				b.WriteByte('-')
			}
		}
		b.WriteRune(r)
	}

	// Step 3: lowercase.
	return strings.ToLower(b.String())
}
