package database

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

//New connects to a Redis database.
func New(host, port string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:       fmt.Sprintf("%v:%v", host, port),
		MaxConnAge: 5 * time.Minute,
	})

	status := client.Ping(context.Background())
	if status.Err() != nil {
		return nil, status.Err()
	}

	return client, nil
}
