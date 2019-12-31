# Description

use docker service to auto scale backend service

```yaml
- handler: "tmpdocker"
  # required
  service_name: "demo-service"
  # if no request after 20m will scale the service to 0 , if have request will scale to 1
  freeze_timeout: "20m"
- handler: "reverse_proxy"
  # set the server docker service backend
  upstreams: [{ dial: "demo-service" }]
```

# Todo

- if return status code == 502 check docker service is healthy, if docker service scale is 0 clear hold timer

# Build

```sh
cd cmd/caddy
go build
./caddy list-modules | grep docker
# you will see `http.handlers.tmpdocker` plugin
```
