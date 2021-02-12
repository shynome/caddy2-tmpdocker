# Description

use docker service to auto scale backend service

```conf
# create docker service
# docker service create -p 8081:80 --name test nginx:1.19.6-alpine@sha256:c2ce58e024275728b00a554ac25628af25c54782865b3487b11c21cafb7fabda
http://127.0.0.1:8080 {
    route {
        tmpdocker {
            service test
            wait 1m
            #wait 20m
            #scale_timeout 10s
        }
        #tmpdocker test
        reverse_proxy 127.0.0.1:8081
    }
}
```

# Build

```sh
git clone https://github.com/shynome/caddy2-tmpdocker.git
cd caddy2-tmpdocker
# download vendor way 1
git clone https://github.com/shynome/caddy2-tmpdocker-vendor.git vendor
# or download vendor way 2
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
