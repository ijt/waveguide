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
				<td><a href="{{.Loc.MagicSeaweedUrl}}">{{.Loc.Name}}</a></td>
				<td>{{.Rating}}</td>
				<td>{{.Details}}</td>
			</tr>
			{{end}}
		</table>

		<table>
			{{range .Errs}}
			<tr>
				<td><a href="{{.Loc.MagicSeaweedUrl}}">{{.Loc.Name}}</a></td>
				<td>{{.Err}}</td>
			</tr>
			{{end}}
		</table>
	</body>
</html>
`))

type Location struct {
	Name            string
	MagicSeaweedUrl string
}

type Conditions struct {
	Loc     *Location
	Rating  string
	Details string
}

type Error struct {
	Loc Location
	Err error
}

var locations = []Location{
	Location{"Lindamar-Pacifica", "http://magicseaweed.com/Linda-Mar-Pacifica-Surf-Report/819/"},
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	// Spawn requests to get the conditions.
	ch := make(chan *Conditions)
	ech := make(chan *Error)
	var wg sync.WaitGroup
	for _, loc := range locations {
		loc := loc
		wg.Add(1)
		go func() {
			defer wg.Done()
			cond, err := loc.GetConditions()
			if err != nil {
				ech <- &Error{
					Loc: loc,
					Err: err,
				}
				return
			}
			ch <- cond
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
		close(ech)
	}()

	// Render the results.
	data := struct {
		Conds chan *Conditions
		Errs  chan *Error
	}{Conds: ch, Errs: ech}
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Failed to execute template. %v", err)
	}
}

var starRx = regexp.MustCompile(`<li class="active"> *<i class="glyphicon glyphicon-star"></i> *</li>`)
var heightRx = regexp.MustCompile(`(\d+-\d+)<small>ft`)

func (loc *Location) GetConditions() (*Conditions, error) {
	log.Printf("Fetching %s", loc.MagicSeaweedUrl)
	resp, err := http.Get(loc.MagicSeaweedUrl)
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
