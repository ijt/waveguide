package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var addr = flag.String("addr", ":4089", "Server address")
var timeout = flag.Duration("timeout", 5*time.Second, "Timeout for HTTP GETs")
var conditionsInterval = flag.Duration("conditions_interval", 10*time.Minute, "Interval between updating conditions")
var siteMapUrl = flag.String("site_map_url", "http://magicseaweed.com/site-map.php", "URL of the site map")
var conditionsFilePath = flag.String("conditions_file", "conditions.json", "Path to file with conditions cache")
var conditionsPeriod = flag.Duration("conditions_period", 10*time.Second, "How often to save the conditions file")
var help = flag.Bool("h", false, "Show help message")

func main() {
	flag.Parse()
	if *help {
		flag.Usage()
		os.Exit(0)
	}
	loadConditionsFile()
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/errors", handleErrors)
	go keepConditionsUpdated()
	log.Printf("Listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var head = template.HTML(`
	<head>
		<title>Waveguide</title>
		<style>
			body {
				font-family: monospace;
			}
			table {
				border-collapse: separate;
				font-size: 12pt;
			}
			th {
				text-align: left;
			}
			th, td {
				padding: 0 1em 0.5ex 0;
			}
		</style>
	</head>
`)

var errsTmpl = template.Must(template.New("errs").Parse(`
<html>
{{.Head}}
	<body>
		<b>Errors</b>:
		<table>
			{{range .Errs}}
			<tr>
				<td><a href="http://magicseaweed.com{{.Loc.MagicSeaweedPath}}">{{.Loc.HTMLName}}</a></td>
				<td>{{.Err}}</td>
			</tr>
			{{end}}
		</table>
	</body>
</html>
`))

var tmpl = template.Must(template.New("main").Parse(`
<html>
{{.Head}}
	<body>
		{{if .Conds}}
		<table>
			<thead>
				<th>Location</th>
				<th>Conditions</th>
				<th>Wave Height</th>
			</thead>
			<tbody>
				{{range .Conds}}
				<tr>
					<td><a href="http://magicseaweed.com{{.Loc.MagicSeaweedPath}}">{{.Loc.HTMLName}}</a></td>
					<td>{{.Stars}}</td>
					<td>{{.Details}}</td>
				</tr>
				{{end}}
			</tbody>
		</table>
		{{end}}
	</body>
</html>
`))

type Location struct {
	Name             string
	MagicSeaweedPath string
}

func (loc *Location) HTMLName() template.HTML {
	return template.HTML(loc.Name)
}

type Conditions struct {
	Loc     *Location
	Rating  int
	Details string
}

type Error struct {
	Loc *Location
	Err error
}

func handleErrors(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	data := struct {
		Errs []*Error
		Head template.HTML
	}{
		Errs: errs,
		Head: head,
	}
	err := errsTmpl.Execute(w, data)
	if err != nil {
		log.Printf("Failed to execute template. %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	// Sort conditions by rating and name.
	mu.Lock()
	defer mu.Unlock()
	conds2 := make([]*Conditions, 0, len(conds))
	for _, c := range conds {
		conds2 = append(conds2, c)
	}
	sort.Sort(ByRating(conds2))

	// Render the results.
	data := struct {
		Conds []*Conditions
		Head  template.HTML
	}{Conds: conds2, Head: head}
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Failed to execute template. %v", err)
	}
}

func updateConditionsAllLocations() {
	client := &http.Client{Timeout: *timeout}

	// Gather locations.
	log.Printf("Fetching %s", *siteMapUrl)
	resp, err := client.Get(*siteMapUrl)
	if err != nil {
		log.Printf("Failed to get site map. %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body of site map. %v", err)
	}
	reportMatches := reportRx.FindAll(body, -1)
	for _, match := range reportMatches {
		path := string(match)
		name, err := surfReportPathToName(path)
		if err != nil {
			log.Printf("Failed to convert surf report path to name. %v", err)
			continue
		}
		_, ok := locations[name]
		if !ok {
			log.Printf("Got new location: %s: %s", name, path)
		}
		loc := &Location{
			Name:             name,
			MagicSeaweedPath: path,
		}
		locations[name] = loc
	}
	log.Printf("Found %d reports", len(reportMatches))
	log.Printf("New number of locations: %d", len(locations))

	// Gather conditions and errors.
	mu.Lock()
	errs = make([]*Error, 0, len(locations))
	mu.Unlock()
	for _, loc := range locations {
		cond, err := loc.GetConditions(client)
		if err != nil {
			mu.Lock()
			e := &Error{
				Loc: loc,
				Err: err,
			}
			errs = append(errs, e)
			mu.Unlock()
			continue
		}
		mu.Lock()
		conds[loc.Name] = cond
		mu.Unlock()
		saveConditionsFile()
	}
}

var srpTailRx = regexp.MustCompile(`-Surf-Report/\d+/`)

func surfReportPathToName(srp string) (string, error) {
	s := srpTailRx.ReplaceAllString(srp, "")
	s, err := url.PathUnescape(s)
	if err != nil {
		return "", fmt.Errorf("Failed to unescape path in surfReportPathToName(%q). %v", srp, err)
	}
	s = html.UnescapeString(s)
	s = s[1:] // Remove leading /
	s = strings.Replace(s, "-", " ", -1)
	return s, nil
}

func loadConditionsFile() {
	f, err := os.Open(*conditionsFilePath)
	if err != nil {
		log.Printf("Failed to open conditions file %s. %v. That's okay.", *conditionsFilePath, err)
		return
	}
	contents, err := ioutil.ReadAll(f)
	err = json.Unmarshal(contents, &conds)
	if err != nil {
		log.Printf("Failed to unmarshall the conditions map. %v", err)
		os.Exit(1)
	}
	log.Printf("Loaded conditions file %s", *conditionsFilePath)
}

func saveConditionsFile() {
	f, err := os.Create(*conditionsFilePath)
	if err != nil {
		log.Printf("Failed to open conditions file %s for writing. %v", *conditionsFilePath)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	contents, err := json.Marshal(conds)
	if err != nil {
		log.Printf("Failed to marshal conditions. %v", conds)
		return
	}
	f.Write(contents)
	f.Close()
	log.Printf("Saved conditions file %s", *conditionsFilePath)
}

var locations = make(map[string]*Location)
var mu sync.Mutex // for conds and errs
var conds = make(map[string]*Conditions)
var errs []*Error

type ByRating []*Conditions

func (r ByRating) Len() int      { return len(r) }
func (r ByRating) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r ByRating) Less(i, j int) bool {
	ci := r[i]
	cj := r[j]
	if ci.Rating == cj.Rating {
		return ci.Loc.Name < cj.Loc.Name
	}
	return ci.Rating > cj.Rating
}

var starSectionRx = regexp.MustCompile(`<ul class="rating rating-large clearfix">.*?</ul>`)
var starRx = regexp.MustCompile(`<li class="active"> *<i class="glyphicon glyphicon-star"></i> *</li>`)
var heightRx = regexp.MustCompile(`(\d+(?:-\d+)?)<small>ft`)
var reportRx = regexp.MustCompile(`/[^"/]+-Surf-Report/\d+/`)

func (loc *Location) GetConditions(client *http.Client) (*Conditions, error) {
	url := "http://magicseaweed.com" + loc.MagicSeaweedPath
	log.Printf("Fetching %s", url)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read body. %v", err)
	}
	rating := countTopStars(body)
	hMatch := heightRx.FindSubmatch(body)
	if len(hMatch) != 2 {
		return nil, fmt.Errorf("Wave height regex failed.")
	}
	details := fmt.Sprintf("%s ft", hMatch[1])
	cond := &Conditions{
		Loc:     loc,
		Rating:  rating,
		Details: details,
	}
	return cond, nil
}

// countTopStars returns the number of stars in the first rating section on the page.
func countTopStars(body []byte) int {
	starSection := starSectionRx.Find(body)
	foundStars := starRx.FindAll(starSection, -1)
	return len(foundStars)
}

func (c *Conditions) Stars() string {
	runes := make([]rune, 0, 5)
	for i := 0; i < c.Rating; i++ {
		runes = append(runes, '★')
	}
	for i := 0; i < 5-c.Rating; i++ {
		runes = append(runes, '☆')
	}
	return string(runes)
}

func keepConditionsUpdated() {
	updateConditionsAllLocations()
	tick := time.Tick(*conditionsInterval)
	for {
		<-tick
		updateConditionsAllLocations()
	}
}
