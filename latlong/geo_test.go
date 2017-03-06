package main

import (
	"testing"
)

func TestParseCoords(t *testing.T) {
	exes := []struct {
		in string
		want Coordinates
	} {
		{`33.801944° N 118.401028°E`, Coordinates{33.801944, 118.401028}},
		{`33.801944° S 118.401028°W`, Coordinates{-33.801944, -118.401028}},
		{`-33.801944,118.401028`, Coordinates{-33.801944, 118.401028}},
	}
	for _, e := range exes {
		got, err := parseCoords(e.in)
		if err != nil {
			t.Errorf("Want no error for %q, got %v", e.in, err)
			continue
		}
		if got.String() != e.want.String() {
			t.Errorf("parseCoords(%q) = %v, want %v", e.in, got, e.want)
		}
	}
}