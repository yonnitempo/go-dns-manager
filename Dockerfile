FROM arm32v6/alpine:3.12
# GET THIS FROM: https://github.com/docker-library/python/tree/master/3.9-rc/alpine3.12

RUN apk add --no-cache musl-dev go git
run go get fmt net/http log crypto/sha1 encoding/json io/ioutil net time google.golang.org/api/dns/v1 google.golang.org/api/option
workdir /repo/
copy *.go /repo/
run go build -o /bin/server

# ENTRYPOINT ["/bin/server"]
