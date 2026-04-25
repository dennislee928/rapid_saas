export type Bbox = [number, number, number, number];

export interface RgeoClientOptions {
  baseUrl?: string;
  tokenEndpoint?: string;
  ephemeralToken?: string;
  fetch?: typeof fetch;
}

export interface SubscribeOptions {
  bbox?: Bbox;
  snapshot?: boolean;
}

export type RgeoEvent =
  | { t: "hello"; session: string; server_time: number; heartbeat_s: number }
  | { t: "snapshot"; entities: RgeoEntity[] }
  | { t: "pos"; e: string; lat: number; lng: number; ts: number; meta?: Record<string, unknown> }
  | { t: "pos_batch"; items: RgeoPosition[] }
  | { t: "ping"; ts: number }
  | { t: "error"; code: string; msg: string };

export interface RgeoEntity {
  entity_id: string;
  lat: number;
  lng: number;
  ts: number;
  meta?: Record<string, unknown>;
}

export interface RgeoPosition {
  e: string;
  lat: number;
  lng: number;
  ts: number;
  meta?: Record<string, unknown>;
}

export type RgeoListener = (event: RgeoEvent) => void;

export class RgeoClient {
  private readonly fetchImpl: typeof fetch;
  private readonly baseUrl: string;

  constructor(private readonly options: RgeoClientOptions) {
    this.fetchImpl = options.fetch ?? fetch;
    this.baseUrl = options.baseUrl ?? globalThis.location?.origin ?? "http://localhost:8787";
  }

  subscribe(groupId: string, options: SubscribeOptions = {}): RgeoSubscription {
    return new RgeoSubscription({
      groupId,
      baseUrl: this.baseUrl,
      tokenProvider: () => this.getToken(groupId),
      options
    });
  }

  private async getToken(groupId: string): Promise<string> {
    if (this.options.ephemeralToken) {
      return this.options.ephemeralToken;
    }
    if (!this.options.tokenEndpoint) {
      throw new Error("RgeoClient requires tokenEndpoint or ephemeralToken.");
    }
    const response = await this.fetchImpl(this.options.tokenEndpoint, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ group_id: groupId })
    });
    if (!response.ok) {
      throw new Error(`Token endpoint failed with HTTP ${response.status}.`);
    }
    const body = await response.json() as { token?: string };
    if (!body.token) {
      throw new Error("Token endpoint did not return token.");
    }
    return body.token;
  }
}

interface SubscriptionInit {
  groupId: string;
  baseUrl: string;
  tokenProvider: () => Promise<string>;
  options: SubscribeOptions;
}

export class RgeoSubscription extends EventTarget {
  private ws?: WebSocket;
  private listeners = new Set<RgeoListener>();
  private closed = false;
  private attempt = 0;
  private bbox?: Bbox;

  constructor(private readonly init: SubscriptionInit) {
    super();
    this.bbox = init.options.bbox;
    void this.connect();
    if (typeof document !== "undefined") {
      document.addEventListener("visibilitychange", () => this.handleVisibilityChange());
    }
  }

  onMessage(listener: RgeoListener): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  updateFilter(filter: { bbox?: Bbox }): void {
    this.bbox = filter.bbox;
    this.send({ t: "update_filter", bbox: filter.bbox });
  }

  close(): void {
    this.closed = true;
    this.ws?.close(1000, "client closed");
  }

  private async connect(): Promise<void> {
    const token = await this.init.tokenProvider();
    const url = new URL(`/v1/groups/${encodeURIComponent(this.init.groupId)}/ws`, this.init.baseUrl);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    url.searchParams.set("token", token);
    url.searchParams.set("snapshot", String(this.init.options.snapshot ?? true));
    if (this.bbox) {
      url.searchParams.set("bbox", this.bbox.join(","));
    }

    this.ws = new WebSocket(url);
    this.ws.addEventListener("open", () => {
      this.attempt = 0;
    });
    this.ws.addEventListener("message", (event) => this.handleMessage(event.data));
    this.ws.addEventListener("close", () => this.reconnect());
  }

  private handleMessage(data: unknown): void {
    if (typeof data !== "string") {
      return;
    }
    const event = JSON.parse(data) as RgeoEvent;
    if (event.t === "ping") {
      this.send({ t: "pong", ts: Date.now() });
    }
    this.listeners.forEach((listener) => listener(event));
    this.dispatchEvent(new CustomEvent("message", { detail: event }));
  }

  private reconnect(): void {
    if (this.closed) {
      return;
    }
    const backoff = Math.min(5000, 250 * 2 ** this.attempt) + Math.floor(Math.random() * 200);
    this.attempt += 1;
    setTimeout(() => void this.connect(), backoff);
  }

  private handleVisibilityChange(): void {
    if (document.visibilityState === "hidden") {
      this.send({ t: "update_filter", bbox: [0, 0, 0, 0] });
      return;
    }
    this.send({ t: "update_filter", bbox: this.bbox });
  }

  private send(payload: unknown): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(payload));
    }
  }
}
