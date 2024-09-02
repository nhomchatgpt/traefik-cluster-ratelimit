package traefik_cluster_ratelimit

import (
	"context"
	"net/http"

	//	redisrate "github.com/go-redis/redis_rate/v10"
	redis "github.com/go-redis/redis/v8"
)

// Config the plugin configuration.
type Config struct {
	RedisAddress string `json:"redisaddress"`
	RedisDB      uint   `json:"redisdb"`
	Average      uint   `json:"average"`
	Burst        uint   `json:"burst"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// Demo a Demo plugin.
type ClusterRateLimit struct {
	next   http.Handler
	client *redis.Client
	name   string
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// _, err := client.Ping(context.Background()).Result()
	// if err != nil {
	// 	log.Fatal("Error connecting to Redis:", err)
	// }

	return &ClusterRateLimit{
		next:   next,
		client: client,
		name:   name,
	}, nil
}

func (rl *ClusterRateLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// ctx := context.Background()
	// rr := redisrate.NewLimiter(rl.client)
	// res, _ := rr.Allow(ctx, fmt.Sprintf("namespace_", "userName"), redisrate.Limit{
	// 	Rate:   10,
	// 	Burst:  10,
	// 	Period: time.Second,
	// })
	// if res.Allowed <= 0 {
	// 	return
	// }

	rl.next.ServeHTTP(rw, req)
}
