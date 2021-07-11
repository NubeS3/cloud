package arango

import (
	"context"
	"github.com/arangodb/go-driver"
	"time"
)

func ArangoClusterInfo() (driver.Cluster, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	return arangoClient.Cluster(ctx)
}

func ArangoInfo() (driver.ServerStatistics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*CONTEXT_EXPIRED_TIME)
	defer cancel()

	return arangoClient.Statistics(ctx)
}
