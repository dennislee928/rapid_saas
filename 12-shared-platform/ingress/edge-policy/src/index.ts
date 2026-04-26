export type EdgePolicyOptions = {
  maxBodyBytes?: number;
  requireAuth?: boolean;
  allowedAuthHeaders?: string[];
};

export type EdgePolicyResult = {
  request: Request;
  requestId: string;
  response?: Response;
};

const DEFAULT_MAX_BODY_BYTES = 1024 * 1024;
const DEFAULT_AUTH_HEADERS = [
  "authorization",
  "x-api-key",
  "x-signature",
  "x-router-signature",
  "x-routekit-sandbox-signature",
];

export function applyEdgePolicy(
  request: Request,
  options: EdgePolicyOptions = {},
): EdgePolicyResult {
  const requestId = request.headers.get("x-request-id") || crypto.randomUUID();
  const headers = new Headers(request.headers);
  headers.set("x-request-id", requestId);
  headers.set("x-correlation-id", requestId);

  const maxBodyBytes = options.maxBodyBytes ?? DEFAULT_MAX_BODY_BYTES;
  const contentLength = request.headers.get("content-length");
  if (contentLength && Number(contentLength) > maxBodyBytes) {
    return {
      request,
      requestId,
      response: jsonError("payload_too_large", 413, requestId),
    };
  }

  if (options.requireAuth ?? true) {
    const authHeaders = options.allowedAuthHeaders ?? DEFAULT_AUTH_HEADERS;
    const hasAuth = authHeaders.some((header) => request.headers.has(header));
    if (!hasAuth) {
      return {
        request,
        requestId,
        response: jsonError("missing_gateway_auth_material", 401, requestId),
      };
    }
  }

  return {
    request: new Request(request, { headers }),
    requestId,
  };
}

export function withSecurityHeaders(response: Response, requestId: string): Response {
  const headers = new Headers(response.headers);
  headers.set("x-request-id", requestId);
  headers.set("x-content-type-options", "nosniff");
  headers.set("x-frame-options", "DENY");
  headers.set("referrer-policy", "no-referrer");
  headers.set("permissions-policy", "camera=(), microphone=(), geolocation=()");
  headers.set("content-security-policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'");
  headers.set("strict-transport-security", "max-age=31536000; includeSubDomains");
  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers,
  });
}

function jsonError(error: string, status: number, requestId: string): Response {
  return withSecurityHeaders(
    Response.json({ error }, { status }),
    requestId,
  );
}
