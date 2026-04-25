import { compactPosition, inBbox, isBbox, normalizePosition, type Bbox, type EntityState, type PositionInput } from "./protocol";

interface Env {
  COALESCE_MS?: string;
  DEFAULT_RETENTION_TTL_SECONDS?: string;
}

interface SocketAttachment {
  session: string;
  tenant_id: string;
  group_id: string;
  bbox?: Bbox;
  snapshot: boolean;
  connected_at: number;
}

export class GroupHub implements DurableObject {
  private entities = new Map<string, EntityState>();
  private pending = new Set<string>();
  private flushTimer: ReturnType<typeof setTimeout> | undefined;
  private loaded = false;

  constructor(private readonly state: DurableObjectState, private readonly env: Env) {}

  async fetch(request: Request): Promise<Response> {
    await this.load();
    const url = new URL(request.url);

    if (request.method === "POST" && url.pathname === "/internal/publish") {
      const body = await request.json<{ items: PositionInput[] }>();
      for (const item of body.items) {
        this.acceptPosition(normalizePosition(item));
      }
      return json({ ok: true, accepted: body.items.length });
    }

    if (request.method === "GET" && url.pathname === "/internal/stats") {
      return json({
        entities: this.entities.size,
        sockets: this.getSockets().length,
        pending: this.pending.size
      });
    }

    if (request.method === "GET" && url.pathname === "/internal/subscribe") {
      if (request.headers.get("upgrade") !== "websocket") {
        return json({ error: "upgrade_required" }, 426);
      }
      const pair = new WebSocketPair();
      const client = pair[0];
      const server = pair[1];
      const attachment: SocketAttachment = {
        session: crypto.randomUUID(),
        tenant_id: required(url.searchParams.get("tenant_id"), "tenant_id"),
        group_id: required(url.searchParams.get("group_id"), "group_id"),
        bbox: parseAttachmentBbox(url.searchParams.get("bbox")),
        snapshot: url.searchParams.get("snapshot") !== "false",
        connected_at: Date.now()
      };
      server.serializeAttachment(attachment);
      this.acceptSocket(server);
      this.sendHelloAndSnapshot(server, attachment);
      return new Response(null, { status: 101, webSocket: client });
    }

    return json({ error: "not_found" }, 404);
  }

  async webSocketMessage(ws: WebSocket, message: string | ArrayBuffer): Promise<void> {
    if (typeof message !== "string") {
      this.safeSend(ws, { t: "error", code: "BAD_PROTOCOL", msg: "Only JSON text frames are supported." });
      return;
    }

    let parsed: unknown;
    try {
      parsed = JSON.parse(message);
    } catch {
      this.safeSend(ws, { t: "error", code: "BAD_JSON", msg: "Message must be valid JSON." });
      return;
    }

    const attachment = ws.deserializeAttachment() as SocketAttachment;
    const input = parsed as { t?: string; bbox?: unknown };
    if (input.t === "pong") {
      return;
    }
    if (input.t === "update_filter") {
      if (input.bbox !== undefined && !isBbox(input.bbox)) {
        this.safeSend(ws, { t: "error", code: "BAD_FILTER", msg: "bbox must be [minLat,minLng,maxLat,maxLng]." });
        return;
      }
      const next = { ...attachment, bbox: input.bbox as Bbox | undefined };
      ws.serializeAttachment(next);
      this.sendSnapshot(ws, next);
      return;
    }

    this.safeSend(ws, { t: "error", code: "BAD_PROTOCOL", msg: "Unsupported message type." });
  }

  async webSocketClose(): Promise<void> {
    await this.persist();
  }

  async alarm(): Promise<void> {
    await this.persist();
  }

  private acceptPosition(position: PositionInput): void {
    this.entities.set(position.entity_id, {
      entity_id: position.entity_id,
      lat: position.lat,
      lng: position.lng,
      ts: Date.now(),
      meta: position.meta
    });
    this.pending.add(position.entity_id);
    this.scheduleFlush();
  }

  private scheduleFlush(): void {
    if (this.flushTimer) {
      return;
    }
    this.flushTimer = setTimeout(() => {
      this.flushTimer = undefined;
      void this.flush();
    }, intVar(this.env.COALESCE_MS, 250));
  }

  private async flush(): Promise<void> {
    const ids = [...this.pending];
    this.pending.clear();
    const updates = ids.map((id) => this.entities.get(id)).filter((item): item is EntityState => Boolean(item));
    for (const ws of this.getSockets()) {
      const attachment = ws.deserializeAttachment() as SocketAttachment;
      const visible = updates.filter((entity) => inBbox(entity, attachment.bbox));
      if (visible.length === 0) {
        continue;
      }
      const payload = visible.length === 1
        ? { t: "pos", ...compactPosition(visible[0]) }
        : { t: "pos_batch", items: visible.map(compactPosition) };
      this.safeSend(ws, payload);
    }
    await this.persist();
  }

  private sendHelloAndSnapshot(ws: WebSocket, attachment: SocketAttachment): void {
    this.safeSend(ws, { t: "hello", session: attachment.session, server_time: Date.now(), heartbeat_s: 25 });
    if (attachment.snapshot) {
      this.sendSnapshot(ws, attachment);
    }
  }

  private sendSnapshot(ws: WebSocket, attachment: SocketAttachment): void {
    const cutoff = Date.now() - intVar(this.env.DEFAULT_RETENTION_TTL_SECONDS, 60) * 1000;
    const entities = [...this.entities.values()].filter((entity) => entity.ts >= cutoff && inBbox(entity, attachment.bbox));
    this.safeSend(ws, { t: "snapshot", entities });
  }

  private safeSend(ws: WebSocket, payload: unknown): void {
    try {
      ws.send(JSON.stringify(payload));
    } catch {
      try {
        ws.close(4503, "send failed; please reconnect");
      } catch {
        // Ignore failed close on already-closed sockets.
      }
    }
  }

  private acceptSocket(ws: WebSocket): void {
    if ("acceptWebSocket" in this.state && typeof this.state.acceptWebSocket === "function") {
      this.state.acceptWebSocket(ws);
      return;
    }
    ws.accept();
    ws.addEventListener("message", (event) => {
      void this.webSocketMessage(ws, event.data as string | ArrayBuffer);
    });
  }

  private getSockets(): WebSocket[] {
    if ("getWebSockets" in this.state && typeof this.state.getWebSockets === "function") {
      return this.state.getWebSockets();
    }
    return [];
  }

  private async load(): Promise<void> {
    if (this.loaded) {
      return;
    }
    const stored = await this.state.storage.get<EntityState[]>("entities");
    if (stored) {
      this.entities = new Map(stored.map((entity) => [entity.entity_id, entity]));
    }
    this.loaded = true;
  }

  private async persist(): Promise<void> {
    await this.state.storage.put("entities", [...this.entities.values()]);
    await this.state.storage.setAlarm(Date.now() + 30_000);
  }
}

function parseAttachmentBbox(raw: string | null): Bbox | undefined {
  if (!raw) {
    return undefined;
  }
  const bbox = JSON.parse(raw) as Bbox;
  if (!isBbox(bbox)) {
    throw new Error("Invalid bbox.");
  }
  return bbox;
}

function required(value: string | null, name: string): string {
  if (!value) {
    throw new Error(`${name} is required.`);
  }
  return value;
}

function intVar(value: string | undefined, fallback: number): number {
  const parsed = value ? Number.parseInt(value, 10) : Number.NaN;
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json; charset=utf-8" }
  });
}
