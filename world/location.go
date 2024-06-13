package world

import (
	"encoding/gob"
	"errors"
	"fmt"
)

var (
	LocationErrorRequiredId        = errors.New("location id is required")
	LocationErrorInvalidLatitude   = errors.New("invalid latitude")
	LocationErrorInvalidLongitude  = errors.New("invalid longitude")
	LocationErrorRequiredNamespace = errors.New("namespace is required")
)

func init() {
	gob.Register(Location{})
}

type Location struct {
	Id  string  `json:"id"`
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
	Ns  string  `json:"ns"`
}

func NewLocation(ns string, locId string, lat float64, lon float64) (*Location, error) {
	loc := &Location{
		Id:  locId,
		Lat: lat,
		Lon: lon,
		Ns:  ns,
	}

	if err := loc.validate(); err != nil {
		return nil, err
	}

	return loc, nil
}

func (l *Location) validate() error {
	if len(l.Id) == 0 {
		return LocationErrorRequiredId
	}

	if l.Lat < -90 || l.Lat > 90 {
		return LocationErrorInvalidLatitude
	}

	if l.Lon < -180 || l.Lon > 180 {
		return LocationErrorInvalidLongitude
	}

	if len(l.Ns) == 0 {
		return LocationErrorRequiredNamespace
	}

	return nil
}

func (l *Location) String() string {
	return fmt.Sprintf("%s,%s,%f,%f", l.Ns, l.Id, l.Lat, l.Lon)
}
