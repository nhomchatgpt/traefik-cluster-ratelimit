# traefik-redis-rate-limit

Traefik comes with a default [rate limiter](https://doc.traefik.io/traefik/middlewares/http/ratelimit/) middleware, but the rate limiter doesn't share a state if you are using several instance of Traefik (think kubernetes HA deployment for example).

This plugin is here to solve this issue: using a Redis as a common state, this plugin implement the [token bucket algorithm](https://en.wikipedia.org/wiki/Token_bucket).

## Configuration

You need to setup the static and dynamic configuration

The following declaration (given here in YAML) defines a plugin:

```
# Static configuration

experimental:
  plugins:
    clusterRatelimit:
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
        clusterRatelimit:
          average: 50
          burst: 100
```

With a kubernetesingress provider:

```
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: clusterratelimit
  namespace: ingress-traefik
spec:
  clusterRatelimit:
    average: 100
    burst: 200
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
  namespace: ingress-traefik
  annotations:
    traefik.ingress.kubernetes.io/router.middlewares: ingress-traefik-clusterratelimit@kubernetescrd
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: example-service
            port:
              number: 80
```

## Extra configuration

The `average` and the `burst` are the number of allowed connection per second, there are other variables:

| Variable                    | Description                                        | default    |
|-----------------------------|----------------------------------------------------|------------|
| period                      | the period (in seconds) of the rate limiter window | 1          |
| average                     | allowed requests per "period" ( 0 = unlimited)     |            |
| burst                       | allowed burst requests per "period"                |            |
| redisaddress                | address of the redis server                        | redis:6379 |
| redisdb                     | redis db to use                                    | 0          |
| redispassword               | redis authentication (if any)                      |            |
| sourceCriterion.*           | defines what criterion is used to group requests. See next | ipStrategy |
| sourceCriterion.ipStrategy  | client IP based source                             |            |
| sourceCriterion.ipStrategy.depth | tells Traefik to use the X-Forwarded-For header and select the IP located at the depth position |    |
| sourceCriterion.ipStrategy.excludedIPs | list of X-Forwarded-For IPs that are to be excluded | |
| sourceCriterion.requestHost | based source on request host                       |            |
| sourceCriterion.requestHeaderName | Name of the header used to group incoming requests|       |
| breakerthreshold            | number of failed connection before pausing Redis   | 3          |
| breakerreattempt            | nb seconds before attempting to reconnect to Redis | 15         |

Notes:
- for more information about sourceCriteron check the Traefik [ratelimit](https://doc.traefik.io/traefik/middlewares/http/ratelimit/) page
- regarding redispassword, if you dont want to set it in clear text in the traefik configuration, you can specify a variable name starting with '$'. For example `$REDIS_PASSWORD` will use the `REDIS_PASSWORD` environment variable

## Circuit-breaker

If the Redis server is not available, we will stop talking to it, and let pass through.
There are 2 environment variables you can use to change the default behaviour

## Benchmark

You can test traefik with the rate limiter with some tools. For example with vegeta (you probably need to install it):
```
docker-compose up -d

echo "GET http://localhost:8000/" | vegeta attack -duration=5s -rate=200 | tee results.bin | vegeta report
```
