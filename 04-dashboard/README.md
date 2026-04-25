# Security Webhook Router Dashboard

Next.js 15 dashboard scaffold for the security webhook router MVP. It targets Cloudflare Pages and keeps API, Clerk, and replay actions as explicit placeholders until the backend contracts are available.

## Routes

- `/` landing page
- `/dashboard` endpoint list, event timeline, quota meter, Clerk placeholder
- `/rules` raw JSONLogic rule editor placeholder
- `/dlq` dead-letter queue and replay placeholder

## Commands

```sh
npm install
npm run dev
npm run build
npm run pages:build
npm run pages:preview
```
