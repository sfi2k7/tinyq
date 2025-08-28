package v2

import (
	"github.com/go-redis/redis/v8"
)

var Red *redis.Client

func init() {
	Red = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "passme", // no password set
		DB:       0,        // use default DB
	})
}
