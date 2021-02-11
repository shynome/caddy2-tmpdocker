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

- <del>
  we can't do this because proxy is not a middleware<br/>
  if return status code == 502 check docker service is healthy, if docker service scale is 0 clear hold timer
  </del>

# Build

```sh
git clone https://github.com/shynome/caddy2-tmpdocker.git
cd caddy2-tmpdocker
# download vendor 1
git clone https://github.com/shynome/caddy2-tmpdocker-vendor.git vendor
# download vendor 2
go mod download
cd cmd/caddy
go build -mod=vendor -o caddy
./caddy list-modules | grep docker
# you will see `http.handlers.tmpdocker` plugin
```

# Test

```sh
docker service create -p 8081:80 --name test nginx:1.19.6-alpine@sha256:c2ce58e024275728b00a554ac25628af25c54782865b3487b11c21cafb7fabda
cd cmd/caddy
go run . run -config Caddyfile --adapter caddyfile
# another shell
docker service ls
curl http://127.0.0.1:8080
docker service ls
docker service rm test
```
