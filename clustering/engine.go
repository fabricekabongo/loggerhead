package clustering

import "github.com/fabricekabongo/loggerhead/query"

type EngineDecorator struct {
	cluster *Cluster
	engine  *query.Engine
}

func (e EngineDecorator) ExecuteQuery(query string) string {
	go e.cluster.Broadcasts().QueueBroadcast(NewLocationBroadcast(query)) // At least broadcast the query to the cluster in case we go down before executing it

	return e.engine.ExecuteQuery(query)
}

func NewEngineDecorator(cluster *Cluster, engine *query.Engine) query.EngineInterface {
	return &EngineDecorator{
		cluster: cluster,
		engine:  engine,
	}
}
