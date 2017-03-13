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
	// TODO: Change to using a general url to be less coupled to a particular site.
	MswPath string
	// Surf conditions at this spot
	Cond Quality
	// Legacy field. Ignored.
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

// SetSpotQuality adds a surfing condition quality to the spot with the given surf report path.
// If there is no such spot in datastore, SetSpotQuality creates one.
func SetSpotQuality(ctx context.Context, path string, qual *Quality) error {
	var s Spot
	key := SpotKey(ctx, path)
	err := datastore.Get(ctx, key, &s)
	if err == datastore.ErrNoSuchEntity {
		// This spot isn't yet in the db. That's okay, we'll just fill in the blanks and add it with the
		// quality.
		s.Name, err = surfReportPathToName(path)
		if err != nil {
			return fmt.Errorf("SetSpotQuality: %v", err)
		}
		s.MswPath = path
	} else if err != nil {
		return err
	}
	s.Cond = *qual
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
	return &s.Cond
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

// RatingExt gives the rating for the quality, or -1 if it is nil.
// This is a helper for sorting.
func (q *Quality) RatingExt() int {
	if q == nil {
		return -1
	}
	return q.Rating
}

// Sorting support
type ByRating []Spot

func (r ByRating) Len() int      { return len(r) }
func (r ByRating) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r ByRating) Less(i, j int) bool {
	rati := r[i].LatestQuality().RatingExt()
	ratj := r[j].LatestQuality().RatingExt()
	return rati > ratj || (rati == ratj && r[i].Name < r[j].Name)
}
