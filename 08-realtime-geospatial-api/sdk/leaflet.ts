import type { RgeoEntity, RgeoPosition, RgeoSubscription } from "./index";

interface LeafletMarker {
  setLatLng(value: [number, number]): void;
  remove(): void;
}

interface LeafletLike {
  marker(value: [number, number], options?: Record<string, unknown>): LeafletMarker;
}

interface LeafletMapLike {
  addLayer(layer: LeafletMarker): void;
}

export interface LeafletAdapterOptions {
  markerOptions?: Record<string, unknown>;
  removeAfterMs?: number;
}

export function attachToLeaflet(
  subscription: RgeoSubscription,
  map: LeafletMapLike,
  leaflet: LeafletLike,
  options: LeafletAdapterOptions = {}
): () => void {
  const markers = new Map<string, { marker: LeafletMarker; updatedAt: number }>();
  const removeAfterMs = options.removeAfterMs ?? 30_000;

  const unsubscribe = subscription.onMessage((event) => {
    if (event.t === "snapshot") {
      clear();
      event.entities.forEach((entity) => upsert(fromEntity(entity)));
    }
    if (event.t === "pos") {
      upsert(event);
    }
    if (event.t === "pos_batch") {
      event.items.forEach(upsert);
    }
    prune();
  });

  function upsert(entity: RgeoPosition): void {
    const existing = markers.get(entity.e);
    if (existing) {
      existing.marker.setLatLng([entity.lat, entity.lng]);
      existing.updatedAt = entity.ts;
      return;
    }
    const marker = leaflet.marker([entity.lat, entity.lng], options.markerOptions);
    map.addLayer(marker);
    markers.set(entity.e, { marker, updatedAt: entity.ts });
  }

  function prune(): void {
    const cutoff = Date.now() - removeAfterMs;
    for (const [id, record] of markers) {
      if (record.updatedAt < cutoff) {
        record.marker.remove();
        markers.delete(id);
      }
    }
  }

  function clear(): void {
    for (const record of markers.values()) {
      record.marker.remove();
    }
    markers.clear();
  }

  return () => {
    unsubscribe();
    clear();
  };
}

function fromEntity(entity: RgeoEntity): RgeoPosition {
  return { e: entity.entity_id, lat: entity.lat, lng: entity.lng, ts: entity.ts, meta: entity.meta };
}
