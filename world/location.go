package world

import (
	"encoding/gob"
	"errors"
	"fmt"
	"time"
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
	id        string
	lat       float64
	lon       float64
	ns        string
	updatedAt time.Time
	hash      string
}

func NewLocation(ns string, id string, lat float64, lon float64) (*Location, error) {
	if len(id) == 0 {
		return nil, LocationErrorRequiredId
	}
	if len(ns) == 0 {
		return nil, LocationErrorRequiredNamespace
	}

	if err := validateLatLon(lat, lon); err != nil {
		return nil, err
	}

	loc := &Location{
		id:        id,
		lat:       lat,
		lon:       lon,
		ns:        ns,
		updatedAt: time.Now(),
	}

	loc.updateHash()

	return loc, nil
}

func (l *Location) updateHash() {
	l.hash = fmt.Sprintf("%s,%s,%f,%f", l.ns, l.id, l.lat, l.lon)
}

func (l *Location) Update(lat float64, lon float64) error {
	err := validateLatLon(lat, lon)

	if err != nil {
		return err
	}

	l.lat = lat
	l.lon = lon
	l.updatedAt = time.Now()
	l.updateHash()

	return nil
}

func validateLatLon(lat float64, lon float64) error {

	if lat < -90 || lat > 90 {
		return LocationErrorInvalidLatitude
	}

	if lon < -180 || lon > 180 {
		return LocationErrorInvalidLongitude
	}

	return nil
}

func (l *Location) String() string {
	return fmt.Sprintf("%s,%s,%f,%f", l.ns, l.id, l.lat, l.lon)
}

func (l *Location) Id() string {
	return l.id
}

func (l *Location) Lat() float64 {
	return l.lat
}

func (l *Location) Lon() float64 {
	return l.lon
}

func (l *Location) Ns() string {
	return l.ns
}

func (l *Location) UpdatedAt() time.Time {
	return l.updatedAt
}

func (l *Location) Hash() string {
	return l.hash
}
