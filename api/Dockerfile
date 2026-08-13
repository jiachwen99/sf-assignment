FROM golang:1.26-alpine AS build
WORKDIR /src
# go.sum arrives with the first dependency, so it is optional here.
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -o /out/server ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/server /app/server
EXPOSE 8080
CMD ["/app/server"]
