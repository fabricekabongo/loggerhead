package world

import (
	"bytes"
	"encoding/gob"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocationInitValidation(t *testing.T) {
	t.Run("should validate inputs", func(t *testing.T) {
		loc := &Location{}

		_, err := loc.init("ns", "", 0, 0)
		assert.ErrorIs(t, err, ErrLocationRequiredId)

		_, err = loc.init("", "id", 0, 0)
		assert.ErrorIs(t, err, ErrLocationRequiredNamespace)

		_, err = loc.init("ns", "id", 100, 0)
		assert.ErrorIs(t, err, ErrLocationInvalidLatitude)

		_, err = loc.init("ns", "id", 0, 200)
		assert.ErrorIs(t, err, ErrLocationInvalidLongitude)

		created, err := loc.init("ns", "id", 1, 1)
		assert.NoError(t, err)
		assert.Equal(t, "ns", created.Ns())
		assert.Equal(t, "id", created.Id())
	})
}

func TestLocationString(t *testing.T) {
	loc, err := NewLocation("ns", "id", 1.5, 2.5)
	assert.NoError(t, err)
	assert.Equal(t, "ns,id,1.500000,2.500000", loc.String())
}

func TestWorldSerialization(t *testing.T) {
	world := NewWorld()
	assert.NoError(t, world.Save("ns1", "loc1", 10, 20))
	assert.NoError(t, world.Save("ns2", "loc2", -10, -20))

	serialized := world.ToBytes()
	assert.NotEmpty(t, serialized)

	restored := NewWorldFromBytes(serialized)

	loc1, ok := restored.GetLocation("ns1", "loc1")
	assert.True(t, ok)
	assert.Equal(t, 10.0, loc1.Lat())
	assert.Equal(t, 20.0, loc1.Lon())

	loc2, ok := restored.GetLocation("ns2", "loc2")
	assert.True(t, ok)
	assert.Equal(t, -10.0, loc2.Lat())
	assert.Equal(t, -20.0, loc2.Lon())
}

func TestWorldFromBytesInvalidPayload(t *testing.T) {
	assert.Panics(t, func() {
		_ = NewWorldFromBytes([]byte("not-a-gob"))
	})
}

func TestWorldFromBytesSaveError(t *testing.T) {
	type serialLocation struct {
		ID  string
		Lat float64
		Lon float64
	}

	type serialNamespace struct {
		Name      string
		Locations []serialLocation
	}

	type serialWorld struct {
		Namespaces []serialNamespace
	}

	invalid := serialWorld{Namespaces: []serialNamespace{{Name: "ns", Locations: []serialLocation{{ID: "bad", Lat: 200, Lon: 0}}}}}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(invalid)
	assert.NoError(t, err)

	assert.Panics(t, func() {
		_ = NewWorldFromBytes(buf.Bytes())
	})
}

func TestWorldNamespacePanics(t *testing.T) {
	world := &World{namespaces: map[string]*Namespace{"panicNS": nil}, mu: sync.RWMutex{}}

	assert.PanicsWithValue(t, ErrUnexpectedNilNamespace, func() {
		world.Save("panicNS", "id", 0, 0)
	})

	assert.PanicsWithValue(t, ErrUnexpectedNilNamespace, func() {
		world.Delete("panicNS", "id")
	})

	assert.PanicsWithValue(t, ErrUnexpectedNilNamespace, func() {
		world.QueryRange("panicNS", 0, 1, 0, 1)
	})

	assert.PanicsWithValue(t, ErrUnexpectedNilNamespace, func() {
		_, _ = world.GetLocation("panicNS", "id")
	})
}

func TestWorldSaveValidationError(t *testing.T) {
	world := NewWorld()

	err := world.Save("ns", "id", 200, 0)
	assert.ErrorIs(t, err, ErrLocationInvalidLatitude)
}

func TestNamespaceUpdateError(t *testing.T) {
	ns := NewNamespace("ns")
	_, err := ns.SaveLocation("id", 1, 1)
	assert.NoError(t, err)

	_, err = ns.SaveLocation("id", 200, 1)
	assert.ErrorIs(t, err, ErrLocationInvalidLatitude)
}

func TestNamespaceInsertOutOfBounds(t *testing.T) {
	ns := &Namespace{Name: "ns", locations: map[string]*Location{}, tree: NewQuadTree(0, 1, 0, 1)}

	_, err := ns.SaveLocation("id", 2, 2)
	assert.ErrorIs(t, err, ErrTreeLocationOutOfBounds)
}

func TestMergeErrorPath(t *testing.T) {
	world1 := NewWorld()
	badWorld := NewWorld()

	// Inject an invalid location directly to bypass validation and force a merge failure.
	badWorld.namespaces["ns"] = &Namespace{locations: map[string]*Location{"bad": {id: "bad", lat: 200, lon: 0, ns: "ns"}}, tree: NewQuadTree(-90, 90, -180, 180)}

	assert.Panics(t, func() {
		world1.Merge(badWorld)
	})
}

func TestTreeInsertOutOfBounds(t *testing.T) {
	node := NewTreeNode(0, 1, 0, 1, 1)
	loc, err := NewLocation("ns", "id", 2, 2)
	assert.NoError(t, err)

	assert.ErrorIs(t, node.insert(loc), ErrTreeLocationOutOfBounds)
}

func TestTreeDivideNoOpWhenAlreadyDivided(t *testing.T) {
	node := NewTreeNode(0, 1, 0, 1, 1)
	node.IsDivided = true

	node.divide()

	assert.True(t, node.IsDivided)
}

func TestTreeQueryRangeNoOverlap(t *testing.T) {
	node := NewTreeNode(0, 1, 0, 1, 1)
	results := node.QueryRange(2, 3, 2, 3)

	assert.Empty(t, results)
}

func TestTreeDivideFallbacks(t *testing.T) {
	node := NewTreeNode(0, 10, 0, 10, 1)

	locNW, err := NewLocation("ns", "nw", 1, 1)
	assert.NoError(t, err)
	assert.NoError(t, node.insert(locNW))

	locSE, err := NewLocation("ns", "se", 1, 9)
	assert.NoError(t, err)
	assert.NoError(t, node.insert(locSE))

	assert.True(t, node.IsDivided)
	assert.Equal(t, node.SE, locSE.Node)
}

func TestTreeInsertRelocatesExistingNode(t *testing.T) {
	tree := NewQuadTree(-90, 90, -180, 180)

	loc, err := NewLocation("ns", "id", -80, -170)
	assert.NoError(t, err)
	assert.NoError(t, tree.Insert(loc))

	err = loc.Update(80, 170)
	assert.NoError(t, err)

	assert.NoError(t, tree.Insert(loc))
	assert.NotNil(t, loc.Node)
	assert.True(t, loc.Node.Lat1 >= 0)
}

func TestTreeDivideWithNilLocationPanics(t *testing.T) {
	node := NewTreeNode(0, 10, 0, 10, 1)
	node.Objects["nil"] = nil

	assert.Panics(t, func() {
		node.divide()
	})
}

func TestTreeInsertRemovesFromPreviousNode(t *testing.T) {
	node := NewTreeNode(0, 1, 0, 1, 2)
	previous := NewTreeNode(0, 1, 0, 1, 2)

	loc, err := NewLocation("ns", "reassign", 0.5, 0.5)
	assert.NoError(t, err)

	loc.SetNode(previous)
	previous.Objects[loc.Id()] = loc

	assert.NoError(t, node.insert(loc))
	_, exists := previous.Objects[loc.Id()]
	assert.False(t, exists)
}

func TestTreeInsertNilLocation(t *testing.T) {
	node := NewTreeNode(0, 1, 0, 1, 1)

	assert.ErrorIs(t, node.insert(nil), ErrTreeLocationNil)
}
