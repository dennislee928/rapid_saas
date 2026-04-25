export type Bbox = [number, number, number, number];

export interface EntityState {
  entity_id: string;
  lat: number;
  lng: number;
  ts: number;
  meta?: Record<string, unknown>;
}

export interface PositionInput {
  group_id: string;
  entity_id: string;
  lat: number;
  lng: number;
  ts?: number;
  meta?: Record<string, unknown>;
}

export interface SubscribeClaims {
  tenant_id: string;
  group_id: string;
  exp: number;
  allowed_filters?: {
    bbox?: Bbox;
  };
}

export type ServerMessage =
  | { t: "hello"; session: string; server_time: number; heartbeat_s: number }
  | { t: "snapshot"; entities: EntityState[] }
  | { t: "pos"; e: string; lat: number; lng: number; ts: number; meta?: Record<string, unknown> }
  | { t: "pos_batch"; items: Array<{ e: string; lat: number; lng: number; ts: number; meta?: Record<string, unknown> }> }
  | { t: "ping"; ts: number }
  | { t: "error"; code: string; msg: string };

export function isBbox(value: unknown): value is Bbox {
  return Array.isArray(value)
    && value.length === 4
    && value.every((item) => typeof item === "number" && Number.isFinite(item))
    && value[0] <= value[2]
    && value[1] <= value[3]
    && value[0] >= -90
    && value[2] <= 90
    && value[1] >= -180
    && value[3] <= 180;
}

export function inBbox(entity: Pick<EntityState, "lat" | "lng">, bbox?: Bbox): boolean {
  if (!bbox) {
    return true;
  }
  return entity.lat >= bbox[0] && entity.lat <= bbox[2] && entity.lng >= bbox[1] && entity.lng <= bbox[3];
}

export function normalizePosition(input: unknown, now = Date.now()): PositionInput {
  const raw = input as Partial<PositionInput>;
  if (!raw || typeof raw !== "object") {
    throw new Error("Position body must be a JSON object.");
  }
  if (!isSafeId(raw.group_id)) {
    throw new Error("group_id is required and may only contain letters, numbers, '.', '_', ':' and '-'.");
  }
  if (!isSafeId(raw.entity_id)) {
    throw new Error("entity_id is required and may only contain letters, numbers, '.', '_', ':' and '-'.");
  }
  if (!validLat(raw.lat) || !validLng(raw.lng)) {
    throw new Error("lat/lng must be valid WGS84 coordinates.");
  }
  if (raw.meta !== undefined && byteLength(JSON.stringify(raw.meta)) > 256) {
    throw new Error("meta must be 256 bytes or smaller.");
  }

  return {
    group_id: raw.group_id,
    entity_id: raw.entity_id,
    lat: raw.lat,
    lng: raw.lng,
    ts: typeof raw.ts === "number" && Number.isFinite(raw.ts) ? raw.ts : now,
    meta: raw.meta && typeof raw.meta === "object" && !Array.isArray(raw.meta) ? raw.meta : undefined
  };
}

export function parseBboxParam(value: string | null): Bbox | undefined {
  if (!value) {
    return undefined;
  }
  const bbox = value.split(",").map((part) => Number.parseFloat(part));
  if (!isBbox(bbox)) {
    throw new Error("bbox must be minLat,minLng,maxLat,maxLng.");
  }
  return bbox;
}

export function compactPosition(entity: EntityState) {
  return { e: entity.entity_id, lat: entity.lat, lng: entity.lng, ts: entity.ts, meta: entity.meta };
}

function isSafeId(value: unknown): value is string {
  return typeof value === "string" && /^[A-Za-z0-9_.:-]{1,128}$/.test(value);
}

function validLat(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value) && value >= -90 && value <= 90;
}

function validLng(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value) && value >= -180 && value <= 180;
}

function byteLength(value: string): number {
  return new TextEncoder().encode(value).byteLength;
}
