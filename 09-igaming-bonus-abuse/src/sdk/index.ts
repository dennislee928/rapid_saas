import type { BrowserSignals } from "../shared/types";

export interface TiltGuardOptions {
  endpoint: string;
  tenantId: string;
  sessionId?: string;
}

export interface TiltGuardCollectResult {
  visit_token: string;
  visit_id: string;
  device_id: string;
  expires_at: number;
}

const SDK_VERSION = "0.1.0";

export async function collect(options: TiltGuardOptions): Promise<TiltGuardCollectResult> {
  const signals = await collectSignals();
  const response = await fetch(new URL("/collect", options.endpoint), {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      tenant_id: options.tenantId,
      session_id: options.sessionId,
      signals
    })
  });
  if (!response.ok) {
    throw new Error(`TiltGuard collect failed with HTTP ${response.status}`);
  }
  return response.json() as Promise<TiltGuardCollectResult>;
}

export async function collectSignals(): Promise<BrowserSignals> {
  const nav = globalThis.navigator;
  const screen = globalThis.screen;
  const signals: BrowserSignals = {
    sdkVersion: SDK_VERSION,
    userAgent: nav?.userAgent,
    language: nav?.language,
    languages: nav?.languages ? Array.from(nav.languages) : undefined,
    timezone: safeTimezone(),
    screen: screen ? {
      width: screen.width,
      height: screen.height,
      colorDepth: screen.colorDepth,
      devicePixelRatio: globalThis.devicePixelRatio
    } : undefined,
    hardwareConcurrency: nav?.hardwareConcurrency,
    deviceMemory: readDeviceMemory(nav),
    webdriver: nav?.webdriver,
    canvasHash: await hashCanvasDemoSafe(),
    webglVendor: readWebglInfo().vendor,
    webglRenderer: readWebglInfo().renderer,
    fontsHash: await hashFontsDemoSafe(),
    storageQuota: await readStorageQuota(),
    behaviour: {
      mouseEvents: 0,
      mouseStraightLineRatio: 0,
      typingCadenceStddevMs: 0,
      pasteEvents: 0
    }
  };
  return signals;
}

function safeTimezone(): string | undefined {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone;
  } catch {
    return undefined;
  }
}

function readDeviceMemory(nav: Navigator | undefined): number | undefined {
  return typeof nav === "object" && "deviceMemory" in nav ? Number(nav.deviceMemory) : undefined;
}

async function hashCanvasDemoSafe(): Promise<string | undefined> {
  if (!globalThis.document) {
    return undefined;
  }
  const canvas = document.createElement("canvas");
  canvas.width = 220;
  canvas.height = 40;
  const context = canvas.getContext("2d");
  if (!context) {
    return undefined;
  }
  context.textBaseline = "top";
  context.fillStyle = "#f60";
  context.fillRect(0, 0, 220, 40);
  context.fillStyle = "#069";
  context.font = "16px serif";
  context.fillText("TiltGuard demo signal", 4, 8);
  return sha256Hex(canvas.toDataURL());
}

function readWebglInfo(): { vendor?: string; renderer?: string } {
  if (!globalThis.document) {
    return {};
  }
  const canvas = document.createElement("canvas");
  const gl = canvas.getContext("webgl");
  if (!gl) {
    return {};
  }
  const debug = gl.getExtension("WEBGL_debug_renderer_info");
  if (!debug) {
    return {};
  }
  return {
    vendor: String(gl.getParameter(debug.UNMASKED_VENDOR_WEBGL)),
    renderer: String(gl.getParameter(debug.UNMASKED_RENDERER_WEBGL))
  };
}

async function hashFontsDemoSafe(): Promise<string | undefined> {
  if (!globalThis.document) {
    return undefined;
  }
  const candidates = ["Arial", "Courier New", "Georgia", "Times New Roman", "Verdana"];
  const body = candidates.join("|");
  return sha256Hex(body);
}

async function readStorageQuota(): Promise<number | undefined> {
  try {
    const estimate = await navigator.storage?.estimate();
    return estimate?.quota;
  } catch {
    return undefined;
  }
}

async function sha256Hex(value: string): Promise<string> {
  const bytes = new Uint8Array(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(value)));
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

declare global {
  interface Window {
    TiltGuard?: { collect: typeof collect; collectSignals: typeof collectSignals };
  }
}

if (typeof window !== "undefined") {
  window.TiltGuard = { collect, collectSignals };
}
