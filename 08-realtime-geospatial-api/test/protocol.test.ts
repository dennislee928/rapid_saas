import { describe, expect, it } from "vitest";
import { inBbox, normalizePosition, parseBboxParam } from "../src/protocol";

describe("protocol validation", () => {
  it("normalizes valid position payloads", () => {
    expect(normalizePosition({
      group_id: "fleet-london-1",
      entity_id: "driver-1",
      lat: 51.51,
      lng: -0.118,
      meta: { battery: 0.83 }
    }, 123)).toEqual({
      group_id: "fleet-london-1",
      entity_id: "driver-1",
      lat: 51.51,
      lng: -0.118,
      ts: 123,
      meta: { battery: 0.83 }
    });
  });

  it("rejects malformed coordinates", () => {
    expect(() => normalizePosition({
      group_id: "fleet",
      entity_id: "driver",
      lat: 95,
      lng: 1
    })).toThrow(/lat\/lng/);
  });

  it("parses and applies bbox filters", () => {
    const bbox = parseBboxParam("51.50,-0.13,51.52,-0.10");

    expect(bbox).toEqual([51.5, -0.13, 51.52, -0.1]);
    expect(inBbox({ lat: 51.51, lng: -0.12 }, bbox)).toBe(true);
    expect(inBbox({ lat: 51.6, lng: -0.12 }, bbox)).toBe(false);
  });
});
