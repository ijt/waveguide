package main

import (
	"regexp"
	"fmt"
	"strconv"
)

type Coordinates struct {
	Lat  float64
	Long float64
}

// https://en.wikipedia.org/wiki/Geographic_coordinate_conversion
var decimalDegreesRx = regexp.MustCompile(`(\d+)\.(\d+)°\s*([NS]),?\s*(\d+)\.(\d+)°\s*([EW])`)
var decimalDegreesNoDirectionsRx = regexp.MustCompile(`(-?\d+)\.(\d+),\s*(-?\d+)\.(\d+)`)

func parseCoords(s string) (*Coordinates, error) {
	sms := decimalDegreesRx.FindStringSubmatch(s)
	if len(sms) == 1+6 {
		latStr := fmt.Sprintf("%s.%s", sms[1], sms[2])
		lat, err := strconv.ParseFloat(latStr, 32)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse latitude float %q found in %q. %v", latStr, s, err)
		}
		if sms[3] == "S" {
			lat *= -1
		}

		longStr := fmt.Sprintf("%s.%s", sms[4], sms[5])
		long, err := strconv.ParseFloat(longStr, 32)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse longitude float %q found in %q. %v", longStr, s, err)
		}
		if sms[6] == "W" {
			long *= -1
		}
		return &Coordinates{Lat: lat, Long: long}, nil
	}

	sms = decimalDegreesNoDirectionsRx.FindStringSubmatch(s)
	if len(sms) == 1+4 {
		latStr := fmt.Sprintf("%s.%s", sms[1], sms[2])
		lat, err := strconv.ParseFloat(latStr, 32)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse latitude float %q found in %q. %v", latStr, s, err)
		}

		longStr := fmt.Sprintf("%s.%s", sms[3], sms[4])
		long, err := strconv.ParseFloat(longStr, 32)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse longitude float %q found in %q. %v", longStr, s, err)
		}
		return &Coordinates{Lat: lat, Long: long}, nil
	}

	return nil, fmt.Errorf("Failed to match coordinate string %q", s)
}

func (c Coordinates)String() string {
	return fmt.Sprintf("(%.4f, %.4f)", c.Lat, c.Long)
}