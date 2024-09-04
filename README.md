# traefik-redis-rate-limit

Traefik comes with a default rate limiter middleware, but the rate limiter doesn't share a state if you are using several instance of Traefik (think kubernetes HA deployment for example).

This plugin is here to solve this issue: using a Redis as a common state, this plugin implement the [token bucket algorithm](https://en.wikipedia.org/wiki/Token_bucket).

## Configuration

You need to setup the static and dynamic configuration

### Static configuration

The following declaration (given here in YAML) defines a plugin:

```
# Static configuration

experimental:
  plugins:
    cluster-ratelimit:
      moduleName: "github.com/nzin/traefik-cluster-ratelimit"
      version: "v1.0.0"
```

Here is an example of a file provider dynamic configuration (given here in YAML), where the interesting part is the http.middlewares section:

```
# Dynamic configuration

http:
  routers:
    my-router:
      rule: host(`demo.localhost`)
      service: service-foo
      entryPoints:
        - web
      middlewares:
        - my-middleware

  services:
   service-foo:
      loadBalancer:
        servers:
          - url: http://127.0.0.1:5000
  
  middlewares:
    my-middleware:
      plugin:
        cluster-ratelimit:
          average: 50
          burst: 100
```

## Extra configuration

The `average` and the `burst` are the number of allowed connection per second, there are other variables:

| Variable      | Description                                        | default    |
|---------------|----------------------------------------------------|------------|
| period        | the period (in seconds) of the rate limiter window | 1          |
| average       | allowed requests per "period"                      |            |
| burst         | allowed burst requests per "period"                |            |
| redisaddress  | address of the redis server                        | redis:6379 |
| redisdb       | redis db to use                                    | 0          |
| redispassword | redis authentication (if any)                      |            |

If you are using redispassword, but dont want to place it in clear text in the traefik configuration, you can specify a variable name, starting with '$'. For example `$REDIS_PASSWORD` will use the `REDIS_PASSWORD` environment variable


## Benchmark

You can test traefik with the rate limiter with some tools. For example with vegeta (you probably need to install it):
```
docker-compose up -d

echo "GET http://localhost:8000/" | vegeta attack -duration=5s -rate=200 | tee results.bin | vegeta report
```
