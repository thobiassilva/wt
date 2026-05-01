package naming

import (
	"strings"
	"testing"
)

func TestValidateWorktreeName(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantErr   bool
		errSubstr string
	}{
		{name: "empty", in: "", wantErr: true, errSubstr: "empty"},
		{name: "contains dot dot escape", in: "../escape", wantErr: true, errSubstr: ".."},
		{name: "absolute path", in: "/abs/path", wantErr: true, errSubstr: "absolute path"},
		{name: "with space", in: "with space", wantErr: true, errSubstr: "space"},
		{name: "valid kebab", in: "feature-login", wantErr: false},
		{name: "single dot is fine", in: "bugfix.fix", wantErr: false},
		{name: "non-leading slash is fine", in: "a/b", wantErr: false},
		{name: "dot dot embedded", in: "foo..bar", wantErr: true, errSubstr: ".."},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateWorktreeName(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ValidateWorktreeName(%q) = nil, want error containing %q", tc.in, tc.errSubstr)
				}
				if tc.errSubstr != "" && !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("ValidateWorktreeName(%q) error = %q, want substring %q", tc.in, err.Error(), tc.errSubstr)
				}
				return
			}
			if err != nil {
				t.Errorf("ValidateWorktreeName(%q) returned unexpected error: %v", tc.in, err)
			}
		})
	}
}
