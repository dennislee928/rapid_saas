# Cloudflare Deployment Notes

This scaffold assumes Cloudflare is used for edge ingress and dashboard hosting:

- Workers receive customer webhooks at `https://in.<domain>/w/<endpoint_id>`.
- Workers KV stores cached endpoint signing/routing metadata.
- Cloudflare Queues buffers events before the Go router consumes them on Fly.io.
- Pages can host the dashboard when the frontend app exists.

## Ingress Worker

1. Create a queue:

   ```sh
   wrangler queues create security-webhook-events
   ```

2. Create endpoint config KV namespaces:

   ```sh
   wrangler kv namespace create router-endpoints
   wrangler kv namespace create router-endpoints --preview
   ```

3. Copy `infra/cloudflare/wrangler.ingress.example.toml` into the worker project as `wrangler.toml`.

4. Store secrets:

   ```sh
   wrangler secret put QUEUE_PUSH_AUTH_TOKEN
   ```

5. Deploy:

   ```sh
   wrangler deploy
   ```

## Dashboard Pages

When the dashboard app exists, copy `infra/cloudflare/wrangler.dashboard.example.toml` into that project or wire the same values into the Cloudflare Pages dashboard.

Required production values:

- `NEXT_PUBLIC_API_BASE_URL`
- `NEXT_PUBLIC_INGRESS_BASE_URL`
- Clerk publishable/secret keys through the Pages environment

## Queue Consumer Contract

The Worker should push batches to the Fly router using `ROUTER_PUSH_URL`. Include an `Authorization: Bearer <QUEUE_PUSH_AUTH_TOKEN>` header so the Go service can reject direct unauthenticated queue pushes.
