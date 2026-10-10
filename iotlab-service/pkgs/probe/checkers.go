package probe

import (
	"context"

	"github.com/tdatIT/iotlab-service/pkgs/database"
	"github.com/tdatIT/iotlab-service/pkgs/rdclient"
)

// DBChecker returns a Checker that pings the relational database.
func DBChecker(db database.DB) CheckerFunc {
	return func(ctx context.Context) error {
		return db.SQLDB().PingContext(ctx)
	}
}

// RedisChecker returns a Checker that pings Redis.
func RedisChecker(rc rdclient.RedisClient) CheckerFunc {
	return func(ctx context.Context) error {
		return rc.Client().Ping(ctx).Err()
	}
}
