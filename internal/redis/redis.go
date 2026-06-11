package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

func New(url string) (*goredis.Client, error) {
	opts, err := goredis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := goredis.NewClient(opts)

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
