package clustering

import (
	"context"

	"github.com/fabricekabongo/loggerhead/query"
)

type EngineDecorator struct {
	cluster     *Cluster
	engine      *query.Engine
	commandChan chan string
	ctx         context.Context
}

func (e EngineDecorator) ExecuteQuery(query string) string {
	defer func() {
		e.commandChan <- query
	}() //Here I prioritize memory instead of the cluster broadcast as memory is faster than the network
	return e.engine.ExecuteQuery(query)
}

func NewEngineDecorator(ctx context.Context, cluster *Cluster, engine *query.Engine) query.EngineInterface {
	eng := &EngineDecorator{
		cluster:     cluster,
		engine:      engine,
		commandChan: make(chan string),
		ctx:         ctx,
	}

	go eng.commandLoop()

	return eng
}

func (e EngineDecorator) commandLoop() {
	for {
		select {
		case <-e.ctx.Done():
			return
		case command := <-e.commandChan:
			e.cluster.Broadcasts().QueueBroadcast(NewLocationBroadcast(command))
		}
	}
}
