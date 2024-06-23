package world

import (
	"errors"
	"github.com/hashicorp/go-uuid"
	"log"
	"math/rand/v2"
	"testing"
)

func CreateRandomTestLocation() (*Location, error, string, string, float64, float64) {
	id, err := uuid.GenerateUUID()
	if err != nil {
		log.Fatal(err)
	}
	ns, err := uuid.GenerateUUID()
	if err != nil {
		log.Fatal(err)
	}
	lat := rand.Float64()*180 - 90
	lon := rand.Float64()*360 - 180

	loc, err := NewLocation(ns, id, lat, lon)

	return loc, err, ns, id, lat, lon
}

func TestLocation(t *testing.T) {
	t.Parallel()

	t.Run("NewLocation", func(t *testing.T) {
		t.Parallel()
		t.Run("should return an error if the namespace id is missing", func(t *testing.T) {
			t.Parallel()
			loc, err := NewLocation("", "Hey", 0, 0)

			if !errors.Is(err, LocationErrorRequiredNamespace) {
				t.Errorf("expected error %v, got %v", LocationErrorRequiredId, err)
			}

			if loc != nil {
				t.Errorf("expected location to be nil")
			}
		})
		t.Run("should return an error if location id is missing", func(t *testing.T) {
			t.Parallel()
			loc, err := NewLocation("theNamespace", "", 0, 0)

			if !errors.Is(err, LocationErrorRequiredId) {
				t.Errorf("expected error %v, got %v", LocationErrorRequiredId, err)
			}

			if loc != nil {
				t.Errorf("expected location to be nil")
			}
		})

		t.Run("should return an error if latitude is invalid", func(t *testing.T) {
			t.Parallel()
			loc, err := NewLocation("1", "id", -100, 0)

			if !errors.Is(err, LocationErrorInvalidLatitude) {
				t.Errorf("expected error %v, got %v", LocationErrorInvalidLatitude, err)
			}

			if loc != nil {
				t.Errorf("expected location to be nil")
			}
		})

		t.Run("should return an error if longitude is invalid", func(t *testing.T) {
			t.Parallel()
			loc, err := NewLocation("1", "id", 0, 200)

			if !errors.Is(err, LocationErrorInvalidLongitude) {
				t.Errorf("expected error %v, got %v", LocationErrorInvalidLongitude, err)
			}

			if loc != nil {
				t.Errorf("expected location to be nil")
			}
		})

		t.Run("should return a location entity", func(t *testing.T) {
			t.Parallel()
			loc, err, _, id, lat, lon := CreateRandomTestLocation()

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if loc.Id() != id {
				t.Errorf("expected loc id to be %s, got %s", id, loc.Id())
			}

			if loc.Lat() != lat {
				t.Errorf("expected lat to be %f, got %f", lat, loc.Lat())
			}

			if loc.Lon() != lon {
				t.Errorf("expected lon to be %f, got %f", lon, loc.Lon())
			}
		})
	})
}
