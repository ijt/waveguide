package waveguide

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

type Spot struct {
	Name string
	// TODO: Change to using a general url to be less coupled
	// to a particular site.
	MswPath string
	// Surf conditions at this spot, most recent last
	Qual []Quality
}

type Quality struct {
	// How many stars out of five
	Rating     int
	WaveHeight string
	TimeUnix   int64
}

func SpotKey(ctx context.Context, mswPath string) *datastore.Key {
	return datastore.NewKey(ctx, "Spot", mswPath, 0, nil)
}

// AddQualityToSpot adds a surfing condition quality to the spot with the given surf report path.
// If there is no such spot in datastore, AddQualityToSpot creates one.
func AddQualityToSpot(ctx context.Context, path string, qual *Quality) error {
	var s Spot
	key := SpotKey(ctx, path)
	err := datastore.Get(ctx, key, &s)
	if err == datastore.ErrNoSuchEntity {
		// This spot isn't yet in the db. That's okay, we'll just fill in the blanks and add it with the
		// quality.
		s.Name, err = surfReportPathToName(path)
		if err != nil {
			return fmt.Errorf("AddQualityToSpot: %v", err)
		}
		s.MswPath = path
	} else if err != nil {
		return err
	}
	s.Qual = append(s.Qual, *qual)
	for len(s.Qual) > maxQualitiesPerSpot {
		drop := len(s.Qual) - maxQualitiesPerSpot
		s.Qual = s.Qual[drop:len(s.Qual)]
	}
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

func (s *Spot) ReportURL() string {
	return "http://magicseaweed.com" + s.MswPath
}

func (s *Spot) MapURL() string {
	return strings.Replace(s.ReportURL(), "Report", "Guide", 1)
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

var firstAlphaRx = regexp.MustCompile(`([a-z]+).*`)
var decimalRx = regexp.MustCompile(`\.\d+`)

// HowLong returns a string telling how long it has been since the time given by its TimeUnix field.
func (q *Quality) HowLong() string {
	t := time.Unix(q.TimeUnix, 0)
	dt := time.Now().Sub(t)
	s := dt.String()
	s = firstAlphaRx.ReplaceAllString(s, "$1")
	s = decimalRx.ReplaceAllString(s, "")
	return s
}

type ByRating []Spot

func (r ByRating) Len() int      { return len(r) }
func (r ByRating) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r ByRating) Less(i, j int) bool {
	qi := r[i].LatestQuality()
	qj := r[j].LatestQuality()
	if qi == nil {
		if qj == nil {
			return true
		} else {
			return false
		}
	} else {
		if qj == nil {
			return true
		} else {
			return qi.Rating > qj.Rating
		}
	}
}
