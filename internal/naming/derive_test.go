package naming

import "testing"

func TestDerive(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "feature with camelCase", in: "feature/loginForm", want: "feature-login-form"},
		{name: "bugfix multi camelCase", in: "bugfix/fixApiTimeout", want: "bugfix-fix-api-timeout"},
		{name: "already kebab", in: "hotfix-urgent", want: "hotfix-urgent"},
		{name: "single word", in: "simple", want: "simple"},
		{name: "empty string", in: "", want: ""},
		{name: "leading capital with camelCase", in: "User/profileSettings", want: "user-profile-settings"},
		{name: "multiple slashes with camelCase tail", in: "feature/A/B/cTest", want: "feature-a-b-c-test"},
		{name: "all caps", in: "ALLCAPS", want: "allcaps"},
		{name: "digit before uppercase", in: "with123Numbers", want: "with123-numbers"},
		{name: "multiple slashes mixed case", in: "multi/Slash/Path", want: "multi-slash-path"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := Derive(tc.in)
			if got != tc.want {
				t.Errorf("Derive(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
