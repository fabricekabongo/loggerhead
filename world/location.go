package world

import (
	"encoding/gob"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	LocationErrorRequiredId        = errors.New("location id is required")
	LocationErrorInvalidLatitude   = errors.New("invalid latitude")
	LocationErrorInvalidLongitude  = errors.New("invalid longitude")
	LocationErrorRequiredNamespace = errors.New("namespace is required")
	validationOps                  = promauto.NewCounter(prometheus.CounterOpts{
		Name: "loggerhead_world_location_error",
		Help: "Total failed location data validation",
	})
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
	Node      *TreeNode
}

func NewLocation(ns string, id string, lat float64, lon float64) (*Location, error) {
	if id == "" {
		validationOps.Inc()
		return nil, LocationErrorRequiredId
	}
	if ns == "" {
		validationOps.Inc()
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

	return loc, nil
}

func (*Location) init(ns string, id string, lat float64, lon float64) (*Location, error) {
	if id == "" {
		validationOps.Inc()
		return nil, LocationErrorRequiredId
	}
	if ns == "" {
		validationOps.Inc()
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

	return loc, nil
}

func (l *Location) Update(lat float64, lon float64) error {
	err := validateLatLon(lat, lon)

	if err != nil {
		return err
	}

	l.lat = lat
	l.lon = lon
	l.updatedAt = time.Now()

	return nil
}

func validateLatLon(lat float64, lon float64) error {

	if lat < -90 || lat > 90 {
		validationOps.Inc()
		return LocationErrorInvalidLatitude
	}

	if lon < -180 || lon > 180 {
		validationOps.Inc()
		return LocationErrorInvalidLongitude
	}

	return nil
}

func (l *Location) SetNode(node *TreeNode) {
	l.Node.mu.Lock()
	l.Node = node
	l.Node.mu.Unlock()
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
