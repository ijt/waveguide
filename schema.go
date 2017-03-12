package waveguide

import (
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
)

type Spot struct {
	Name 		 string
	// TODO: Change to using a general url to be less coupled
	// to a particular site.
	MswPath		 string
	// Surf conditions at this spot, most recent last
	Qual		 []Quality
}

type Quality struct {
	// How many stars out of five
	Rating int
	WaveHeight string
	TimeUnix int64
}

func SpotKey(ctx context.Context, mswPath string) *datastore.Key {
	return datastore.NewKey(ctx, "Spot", mswPath, 0, nil)
}

func AddQualityToSpot(ctx context.Context, key *datastore.Key, qual *Quality) error {
	var s Spot
	err := datastore.Get(ctx, key, &s)
	if err != nil {
		return err
	}
	s.Qual = append(s.Qual, *qual)
	_, err = datastore.Put(ctx, key, &s)
	if err != nil {
		return err
	}
	return nil
}