package waveguide

import (
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
	"html/template"
	"time"
	"regexp"
)

type Spot struct {
	Name 		 string
	// TODO: Change to using a general url to be less coupled
	// to a particular site.
	MswPath		 string
	// Surf conditions at this spot, most recent last
	Qual		 []Quality
}

type Quality struct {
	// How many stars out of five
	Rating int
	WaveHeight string
	TimeUnix int64
}

func SpotKey(ctx context.Context, mswPath string) *datastore.Key {
	return datastore.NewKey(ctx, "Spot", mswPath, 0, nil)
}

func AddQualityToSpot(ctx context.Context, key *datastore.Key, qual *Quality) error {
	var s Spot
	err := datastore.Get(ctx, key, &s)
	if err != nil {
		return err
	}
	// TODO: limit how many qualities we add to a spot.
	s.Qual = append(s.Qual, *qual)
	_, err = datastore.Put(ctx, key, &s)
	if err != nil {
		return err
	}
	return nil
}

func (s *Spot) HTMLName() template.HTML {
	return template.HTML(s.Name)
}

func (s *Spot) LatestQuality() *Quality {
	if len(s.Qual) == 0 {
		return nil
	}
	return &s.Qual[len(s.Qual)-1]
}

func (q *Quality) Stars() string {
	runes := make([]rune, 0, 5)
	for i := 0; i < q.Rating; i++ {
		runes = append(runes, '★')
	}
	for i := 0; i < 5-q.Rating; i++ {
		runes = append(runes, '☆')
	}
	return string(runes)
}

var dotNumRx = regexp.MustCompile(`\.\d+`)

// HowLong returns a string telling how long it has been since the time given by its TimeUnix field.
func (q *Quality) HowLong() string {
	t := time.Unix(q.TimeUnix, 0)
	dt := time.Now().Sub(t)
	s := dt.String()
	return dotNumRx.ReplaceAllString(s, "")
}