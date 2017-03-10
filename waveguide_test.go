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

func TestForecastRx(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{`<option value="/Manhattan-Beach-Surf-Report/281/">Manhattan Beach</option>`, "/Manhattan-Beach-Surf-Report/281/"},
		{`<a href="/Linda-Mar-Pacifica-Surf-Report/819/" style="color: #000000">Linda Mar / Pacifica</a>`, "/Linda-Mar-Pacifica-Surf-Report/819/"},
	}

	for _, c := range cases {
		got := reportRx.FindString(c.input)
		if got != c.want {
			t.Errorf("Got %q for %q, want %q", got, c.input, c.want)
		}
	}
}

func TestSurfReportPathToName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"/Playa-Santa-Teresa-Surf-Report/914/", "Playa Santa Teresa"},
		{"/ColA%26iexcl%3Bn-Surf-Report/3288/", "ColA¡n"},
	}
	for _, c := range cases {
		got, err := surfReportPathToName(c.input)
		if err != nil {
			t.Errorf("Want no error, got %v", err)
			continue
		}
		if got != c.want {
			t.Errorf("Got %s, want %s", got, c.want)
		}
	}
}
