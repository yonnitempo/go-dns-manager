FROM arm32v6/alpine:3.12
# GET THIS FROM: https://github.com/docker-library/python/tree/master/3.9-rc/alpine3.12

rUN apk add --no-cache musl-dev go
