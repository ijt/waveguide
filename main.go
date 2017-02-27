package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"
)

var addr = flag.String("a", ":4089", "server address")

func main() {
	http.HandleFunc("/", handleRoot)
	log.Printf("Listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var tmpl = template.Must(template.New("main").Parse(`
<html>
	<head>
		<title>Wave Hunter</title>
	</head>
	<body>
		<table>
			<thead>
				<th>Name</th>
				<th>Rating</th>
				<th>Details</th>
			</thead>
			{{range .Conds}}
			<tr>
				<td><a href="http://magicseaweed.com{{.Loc.MagicSeaweedPath}}">{{.Loc.Name}}</a></td>
				<td>{{.Rating}}</td>
				<td>{{.Details}}</td>
			</tr>
			{{end}}
		</table>

		<table>
			{{range .Errs}}
			<tr>
				<td><a href="http://magicseaweed.com{{.Loc.MagicSeaweedPath}}">{{.Loc.Name}}</a></td>
				<td>{{.Err}}</td>
			</tr>
			{{end}}
		</table>
	</body>
</html>
`))

type Location struct {
	Name             string
	MagicSeaweedPath string
}

type Conditions struct {
	Loc     *Location
	Rating  string
	Details string
}

type Error struct {
	Loc *Location
	Err error
}

type ConditionsOrError struct {
	Cond *Conditions
	Err  *Error
}

var locations = []Location{
	Location{"Lindamar-Pacifica", "/Linda-Mar-Pacifica-Surf-Report/819/"},
	Location{"Ocean Beach SF", "/Ocean-Beach-Surf-Report/255/"},
	Location{"Bali: Kuta Beach", "/Kuta-Beach-Surf-Report/566/"},
	Location{"Bolinas", "/Bolinas-Surf-Report/4221/"},
	Location{"Bolinas Jetty", "/Bolinas-Jetty-Surf-Report/4215/"},
	// Cairns
	// Cuba
	// Kauai
	// Maui (break this down)
	// Mexico
	// New Zealand
	// Oahu
	// Panama
	// Princeton Jetty
	// San Diego
	// Stinson Beach
	// Sydney
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	// Spawn requests to get the conditions.
	ch := make(chan *ConditionsOrError)
	var wg sync.WaitGroup
	for _, loc := range locations {
		loc := loc
		wg.Add(1)
		go func() {
			defer func() {
				log.Printf("Calling wg.Done()")
				wg.Done()
			}()
			cond, err := loc.GetConditions()
			coe := &ConditionsOrError{}
			if err != nil {
				log.Printf("In goroutine, got an error: %v", err)
				coe.Err = &Error{
					Loc: &loc,
					Err: err,
				}
			} else {
				coe.Cond = cond
			}
			ch <- coe
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	// Gather conditions and errors.
	conds := make([]*Conditions, 0, len(locations))
	errs := make([]*Error, 0, len(locations))
	for coe := range ch {
		if coe.Err != nil {
			errs = append(errs, coe.Err)
			continue
		}
		conds = append(conds, coe.Cond)
	}

	// Render the results.
	data := struct {
		Conds []*Conditions
		Errs  []*Error
	}{Conds: conds, Errs: errs}
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Failed to execute template. %v", err)
	}
}

var starRx = regexp.MustCompile(`<li class="active"> *<i class="glyphicon glyphicon-star"></i> *</li>`)
var heightRx = regexp.MustCompile(`(\d+-\d+)<small>ft`)

func (loc *Location) GetConditions() (*Conditions, error) {
	url := "http://magicseaweed.com" + loc.MagicSeaweedPath
	log.Printf("Fetching %s", url)
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read body. %v", err)
	}
	stars := starRx.FindAll(body, -1)
	hMatch := heightRx.FindSubmatch(body)
	if len(hMatch) != 2 {
		return nil, fmt.Errorf("Wave height regex failed.")
	}
	rating := fmt.Sprintf("%d/5 stars", len(stars))
	details := fmt.Sprintf("%s ft", hMatch[1])
	cond := &Conditions{
		Loc:     loc,
		Rating:  rating,
		Details: details,
	}
	log.Printf("Succeeded, with conditions %+v", cond)
	return cond, nil
}
