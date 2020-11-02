package config

import "github.com/go-redis/redis/v8"

var Redis *redis.Client

func InitRedis() {
	opt, err := redis.ParseURL("redis://localhost:6379/0")
	if err != nil {
		panic(err)
	}

	Redis = redis.NewClient(opt)
}
