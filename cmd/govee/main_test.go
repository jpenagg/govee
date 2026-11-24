package main

import (
	"slices"
	"testing"
)

func TestParseSegments(t *testing.T) {
	tests := map[string][]int{
		"3":        {3},
		"0-4":      {0, 1, 2, 3, 4},
		"0-2,7, 9": {0, 1, 2, 7, 9},
	}
	for in, want := range tests {
		got, err := parseSegments(in)
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("parseSegments(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "a", "4-2", "1-", "1,,2"} {
		if _, err := parseSegments(bad); err == nil {
			t.Errorf("parseSegments(%q) succeeded", bad)
		}
	}
}
