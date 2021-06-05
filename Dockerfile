FROM golang:1.15.6 AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/main ./cmd/main.go
#=========
FROM alpine:latest AS runtime
COPY --from=build /bin /
COPY ./config.json .
EXPOSE 6160
CMD ["./main"]