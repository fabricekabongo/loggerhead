package query

import (
	"errors"
	w "github.com/fabricekabongo/loggerhead/world"
	"log"
	"strconv"
	"strings"
)

var (
	ErrorInvalidQuery = errors.New("invalid query")
	version           = "1.0"
)

type EngineInterface interface {
	ExecuteQuery(query string) string
}

type Engine struct {
	world *w.World
	chain []Processor
}

type Processor interface {
	Execute(query string) string
	CanProcess(query string) bool
}

func NewQueryEngine(world *w.World) EngineInterface {
	return &Engine{
		world: world,
		chain: []Processor{
			&GetQueryProcessor{World: world},
			&DeleteQueryProcessor{World: world},
			&SaveQueryProcessor{World: world},
			&PolyQueryProcessor{World: world},
		},
	}
}

func NewReadQueryEngine(world *w.World) EngineInterface {
	return &Engine{
		world: world,
		chain: []Processor{
			&GetQueryProcessor{World: world},
			&PolyQueryProcessor{World: world},
		},
	}
}

func NewWriteQueryEngine(world *w.World) *Engine {
	return &Engine{
		world: world,
		chain: []Processor{
			&SaveQueryProcessor{World: world},
			&DeleteQueryProcessor{World: world},
		},
	}
}

func (qp *Engine) World() *w.World {
	return qp.world
}

func (qp *Engine) ExecuteQuery(query string) string {
	for _, processor := range qp.chain {
		if processor.CanProcess(query) {
			return processor.Execute(query)
		}
	}

	log.Println(ErrorInvalidQuery.Error(), query)

	return "1.0,\"" + ErrorInvalidQuery.Error() + "\"\n"
}

type GetQueryProcessor struct {
	World *w.World
	Processor
}

func (p *GetQueryProcessor) Execute(query string) string {
	if p.World == nil {
		panic("world is nil")
	}

	if !p.CanProcess(query) {
		panic("call CanProcess before calling me")
	}

	//GET NamespaceID LocationID
	chunks := strings.Split(query, " ")

	if chunks[0] != "GET" { //No trust
		panic("Invalid DELETE query")
	}

	namespaceID := chunks[1]
	locationID := chunks[2]

	location, ok := p.World.GetLocation(namespaceID, locationID)

	if !ok {
		return "1.0,done\n"
	}

	stringBuilder := strings.Builder{}
	stringBuilder.WriteString("1.0," + location.String() + "\n")
	stringBuilder.WriteString("1.0,done\n")

	return stringBuilder.String()
}

func (p *GetQueryProcessor) CanProcess(query string) bool {
	chunks := strings.Split(query, " ")
	if len(chunks) != 3 {
		return false
	}

	return chunks[0] == "GET"
}

type DeleteQueryProcessor struct {
	World *w.World
	Processor
}

func (p *DeleteQueryProcessor) Execute(query string) string {
	if p.World == nil {
		panic("world is nil")
	}

	if !p.CanProcess(query) {
		panic("call CanProcess before calling me")
	}

	//DELETE NamespaceID LocationID
	chunks := strings.Split(query, " ")

	if chunks[0] != "DELETE" { //No trust
		panic("Invalid DELETE query")
	}

	namespaceID := chunks[1]
	locationID := chunks[2]

	p.World.Delete(namespaceID, locationID)

	return "1.0,deleted\n"
}

func (p *DeleteQueryProcessor) CanProcess(query string) bool {
	chunks := strings.Split(query, " ")
	if len(chunks) != 3 {
		return false
	}

	return chunks[0] == "DELETE"
}

type SaveQueryProcessor struct {
	World *w.World
}

func (p *SaveQueryProcessor) Execute(query string) string {
	if p.World == nil {
		panic("world is nil")
	}

	if !p.CanProcess(query) {
		panic("call CanProcess before calling me")
	}

	//SAVE NamespaceID LocationID Latitude Longitude
	chunks := strings.Split(query, " ")

	if chunks[0] != "SAVE" { //No trust
		panic("Invalid SAVE query")
	}

	namespaceID := chunks[1]
	locationID := chunks[2]
	latitude := chunks[3]
	longitude := chunks[4]

	latFloat, err := strconv.ParseFloat(latitude, 64)
	if err != nil {
		return "1.0," + "\"Invalid float64 value for latitude\"\n"
	}

	lonFloat, err := strconv.ParseFloat(longitude, 64)
	if err != nil {
		return "1.0," + "\"Invalid float64 value for longitude\"\n"
	}

	err = p.World.Save(namespaceID, locationID, latFloat, lonFloat)
	if err != nil {
		return "1.0,\"" + err.Error() + "\"\n"
	}

	return "1.0,saved\n"
}

func (p *SaveQueryProcessor) CanProcess(query string) bool {
	chunks := strings.Split(query, " ")
	if len(chunks) != 5 {
		return false
	}

	return chunks[0] == "SAVE"
}

type PolyQueryProcessor struct {
	World *w.World
}

func (p *PolyQueryProcessor) Execute(query string) string {
	if p.World == nil {
		panic("world is nil")
	}

	if !p.CanProcess(query) {
		panic("call CanProcess before calling me")
	}

	//POLY Latitude1 Longitude1 Latitude2 Longitude2
	chunks := strings.Split(query, " ")

	if chunks[0] != "POLY" { //No trust
		panic("Invalid POLY query")
	}

	ns := chunks[1]
	lat1, err := strconv.ParseFloat(chunks[2], 64)
	if err != nil {
		return "1.0," + "\"Invalid float64 value for latitude1\"\n"
	}
	lon1, err := strconv.ParseFloat(chunks[3], 64)
	if err != nil {
		return "1.0," + "\"Invalid float64 value for longitude1\"\n"
	}
	lat2, err := strconv.ParseFloat(chunks[4], 64)
	if err != nil {
		return "1.0," + "\"Invalid float64 value for latitude2\"\n"
	}
	lon2, err := strconv.ParseFloat(chunks[5], 64)
	if err != nil {
		return "1.0," + "\"Invalid float64 value for longitude2\"\n"
	}

	locations := p.World.QueryRange(ns, lat1, lat2, lon1, lon2)

	var result strings.Builder

	for _, location := range locations {
		result.WriteString("1.0," + location.String() + "\n")
	}

	result.WriteString("1.0,done\n")

	return result.String()
}

func (p *PolyQueryProcessor) CanProcess(query string) bool {
	chunks := strings.Split(query, " ")
	if len(chunks) != 6 {
		return false
	}

	return chunks[0] == "POLY"
}
