package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
)

func main() {
	// Hit a bunch of urls to find out which surf spots have good conditions today.
	var errs []error
	for _, loc := range locations {
		cond, err := loc.GetSurflineConditions()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		fmt.Printf("%s: %s, %s, %s\n", loc.name, cond.quality, cond.height, loc.surflineUrl)
	}
	if len(errs) > 0 {
		fmt.Printf("Some errors occurred:\n")
		for _, err := range errs {
			fmt.Println(err)
		}
	}
}

type Location struct {
	name        string
	surflineUrl string
}

type Conditions struct {
	quality []byte // for example "poor", "fair", "good"
	height  []byte // for example "3-4 ft+"
}

var locations = []Location{
	Location{"Pacifica", "http://www.surfline.com/surf-report/pacifica-lindamar-northern-california_5013/"},
	Location{"Ocean Beach SF", "http://www.surfline.com/surf-report/ocean-beach-sf-northern-california_4127/"},
	Location{"Malibu", "http://www.surfline.com/surf-report/malibu-southern-california_4209/"},
	Location{"Pismo Beach", "http://www.surfline.com/surf-report/pismo-beach-pier-central-california_5065/"},
	Location{"Pipeline Oahu", "http://www.surfline.com/surf-report/pipeline-oahu_4750/"},
	Location{"Sunset Oahu", "http://www.surfline.com/surf-report/sunset-oahu_4746/"},
	// half moon bay
	// stinson beach
	// maui
	// sydney
	// cairns
	// bali
	// so-cal
	// cuba
	// mexico
	// panama
	// spain
}

var descRx = regexp.MustCompile(`>(\w+ Conditions)<`)
var heightRx = regexp.MustCompile(`\d+-\d+ ft( \+)?`)

// GetSurflineConditions fetches the surfline page for the given location and returns the subjective evaluation
// of the conditions there.
func (loc *Location) GetSurflineConditions() (*Conditions, error) {
	resp, err := http.Get(loc.surflineUrl)
	if err != nil {
		return nil, fmt.Errorf("GetSurflineConditions for %s failed to GET url. %v", loc.name, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GetSurflineConditions for %s failed to read body. %v", loc.name, err)
	}
	descMatch := descRx.FindSubmatch(body)
	quality := []byte("missing")
	if len(descMatch) == 2 {
		quality = descMatch[1]
	}
	cond := &Conditions{
		quality: quality,
		height:  heightRx.Find(body),
	}
	return cond, nil
}
