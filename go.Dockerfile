# source: https://github.com/vkd/Makefile/blob/master/go-service.Dockerfile
FROM golang AS builder

WORKDIR /workspace
COPY . /workspace

ARG APP
RUN go build -o /workspace/out cmd/${APP}/main.go

# FROM alpine

# WORKDIR /app
# COPY --from=builder /workspace/out /app/out

ENTRYPOINT ["/workspace/out"]
