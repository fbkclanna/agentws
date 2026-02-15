package workspace

import (
	"testing"
)

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		input string
		want  Strategy
		err   bool
	}{
		{"safe", StrategySafe, false},
		{"stash", StrategyStash, false},
		{"reset", StrategyReset, false},
		{"", StrategySafe, false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseStrategy(tt.input)
			if (err != nil) != tt.err {
				t.Errorf("ParseStrategy(%q) error = %v, wantErr %v", tt.input, err, tt.err)
			}
			if got != tt.want {
				t.Errorf("ParseStrategy(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
