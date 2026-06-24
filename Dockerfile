FROM golang:1.26.4-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/web ./cmd/web

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=build /out/web /app/web
EXPOSE 8090
CMD ["/app/web"]
