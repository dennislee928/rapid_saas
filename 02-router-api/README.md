# Security Webhook Router API

Go 1.23 chi service skeleton for the security webhook router backend. This service is owned from the repository root folder `02-router-api/`; commands below assume that as the working directory.

## Development

```sh
env GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache go test ./...
```

Runtime configuration:

- `HTTP_ADDR`: listen address, default `:8080`.
- `WORKER_SHARED_SECRET`: optional bearer token required by `POST /_q/consume`.
- `LOG_LEVEL`: `debug`, `info`, `warn`, or `error`.

## Routes

- `GET /healthz`
- `GET /readyz`
- `POST /_q/consume`
- `GET|POST /v1/endpoints`
- `GET|PUT|DELETE /v1/endpoints/{endpointID}`
- `GET|POST /v1/destinations`
- `GET|PUT|DELETE /v1/destinations/{destinationID}`
- `GET|POST /v1/rules`
- `GET|PUT|DELETE /v1/rules/{ruleID}`

Admin routes currently use `X-Tenant-ID` as a placeholder tenant context. The repository interfaces are shaped so a SQLite/libSQL/sqlc implementation can replace the in-memory store without changing handlers.

## Queue Payload

```json
{
  "events": [
    {
      "id": "evt_123",
      "tenant_id": "tenant_123",
      "endpoint_id": "ep_123",
      "headers": {"content-type": "application/json"},
      "body": {"severity": "High"},
      "received_at_ms": 1714000000000,
      "request_hash": "sha256..."
    }
  ]
}
```
