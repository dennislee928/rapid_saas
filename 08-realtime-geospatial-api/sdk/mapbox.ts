import type { RgeoEntity, RgeoPosition, RgeoSubscription } from "./index";

type PointFeature = {
  type: "Feature";
  geometry: { type: "Point"; coordinates: [number, number] };
  properties: Record<string, unknown>;
};

type FeatureCollection = {
  type: "FeatureCollection";
  features: PointFeature[];
};

interface GeoJsonSource {
  setData(data: FeatureCollection): void;
}

interface MapboxLikeMap {
  addSource(id: string, source: { type: "geojson"; data: FeatureCollection }): void;
  getSource(id: string): GeoJsonSource | undefined;
  addLayer(layer: Record<string, unknown>): void;
  getLayer(id: string): unknown;
}

export interface MapboxAdapterOptions {
  sourceId?: string;
  layerId?: string;
  iconImage?: string;
  smoothMs?: number;
  removeAfterMs?: number;
}

export function attachToMapbox(subscription: RgeoSubscription, map: MapboxLikeMap, options: MapboxAdapterOptions = {}): () => void {
  return attachSymbolLayer(subscription, map, options);
}

export function attachSymbolLayer(subscription: RgeoSubscription, map: MapboxLikeMap, options: MapboxAdapterOptions = {}): () => void {
  const sourceId = options.sourceId ?? "rgeo-entities";
  const layerId = options.layerId ?? "rgeo-entities";
  const removeAfterMs = options.removeAfterMs ?? 30_000;
  const entities = new Map<string, RgeoPosition>();

  if (!map.getSource(sourceId)) {
    map.addSource(sourceId, { type: "geojson", data: emptyCollection() });
  }
  if (!map.getLayer(layerId)) {
    map.addLayer({
      id: layerId,
      type: "symbol",
      source: sourceId,
      layout: {
        "icon-image": options.iconImage ?? "marker-15",
        "icon-allow-overlap": true
      }
    });
  }

  const unsubscribe = subscription.onMessage((event) => {
    if (event.t === "snapshot") {
      entities.clear();
      event.entities.forEach((entity) => entities.set(entity.entity_id, fromEntity(entity)));
      render();
    }
    if (event.t === "pos") {
      entities.set(event.e, event);
      render();
    }
    if (event.t === "pos_batch") {
      event.items.forEach((item) => entities.set(item.e, item));
      render();
    }
  });

  function render(): void {
    const cutoff = Date.now() - removeAfterMs;
    for (const [id, entity] of entities) {
      if (entity.ts < cutoff) {
        entities.delete(id);
      }
    }
    map.getSource(sourceId)?.setData({
      type: "FeatureCollection",
      features: [...entities.values()].map((entity) => ({
        type: "Feature",
        geometry: { type: "Point", coordinates: [entity.lng, entity.lat] },
        properties: { id: entity.e, ts: entity.ts, ...entity.meta }
      }))
    });
  }

  return unsubscribe;
}

function fromEntity(entity: RgeoEntity): RgeoPosition {
  return { e: entity.entity_id, lat: entity.lat, lng: entity.lng, ts: entity.ts, meta: entity.meta };
}

function emptyCollection(): FeatureCollection {
  return { type: "FeatureCollection", features: [] };
}
