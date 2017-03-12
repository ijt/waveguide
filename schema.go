package waveguide

import (
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
)

type Spot struct {
	Name 		 string
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
