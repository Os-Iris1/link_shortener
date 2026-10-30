FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/shorter ./cmd/shorter

FROM alpine:3.23
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/shorter /app/shorter
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/app/shorter"]
