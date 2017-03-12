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
	"strconv"
	"html/template"
	"sort"
)

func init() {
	http.HandleFunc("/", handle(root))
	http.HandleFunc("/update_all", handle(updateAll))
	http.HandleFunc("/update_one", handle(updateOne))
	http.HandleFunc("/show", handle(show))
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
	if r.URL.Path != "/" {
		return http.StatusNotFound, fmt.Errorf("%s not found", r.URL.Path)
	}
	q := datastore.NewQuery("Spot")
	var spots []Spot
	log.Infof(ctx, "root: Before GetAll")
	_, err := q.GetAll(ctx, &spots)
	log.Infof(ctx, "root: After GetAll")
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf( "root: fetching spots: %v", err)
	}
	sort.Sort(ByRating(spots))
	data := struct {
		Head template.HTML
		Spots []Spot
	} {
		Head: head,
		Spots: spots,
	}
	err = rootTmpl.Execute(w, data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("root: template: %v", err)
	}
	return http.StatusOK, nil
}

func updateAll(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if err := r.ParseForm(); err != nil {
		return http.StatusInternalServerError, err
	}
	sn := r.FormValue("n")
	if sn == "" {
		sn = "-1"
	}
	n, err := strconv.Atoi(sn)

	// Download magicseaweed.com/site-map.php
	client := urlfetch.Client(ctx)
	body, err := get(client,"http://magicseaweed.com/site-map.php")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	reportPaths := reportRx.FindAll(body, n)

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
		su := "http://magicseaweed.com" + srp
		t := taskqueue.NewPOSTTask("/update_one", map[string][]string{"url": {su}})
		if _, err := taskqueue.Add(ctx, t, ""); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return http.StatusInternalServerError, err
		}
	}

	fmt.Fprintf(w,"Processed %d paths.", len(reportPaths))

	return http.StatusOK, nil
}

// updateOne updates the conditions for a single surfing spot, given the
// path to the surf report for it.
func updateOne(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if err := r.ParseForm(); err != nil {
		return http.StatusInternalServerError, err
	}
	su := r.FormValue("url")
	log.Infof(ctx, "/update_one: url=%q", su)
	u, err := url.Parse(su)
	if err != nil {
		return http.StatusBadRequest, err
	}
	path := u.Path
	qual, err := fetchSpotQuality(ctx, su)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	AddQualityToSpot(ctx, SpotKey(ctx, path), qual)
	w.Write([]byte("ok"))
	return http.StatusOK, nil
}

func show(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if err := r.ParseForm(); err != nil {
		return http.StatusInternalServerError, err
	}
	log.Infof(ctx, "/show. form: %v", r.Form)
	su := r.FormValue("url")
	u, err := url.Parse(su)
	if err != nil {
		return http.StatusBadRequest, err
	}
	path := u.Path
	key := SpotKey(ctx, path)
	var s Spot
	if err := datastore.Get(ctx, key, &s); err != nil {
		return http.StatusInternalServerError, err
	}
	fmt.Fprintf(w, "%+v", s)
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
	s := Spot{Name: name, MswPath: sp}
	key := SpotKey(ctx, sp)
	// TODO: First check whether there is already something at this key.
	_, err = datastore.Put(ctx, key, &s)
	if err != nil {
		return fmt.Errorf("saveNewSurfReportPaths: %v", err)
	}
	return nil
}

// fetchSpotQuality scrapes the surf report at the given url and returns a Quality struct
// summarizing the conditions.
func fetchSpotQuality(ctx context.Context, url string) (*Quality, error) {
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
		TimeUnix: time.Now().Unix(),
	}
	return q, nil
}

// countTopStars returns the number of stars in the first rating section on the page.
func countTopStars(body []byte) int {
	starSection := starSectionRx.Find(body)
	foundStars := starRx.FindAll(starSection, -1)
	return len(foundStars)
}
