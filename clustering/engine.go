package clustering

import "github.com/fabricekabongo/loggerhead/query"

type EngineDecorator struct {
	cluster     *Cluster
	engine      *query.Engine
	commandChan chan string
}

func (e EngineDecorator) ExecuteQuery(query string) string {
	e.commandChan <- query
	return e.engine.ExecuteQuery(query)
}

func NewEngineDecorator(cluster *Cluster, engine *query.Engine) query.EngineInterface {
	eng := &EngineDecorator{
		cluster:     cluster,
		engine:      engine,
		commandChan: make(chan string),
	}

	go eng.commandLoop()

	return eng
}

func (e EngineDecorator) commandLoop() {

	for {
		select {
		case command := <-e.commandChan:
			e.cluster.Broadcasts().QueueBroadcast(NewLocationBroadcast(command)) // At least broadcast the query to the cluster in case we go down before executing it

		}
	}
}
