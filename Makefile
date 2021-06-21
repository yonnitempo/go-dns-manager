COMMIT_COUNTER=$(shell git rev-list --all --count)

NAME=arm-go-alpine


all:
	docker build -t  ${NAME}:v${COMMIT_COUNTER} .
	docker tag ${NAME}:v${COMMIT_COUNTER} ${NAME}:latest
