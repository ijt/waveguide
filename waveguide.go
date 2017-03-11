package waveguide

import (
	"net/http"
	"net/url"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
	"fmt"
	"io/ioutil"
	"regexp"
	"html"
	"strings"
	"time"
	"google.golang.org/appengine/urlfetch"
)

func init() {
	http.HandleFunc("/", handle(root))
	http.HandleFunc("/update_all", handle(updateAll))
	http.HandleFunc("/update_one", handle(updateOne))
}

func handle(f func(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		status, err := f(ctx, w, r)
		if err != nil {
			log.Errorf(ctx, "%v", err)
			w.WriteHeader(status)
			fmt.Fprintf(w, "%v", err)
		}
	}
}

// root shows the main page.
func root(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	fmt.Fprintf(w, "Waveguide")
	return http.StatusOK, nil
}

func updateAll(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	// Download magicseaweed.com/site-map.php
	client := urlfetch.Client(ctx)
	body, err := get(client,"http://magicseaweed.com/site-map.php")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	reportPaths := reportRx.FindAll(body, -1)

	// For each report url:
	for _, rp := range reportPaths {
		// TODO: See about using PutMulti if this needs to be more
		// efficient.
		err = saveNewSurfReportPath(ctx, rp)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		// Add a task to the queue to download the report and store its stars and wave
		// height to the db.
		srp := string(rp)
		t := taskqueue.NewPOSTTask("/update_one", map[string][]string{"path": {srp}})
		if _, err := taskqueue.Add(ctx, t, ""); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return http.StatusInternalServerError, err
		}
	}

	fmt.Fprintf(w,"Processed %d paths.", len(reportPaths))

	return http.StatusOK, nil
}

// updateOne updates the conditions for a single surfing spot.
func updateOne(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	path := r.FormValue("path")
	log.Infof(ctx, "updateOne: path=%q", path)
	key := datastore.NewKey(ctx, "Spot", path, 0, nil)
	var s Spot
	if err := datastore.Get(ctx, key, &s); err == datastore.ErrNoSuchEntity {
		url := "http://magicseaweed.com" + s.MswPath
		q, err := fetchQuality(ctx, url)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		s.Qual = append(s.Qual, *q)
		_, err = datastore.Put(ctx, key, &s)
		if err != nil {
			return http.StatusInternalServerError, err
		}
	}
	return http.StatusOK, nil
}

func get(client *http.Client, url string) ([]byte, error) {
	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read body of %s", url)
	}
	return body, nil
}

var starSectionRx = regexp.MustCompile(`<ul class="rating rating-large clearfix">.*?</ul>`)
var starRx = regexp.MustCompile(`<li class="active"> *<i class="glyphicon glyphicon-star"></i> *</li>`)
var heightRx = regexp.MustCompile(`(\d+(?:-\d+)?)<small>ft`)
var reportRx = regexp.MustCompile(`/[^"/]+-Surf-Report/\d+/`)
var srpTailRx = regexp.MustCompile(`-Surf-Report/\d+/`)

type Spot struct {
	Name 		 string
	MswPath		 string
	// Surf conditions at this spot, most recent last
	Qual		 []Quality
}

type Quality struct {
	// How many stars out of five
	Rating int
	WaveHeight string
	Time time.Time
}

func surfReportPathToName(srp string) (string, error) {
	s := srpTailRx.ReplaceAllString(srp, "")
	// TODO: Use PathUnescape once GAE supports Go 1.8
	s, err := url.QueryUnescape(s)
	if err != nil {
		return "", fmt.Errorf("Failed to unescape path in surfReportPathToName(%q). %v", srp, err)
	}
	s = html.UnescapeString(s)
	s = s[1:] // Remove leading /
	s = strings.Replace(s, "-", " ", -1)
	return s, nil
}

func saveNewSurfReportPath(ctx context.Context, path []byte) error {
	sp := string(path)
	name, err := surfReportPathToName(sp)
	if err != nil {
		return fmt.Errorf("saveNewSurfReportPaths: %v", err)
	}
	loc := Spot{Name: name, MswPath: sp}
	key := datastore.NewKey(ctx, "Spot", string(sp), 0, nil)
	// TODO: First check whether there is already something at this key.
	_, err = datastore.Put(ctx, key, &loc)
	if err != nil {
		return fmt.Errorf("saveNewSurfReportPaths: %v", err)
	}
	return nil
}

// fetchQuality scrapes the surf report at the given url and returns a Quality struct
// summarizing the conditions.
func fetchQuality(ctx context.Context, url string) (*Quality, error) {
	log.Infof(ctx, "Fetching %s", url)
	client := urlfetch.Client(ctx)
	body, err := get(client, url)
	if err != nil {
		return nil, err
	}
	rating := countTopStars(body)
	hMatch := heightRx.FindSubmatch(body)
	if len(hMatch) != 2 {
		return nil, fmt.Errorf("Wave height regex failed.")
	}
	height := fmt.Sprintf("%s ft", hMatch[1])
	q := &Quality {
		Rating: rating,
		WaveHeight: height,
		Time: time.Now(),
	}
	return q, nil
}

// countTopStars returns the number of stars in the first rating section on the page.
func countTopStars(body []byte) int {
	starSection := starSectionRx.Find(body)
	foundStars := starRx.FindAll(starSection, -1)
	return len(foundStars)
}
