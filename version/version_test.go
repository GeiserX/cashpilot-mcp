package version

import "testing"

func TestString_returns_formatted_version_string(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		want    string
	}{
		{
			name:    "default dev values",
			version: "dev",
			commit:  "none",
			date:    "unknown",
			want:    "dev (none) unknown",
		},
		{
			name:    "release values",
			version: "1.2.3",
			commit:  "abc1234",
			date:    "2025-01-15",
			want:    "1.2.3 (abc1234) 2025-01-15",
		},
		{
			name:    "empty values produce correct delimiters",
			version: "",
			commit:  "",
			date:    "",
			want:    " () ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origV, origC, origD := Version, Commit, Date
			t.Cleanup(func() { Version, Commit, Date = origV, origC, origD })

			Version = tc.version
			Commit = tc.commit
			Date = tc.date

			got := String()
			if got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}
