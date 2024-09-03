package traefik_cluster_ratelimit

import (
	"context"
	"fmt"
	"net/http"

	redis "github.com/nzin/traefik-cluster-ratelimit/redis"
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
	client redis.Client
	name   string
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	client, err := redis.NewClient(
		config.RedisAddress,
		0,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create redis client: %v", err)
	}

	err = client.Ping()
	if err != nil {
		return nil, fmt.Errorf("error connecting to Redis: %v", err)
	}

	return &ClusterRateLimit{
		next:   next,
		client: client,
		name:   name,
	}, nil
}

func (rl *ClusterRateLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// ctx := context.Background()
	// rr := NewLimiter(rl.client)
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
