package waveguide

import (
	"net/url"
	"reflect"
	"testing"

	"google.golang.org/appengine/aetest"
	"google.golang.org/appengine/datastore"
)

func TestPutAndGetSpot(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	spots := []Spot{
		{
			Name:    "Pacifica",
			MswPath: "/pacifica-surf-report/111",
		},
		{
			Name:    "Pacifica",
			MswPath: "/pacifica-surf-report/111",
			Cond:    Quality{Rating: 5, WaveHeight: "7 ft", TimeUnix: 5},
		},
	}
	for _, s := range spots {
		key := SpotKey(ctx, s.MswPath)
		_, err = datastore.Put(ctx, key, &s)
		if err != nil {
			t.Errorf("Got %v, want no error", err)
			continue
		}
		var s2 Spot
		err = datastore.Get(ctx, key, &s2)
		if err != nil {
			t.Errorf("Got %v, want no error", err)
			continue
		}
		got := s2
		want := s
		if want.Name != got.Name {
			t.Fatalf("Name: Got %q, want %q", got, want)
		}
		if want.MswPath != got.MswPath {
			t.Fatalf("MswPath: Got %q, want %q", got, want)
		}
		if !reflect.DeepEqual(want.Cond, got.Cond) {
			t.Fatalf("Cond: Got %+v, want %+v", got.Cond, want.Cond)
		}
	}
}

func TestAddQualityToSpot(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	s := Spot{
		Name:    "Cowell",
		MswPath: "/cowell-surf-report/345",
	}
	key := SpotKey(ctx, s.MswPath)
	_, err = datastore.Put(ctx, key, &s)
	if err != nil {
		t.Fatalf("Put s: Got %v, want no error", err)
	}
	var s2 Spot
	err = datastore.Get(ctx, key, &s2)
	if err != nil {
		t.Fatalf("Get s2: Got %v, want no error", err)
	}
	s2.Cond = Quality{Rating: 5, WaveHeight: "7 ft", TimeUnix: 5}
	_, err = datastore.Put(ctx, key, &s2)
	if err != nil {
		t.Fatalf("Put s2: Got %v, want no error", err)
	}
	var s3 Spot
	err = datastore.Get(ctx, key, &s3)
	if err != nil {
		t.Fatalf("Get s3: Got %v, want no error", err)
	}
	want := s2
	got := s3
	if want.Name != got.Name {
		t.Fatalf("Name: Got %q, want %q", got, want)
	}
	if want.MswPath != got.MswPath {
		t.Fatalf("MswPath: Got %q, want %q", got, want)
	}
	if !reflect.DeepEqual(want.Cond, got.Cond) {
		t.Fatalf("Cond: Got %+v, want %+v", got.Cond, want.Cond)
	}
}

func TestClearCoordsURL(t *testing.T) {
	var s Spot
	s.MswPath = "/Pacifica-Surf-Report/123"
	u := s.ClearCoordsURL()
	u2, err := url.Parse(u)
	if err != nil {
		t.Fatalf("parse: Got %v, want no error", err)
	}
	q := u2.Query()
	p := q.Get("path")
	if p != s.MswPath {
		t.Errorf("Got %q for path, want %q", p, s.MswPath)
	}
}
