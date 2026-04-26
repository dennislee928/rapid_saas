FROM golang:1.23-alpine AS build

WORKDIR /src
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o /out/aegis-worker ./cmd/aegis-worker

FROM alpine:3.20
RUN adduser -D -H appuser
WORKDIR /app
COPY --from=build /out/aegis-worker /app/aegis-worker
USER appuser

CMD ["/app/aegis-worker", "-mode", "once"]
