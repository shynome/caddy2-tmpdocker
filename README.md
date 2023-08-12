# Description

use docker service to auto scale backend service

```conf
# create docker service
# docker service create -p 8081:80 --name tmpdocker_test nginx:1.19.6-alpine@sha256:c2ce58e024275728b00a554ac25628af25c54782865b3487b11c21cafb7fabda
http://127.0.0.1:8080 {
    route {
        tmpdocker {
            service tmpdocker_test
            keep_alive 1m
            #keep_alive 5m
            #scale_timeout 10s
        }
        #tmpdocker tmpdocker_test
        reverse_proxy 127.0.0.1:8081
    }
}
```

# Build

```sh
git clone https://github.com/shynome/caddy2-tmpdocker.git
cd caddy2-tmpdocker
go build -o caddy ./cmd/caddy
./caddy list-modules | grep tmpdocker
# you will see `http.handlers.tmpdocker` plugin
```

# Test

```sh
docker service create --name tmpdocker_test -p 8081:80 --replicas 0 nginx:1.19.6-alpine@sha256:c2ce58e024275728b00a554ac25628af25c54782865b3487b11c21cafb7fabda
go run ./cmd/caddy run --config ./cmd/caddy/Caddyfile --adapter caddyfile
# another shell
docker service ls
curl http://127.0.0.1:8080
docker service ls
docker service rm test
```
