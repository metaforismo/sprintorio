package machine

import "testing"

func TestSafeGitHubPullRequestURL(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		repository string
		want       bool
	}{
		{name: "valid", value: "https://github.com/metaforismo/sprintorio/pull/46", repository: "metaforismo/sprintorio", want: true},
		{name: "case insensitive repository", value: "https://github.com/Metaforismo/Sprintorio/pull/46", repository: "metaforismo/sprintorio", want: true},
		{name: "http", value: "http://github.com/metaforismo/sprintorio/pull/46", repository: "metaforismo/sprintorio"},
		{name: "lookalike host", value: "https://github.com.example.com/metaforismo/sprintorio/pull/46", repository: "metaforismo/sprintorio"},
		{name: "userinfo", value: "https://github.com@evil.example/metaforismo/sprintorio/pull/46", repository: "metaforismo/sprintorio"},
		{name: "other repository", value: "https://github.com/attacker/sprintorio/pull/46", repository: "metaforismo/sprintorio"},
		{name: "repository prefix", value: "https://github.com/metaforismo/sprintorio-malicious/pull/46", repository: "metaforismo/sprintorio"},
		{name: "non pull request", value: "https://github.com/metaforismo/sprintorio/issues/46", repository: "metaforismo/sprintorio"},
		{name: "invalid pull request number", value: "https://github.com/metaforismo/sprintorio/pull/zero", repository: "metaforismo/sprintorio"},
		{name: "pull request subpage", value: "https://github.com/metaforismo/sprintorio/pull/46/files", repository: "metaforismo/sprintorio"},
		{name: "query", value: "https://github.com/metaforismo/sprintorio/pull/46?redirect=https://evil.example", repository: "metaforismo/sprintorio"},
		{name: "malformed repository", value: "https://github.com/metaforismo/sprintorio/pull/46", repository: "metaforismo/sprintorio/extra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := safeGitHubPullRequestURL(tt.value, tt.repository)
			if ok != tt.want {
				t.Fatalf("safeGitHubPullRequestURL() ok = %v, want %v", ok, tt.want)
			}
			if tt.want && got != tt.value {
				t.Fatalf("safeGitHubPullRequestURL() = %q, want %q", got, tt.value)
			}
			if !tt.want && got != "" {
				t.Fatalf("safeGitHubPullRequestURL() returned rejected URL %q", got)
			}
		})
	}
}
