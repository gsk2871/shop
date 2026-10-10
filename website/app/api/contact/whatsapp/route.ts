import { isIP } from "node:net";
export const dynamic = "force-dynamic";
const noStore = { "Cache-Control": "no-store" };
export async function GET(request: Request) {
  if (
    new URL(request.url).search ||
    (request.headers.get("content-length") &&
      request.headers.get("content-length") !== "0")
  )
    return Response.json(
      { code: "INVALID_ARGUMENT" },
      { status: 400, headers: noStore },
    );
  const ip = request.headers.get("x-client-ip")?.trim() || "";
  if (!isIP(ip) || ip.includes("%"))
    return Response.json(
      { code: "INVALID_CLIENT_IP" },
      { status: 400, headers: noStore },
    );
  try {
    const response = await fetch(
      `${process.env.BACKEND_URL || "http://127.0.0.1:8080"}/api/v1/whatsapp-contact`,
      {
        method: "GET",
        cache: "no-store",
        credentials: "omit",
        headers: { "X-Client-IP": ip },
        signal: AbortSignal.timeout(12000),
      },
    );
    return Response.json(await response.json(), {
      status: response.status,
      headers: noStore,
    });
  } catch {
    return Response.json(
      { code: "SERVICE_UNAVAILABLE" },
      { status: 503, headers: noStore },
    );
  }
}
// Next normally synthesizes HEAD from GET. Explicitly reject it so a probe
// cannot advance the account cursor without an actual website contact request.
export function HEAD() {
  return new Response(null, { status: 405, headers: noStore });
}
