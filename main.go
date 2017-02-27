package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
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

		{{if .Errs}}
		<br>
		<b>Errors</b>:
		<table>
			{{range .Errs}}
			<tr>
				<td><a href="http://magicseaweed.com{{.Loc.MagicSeaweedPath}}">{{.Loc.Name}}</a></td>
				<td>{{.Err}}</td>
			</tr>
			{{end}}
		</table>
		{{end}}
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
	Location{"Bay Area: Lindamar-Pacifica", "/Linda-Mar-Pacifica-Surf-Report/819/"},
	Location{"Bay Area: Stinson Beach", "/Stinson-Beach-Surf-Report/4216/"},
	Location{"Bay Area: Ocean Beach SF", "/Ocean-Beach-Surf-Report/255/"},
	Location{"Bay Area: Princeton Jetty", "/Princeton-Jetty-Surf-Report/3679/"},
	Location{"Bali: Kuta Beach", "/Kuta-Beach-Surf-Report/566/"},
	Location{"Bolinas", "/Bolinas-Surf-Report/4221/"},
	Location{"Bolinas Jetty", "/Bolinas-Jetty-Surf-Report/4215/"},
	Location{"Cairns: Sunshine Beach", "/Sunshine-Beach-Surf-Report/1004/"},
	Location{"Oahu: Waikiki Beach", "/Queens-Canoes-Waikiki-Surf-Report/662/"},
	Location{"Kauai: Hanalei Bay", "/Hanalei-Bay-Surf-Report/3051/"},
	Location{"Kauai: Polihale", "/Polihale-Surf-Report/3080/"},
	Location{"Maui: Lahaina", "/Lahaina-Harbor-Breakwall-Surf-Report/4287/"},
	Location{"Oahu: Laniakea", "/Laniakea-Surf-Report/3672/"},
	Location{"Oahu: Pipeline", "/Pipeline-Backdoor-Surf-Report/616/"},
	Location{"Oahu: Sunset", "/Sunset-Surf-Report/657/"},
	Location{"Sydney: Manly Beach", "/Sydney-Manly-Surf-Report/526/"},
	Location{"Sydney: Bodi Beach", "/Sydney-Bondi-Surf-Report/996/"},
	Location{"New Zealand: Dunedin: Martins Bay", "/Martins-Bay-Surf-Report/3913/"},
	Location{"New Zealand: Dunedin: Papatowai", "/Papatowai-Surf-Report/124/"},
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	// Spawn requests to get the conditions.
	ch := make(chan *ConditionsOrError)
	var wg sync.WaitGroup
	for _, loc := range locations {
		loc := loc
		wg.Add(1)
		go func() {
			defer wg.Done()
			cond, err := loc.GetConditions()
			coe := &ConditionsOrError{}
			if err != nil {
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

	// Sort locations by rating and name.
	sort.Slice(conds, func(i, j int) bool {
		ci := conds[i]
		cj := conds[j]
		if ci.Rating == cj.Rating {
			return ci.Loc.Name < cj.Loc.Name
		}
		return ci.Rating > cj.Rating
	})

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
var heightRx = regexp.MustCompile(`(\d+(?:-\d+)?)<small>ft`)

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
	return cond, nil
}
