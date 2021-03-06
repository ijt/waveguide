package waveguide

// TODO: Change the error handling so users see non-technical error messages with at most an id that would correspond to something in the logs.

import (
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/urlfetch"
)

func init() {
	http.HandleFunc("/", handle(root))
	http.HandleFunc("/update_all", handle(updateAll))
	http.HandleFunc("/update_one", handle(updateOne))
	http.HandleFunc("/show", handle(show))
	http.HandleFunc("/clear", handle(clear))
	http.HandleFunc("/delete_one", handle(deleteOne))
	http.HandleFunc("/coords", handle(coords))
	http.HandleFunc("/clear_coords", handle(clearCoords))
	http.HandleFunc("/map", handle(doMap))
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
	if err := r.ParseForm(); err != nil {
		return http.StatusInternalServerError, err
	}
	sn := r.FormValue("n")
	if sn == "" {
		sn = "250"
	}
	n, err := strconv.Atoi(sn)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("Bad value %q for n parameter. Should be an integer.", sn)
	}
	log.Infof(ctx, "Limiting to top %d spots", n)
	q := datastore.NewQuery("Spot").Order("-Cond.Rating").Limit(n)
	var spots []Spot
	_, err = q.GetAll(ctx, &spots)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("root: fetching spots: %v", err)
	}
	sort.Sort(ByRating(spots))
	data := struct {
		Spots []Spot
	}{
		Spots: spots,
	}
	err = tmpl.ExecuteTemplate(w, "root", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("root: template: %v", err)
	}
	return http.StatusOK, nil
}

// updateAll fetches the site map from msw, saves Spots based on it, and adds tasks to update all the Spots.
func updateAll(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	// Get form arg n: number of spots to update, or -1 for all of them
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
	body, err := get(client, "http://magicseaweed.com/site-map.php")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	reportPaths := reportRx.FindAll(body, n)

	// For each report url:
	for _, rp := range reportPaths {
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
	data := struct{ Message string }{fmt.Sprintf("Processed %d paths.\n", len(reportPaths))}
	err = tmpl.ExecuteTemplate(w, "action_response", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("update_all: %v", err)
	}
	return http.StatusOK, nil
}

// updateOne updates the conditions for a single surfing spot, given the
// path to the surf report for it.
func updateOne(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if err := r.ParseForm(); err != nil {
		return http.StatusInternalServerError, err
	}
	su := r.FormValue("url")
	u, err := url.Parse(su)
	if err != nil {
		return http.StatusBadRequest, err
	}
	path := u.Path
	qual, err := fetchSpotQuality(ctx, su)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	SetSpotQuality(ctx, path, qual)
	data := struct{ Message string }{fmt.Sprintf("Updated %s", path)}
	err = tmpl.ExecuteTemplate(w, "action_response", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("update_one: %v", err)
	}
	return http.StatusOK, nil
}

// show shows the data for a single spot given its url, for debugging.
func show(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if err := r.ParseForm(); err != nil {
		return http.StatusInternalServerError, err
	}
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

// clear removes all the Spots from datastore. They can easily be brought back with updateAll.
func clear(ctx context.Context, w http.ResponseWriter, _ *http.Request) (int, error) {
	// TODO: This may time out. Rewrite it to queue up a bunch of tasks to delete individual spots or batches of
	// them.
	q := datastore.NewQuery("Spot")
	var spots []Spot
	_, err := q.GetAll(ctx, &spots)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	keys := make([]*datastore.Key, len(spots))
	for i, s := range spots {
		keys[i] = SpotKey(ctx, s.MswPath)
		t := taskqueue.NewPOSTTask("/delete_one", map[string][]string{"path": {s.MswPath}})
		_, err = taskqueue.Add(ctx, t, "")
		if err != nil {
			return http.StatusInternalServerError, err
		}
	}
	data := struct{ Message string }{fmt.Sprintf("Queued %d spots for deletion.\n", len(spots))}
	err = tmpl.ExecuteTemplate(w, "action_response", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("clear: %v", err)
	}
	return http.StatusOK, nil
}

// deleteOne deletes one Spot, given its MSW path as the "path" form value.
func deleteOne(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if err := r.ParseForm(); err != nil {
		return http.StatusInternalServerError, err
	}
	p := r.FormValue("path")
	key := SpotKey(ctx, p)
	err := datastore.Delete(ctx, key)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	data := struct{ Message string }{fmt.Sprintf("Deleted Spot %q\n", p)}
	err = tmpl.ExecuteTemplate(w, "action_response", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("delete_one: %v", err)
	}
	return http.StatusOK, nil
}

// coords sets the coordinates for a Spot.
func coords(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != "POST" {
		return http.StatusBadRequest, fmt.Errorf("Got %q, want POST.", r.Method)
	}
	if err := r.ParseForm(); err != nil {
		return http.StatusInternalServerError, err
	}
	p := r.FormValue("path")
	coordsStr := r.FormValue("coordinates")
	key := SpotKey(ctx, p)
	var s Spot
	err := datastore.Get(ctx, key, &s)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	m := coordsRx.FindStringSubmatch(coordsStr)
	if len(m) != 3 {
		return http.StatusBadRequest, fmt.Errorf("Got %q, want match to %q", coordsStr, coordsRx)
	}
	latStr := m[1]
	lngStr := m[2]
	lat, err := strconv.ParseFloat(latStr, 32)
	lng, err := strconv.ParseFloat(lngStr, 32)
	s.Coordinates.Lat = lat
	s.Coordinates.Lng = lng
	_, err = datastore.Put(ctx, key, &s)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	data := struct{ Message string }{"ok"}
	err = tmpl.ExecuteTemplate(w, "action_response", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("coords: %v", err)
	}
	return http.StatusOK, nil
}

// clearCoords clears the coordinates for a Spot.
// It's not done by a DELETE request to /coords because it has to be accessible from links on a page.
func clearCoords(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if err := r.ParseForm(); err != nil {
		return http.StatusInternalServerError, err
	}
	p := r.FormValue("path")
	key := SpotKey(ctx, p)
	var s Spot
	err := datastore.Get(ctx, key, &s)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	s.Coordinates = appengine.GeoPoint{}
	_, err = datastore.Put(ctx, key, &s)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	data := struct{ Message string }{"ok"}
	err = tmpl.ExecuteTemplate(w, "action_response", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("clear_coords: %v", err)
	}
	return http.StatusOK, nil
}

func doMap(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if err := r.ParseForm(); err != nil {
		return http.StatusInternalServerError, err
	}
	sn := r.FormValue("n")
	if sn == "" {
		sn = "250"
	}
	n, err := strconv.Atoi(sn)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("map: Bad value %q for n parameter. Should be an integer.", sn)
	}
	log.Infof(ctx, "Limiting to top %d spots", n)
	q := datastore.NewQuery("Spot").Order("-Cond.Rating").Limit(n)
	var spots []Spot
	_, err = q.GetAll(ctx, &spots)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("map: Fetching spots: %v", err)
	}
	sort.Sort(ByRating(spots))
	err = tmpl.ExecuteTemplate(w, "map", spots)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("map: %v", err)
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
	q := &Quality{
		Rating:     rating,
		WaveHeight: height,
		TimeUnix:   time.Now().Unix(),
	}
	return q, nil
}

// countTopStars returns the number of stars in the first rating section on the page.
func countTopStars(body []byte) int {
	starSection := starSectionRx.Find(body)
	foundStars := starRx.FindAll(starSection, -1)
	return len(foundStars)
}
