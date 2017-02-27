package waveguide

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
