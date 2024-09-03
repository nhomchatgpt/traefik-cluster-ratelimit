package traefik_cluster_ratelimit

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	redis "github.com/nzin/traefik-cluster-ratelimit/redis"
)

// Config the plugin configuration.
type Config struct {
	RedisAddress  string `json:"redisaddress"`
	RedisDB       uint   `json:"redisdb"`
	RedisPassword string `json:"redispassword"`
	Average       uint   `json:"average"`
	Burst         uint   `json:"burst"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// Demo a Demo plugin.
type ClusterRateLimit struct {
	next    http.Handler
	limiter *Limiter
	name    string
	Average uint
	Burst   uint
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.RedisAddress == "" {
		config.RedisAddress = "redis:6379"
	}
	client, err := redis.NewClient(
		config.RedisAddress,
		config.RedisDB,
		config.RedisPassword,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create redis client: %v", err)
	}

	// err = client.Ping()
	// if err != nil {
	// 	return nil, fmt.Errorf("error connecting to Redis: %v", err)
	// }

	return &ClusterRateLimit{
		next:    next,
		limiter: NewLimiter(client, name),
		name:    name,
		Average: config.Average,
		Burst:   config.Burst,
	}, nil
}

func (rl *ClusterRateLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// cf https://medium.com/@bingolbalihasan/redis-rate-limiting-in-go-d342bab3d930

	res, err := rl.limiter.Allow(extractHostname(req), Limit{
		Rate:   int(rl.Average),
		Burst:  int(rl.Burst),
		Period: time.Second,
	})
	if err != nil {
		rl.next.ServeHTTP(rw, req)
	} else {
		if res.Allowed <= 0 {
			http.Error(rw, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		rl.next.ServeHTTP(rw, req)
	}
}

func extractHostname(req *http.Request) string {
	// Extract the host
	host := req.Host

	// Use net.SplitHostPort to separate the host and port, if present
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		// If there's an error, it might be because there's no port part
		// so use the host as is
		hostname = host
	}

	return hostname
}
