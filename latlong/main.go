package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var latlongFile = flag.String("-file", "latlong.json", "JSON file storing the results")

type Spot struct {
	Name       string
	ReportPath string // on magicseaweed
}

type SpotCoords struct {
	Spot
	Coords *Coordinates
}

func main() {
	spotsCoords := loadLatLongFile()
	spotsCoords = addSpotsFromMswSiteMap(spotsCoords)
	saveLatLongFile(spotsCoords)

	// For each Spot without coordinates
	defer saveLatLongFile(spotsCoords)
	for _, sc := range spotsCoords {
		if sc.Coords != nil {
			fmt.Printf("Already got coordinates for %s: %v\n", sc.Name, sc.Coords)
			continue
		}

		// Show the user info to help get the coordinates.
		open(sc.Url())
		searchForCoordinates(sc.Spot)

		for {
			coordsStr, err := askf("\nEnter coordinates for %s, or empty to skip. For details see %s\n", sc.Name, sc.Url())
			if len(strings.TrimSpace(coordsStr)) == 0 {
				fmt.Println("Skipping.")
				continue
			}
			if err != nil {
				fmt.Println(err)
				continue
			}
			coords, err := parseCoords(coordsStr)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Printf("Got %.4f, %.4f\n", coords.Lat, coords.Long)
			sc.Coords = coords
			saveLatLongFile(spotsCoords)
			break
		}
	}
}

// askf asks the user to input something in the terminal.
func askf(promptFmt string, args ...interface{}) (string, error) {
	fmt.Printf(promptFmt, args...)
	r := bufio.NewReader(os.Stdin)
	return r.ReadString('\n')
}

func (s Spot) Url() string {
	return fmt.Sprintf("http://magicseaweed.com/%s", s.ReportPath)
}

func searchForCoordinates(s Spot) {
	// TODO: Bring in more context from the msw surf report page.
	// For example, it could be the Santa Cruz in Portugal.
	url := fmt.Sprintf("https://www.google.com/search?q=%s coordinates", s.Name)
	open(url)
}

func loadLatLongFile() (spotsCoords []*SpotCoords) {
	contents, err := ioutil.ReadFile(*latlongFile)
	if err != nil {
		fmt.Printf("Didn't find %s. That's okay, starting fresh.\n", *latlongFile)
		return
	}
	err = json.Unmarshal(contents, &spotsCoords)
	if err != nil {
		log.Fatalf("Failed to parse %s. %v", *latlongFile, err)
	}
	return
}

func saveLatLongFile(spotsCoords []*SpotCoords) {
	contents, err := json.Marshal(&spotsCoords)
	if err != nil {
		log.Fatalf("Failed to encode data to json. %v", err)
	}
	err = ioutil.WriteFile(*latlongFile, contents, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Wrote", *latlongFile)
}

func addSpotsFromMswSiteMap(spotsCoords []*SpotCoords) []*SpotCoords {
	// Grab the beach names and urls from the msw sitemap.
	resp, err := http.Get("http://magicseaweed.com/site-map.php")
	if err != nil {
		log.Fatalf("Failed to get site map. %v", err)
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read body of response for site map. %v", err)
	}
	srPaths := reportRx.FindAll(contents, -1)
	fmt.Printf("Got %d paths from site map.\n", len(srPaths))
	for _, path := range srPaths {
		sPath := string(path)
		name, err := surfReportPathToName(sPath)
		if err != nil {
			log.Println(err)
			continue
		}

		found := false
		for _, sc := range spotsCoords {
			if sc.ReportPath == sPath {
				found = true
			}
		}

		if !found {
			sc := &SpotCoords{
				Spot: Spot{
					Name:       name,
					ReportPath: string(path),
				},
				Coords: nil,
			}
			spotsCoords = append(spotsCoords, sc)
		}
	}
	return spotsCoords
}

var reportRx = regexp.MustCompile(`/[^"/]+-Surf-Report/\d+/`)
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

func showMap(lat float32, long float32, zoom int) {
	url := fmt.Sprintf("https://www.google.com/maps/@%f,%f,%dz", lat, long, zoom)
	open(url)
}

// open opens a url in a browser. This works on Mac, but probably not elsewhere.
func open(url string) {
	cmd := exec.Command("open", url)
	go func() {
		err := cmd.Run()
		if err != nil {
			log.Printf("Failed to open %s. %v", url, err)
		}
	}()
}
