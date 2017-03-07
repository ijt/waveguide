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

var latlongFile = flag.String("file", "latlong.json", "JSON file storing the results")
var waveguideUrl = flag.String("waveguide", "http://104.198.243.193", "URL for waveguide")

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
	spotsCoords, err := orderByStarsOnWaveguide(spotsCoords)
	if err != nil {
		log.Printf("Failed to order by stars on Waveguide. %v", err)
	}
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
		open(sc.GuideUrl())
		searchForCoordinates(sc.Spot)
		showMapForName(sc.Spot.Name)

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
			yn, err := askf("Got %.4f, %.4f. OK (y/n/s)\n", coords.Lat, coords.Long)
			if err != nil {
				log.Fatal(err)
			}
			switch strings.ToLower(yn)[0] {
			case 's':
				fmt.Println("Skipping.\n")
				break coordLoop
			case 'n':
				fmt.Println("Trying again.\n")
				continue coordLoop
			default:
				sc.Coords = coords
				saveLatLongFile(spotsCoords)
				break coordLoop
			}
		}
	}
}

func orderByStarsOnWaveguide(spotsCoords []*SpotCoords) ([]*SpotCoords, error) {
	log.Printf("Ordering by stars on waveguide.")

	// Map from report path to SpotCoords
	rp2sc := make(map[string]*SpotCoords)
	for _, sc := range spotsCoords {
		rp2sc[sc.ReportPath] = sc
	}

	reportPaths, err := getReportPathsFromWaveguide()
	if err != nil {
		return spotsCoords, fmt.Errorf("Failed to get report paths from waveguide. %v", err)
	}

	// Make a new version of spot coords with the given order
	sc2 := make([]*SpotCoords, 0, len(spotsCoords))
	for _, rp := range reportPaths {
		sc2 = append(sc2, rp2sc[rp])
	}
	return sc2, nil
}

func getReportPathsFromWaveguide() ([]string, error) {
	resp, err := http.Get(*waveguideUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read body of %s. %v", *waveguideUrl, err)
	}
	body := string(bodyBytes)
	reportPaths := reportRx.FindAllString(body, -1)
	return reportPaths, nil
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

func (s Spot) GuideUrl() string {
	guidePath := strings.Replace(s.ReportPath, "Report", "Guide", -1)
	return fmt.Sprintf("http://magicseaweed.com/%s", guidePath)
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

func showMapForName(name string) {
	url := fmt.Sprintf("https://www.google.com/maps/place/%s", name)
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
