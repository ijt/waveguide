package main

import "testing"

func TestHeightRx(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"   1<small>ft</small>  ", "1"},
		{"   3-6<small>ft</small>  ", "3-6"},
	}

	for _, c := range cases {
		ssm := heightRx.FindStringSubmatch(c.in)
		if len(ssm) != 2 {
			t.Errorf("Wrong length for submatches: %d", len(ssm))
			continue
		}
		if ssm[1] != c.want {
			t.Errorf("For %q got %q, want %q", c.in, ssm[1], c.want)
		}
	}
}

func TestForecastRx(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`<option value="/Manhattan-Beach-Surf-Report/281/">Manhattan Beach</option>`, []string{"/Manhattan-Beach-Surf-Report/281/", "Manhattan Beach"}},
		{`<a href="/Linda-Mar-Pacifica-Surf-Report/819/" style="color: #000000">Linda Mar / Pacifica</a>`, []string{"/Linda-Mar-Pacifica-Surf-Report/819/", "Linda Mar / Pacifica"}},
	}

	for _, c := range cases {
		ssm := reportRx.FindStringSubmatch(c.in)
		if len(ssm) != 3 {
			t.Errorf("Got %d submatches for %s, want 3", len(ssm), c.in)
			continue
		}
		if ssm[1] != c.want[0] || ssm[2] != c.want[1] {
			t.Errorf("Got %v for rest of submatches, want %v", ssm[1:], c.want)
		}
	}
}
