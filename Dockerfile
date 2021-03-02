FROM golang:1.15.6-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/main github.com/NubeS3/cloud/cmd/main.go
#=========
FROM scratch AS bin
COPY --from=build /bin /
COPY --from=build /src/config.json /
EXPOSE 6160:6160
CMD ["./main"]