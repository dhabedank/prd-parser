package version

import "testing"

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name    string
		latest  string
		current string
		want    bool
	}{
		{"same version", "1.0.0", "1.0.0", false},
		{"patch newer", "1.0.1", "1.0.0", true},
		{"minor newer", "1.1.0", "1.0.0", true},
		{"major newer", "2.0.0", "1.0.0", true},
		{"current newer", "1.0.0", "1.0.1", false},
		{"v prefix handled", "0.4.1", "0.4.0", true},
		{"longer version newer", "1.0.0.1", "1.0.0", true},
		{"double digit", "1.10.0", "1.9.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNewerVersion(tt.latest, tt.current)
			if got != tt.want {
				t.Errorf("isNewerVersion(%q, %q) = %v, want %v",
					tt.latest, tt.current, got, tt.want)
			}
		})
	}
}

func TestParseVersionPart(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"1", 1},
		{"10", 10},
		{"0", 0},
		{"1-beta", 1},
		{"2-rc1", 2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseVersionPart(tt.input)
			if got != tt.want {
				t.Errorf("parseVersionPart(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
