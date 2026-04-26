FROM golang:1.23-alpine AS build

WORKDIR /src
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o /out/audio-orchestrator ./cmd/orchestrator

FROM alpine:3.20
RUN adduser -D -H appuser
WORKDIR /app
COPY --from=build /out/audio-orchestrator /app/audio-orchestrator
USER appuser

EXPOSE 8080
CMD ["/app/audio-orchestrator"]
