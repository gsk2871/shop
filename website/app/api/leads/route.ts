import { isIP } from "node:net";
export const dynamic = "force-dynamic";
const noStore = { "Cache-Control": "no-store" };
const failure = (status: number, code: string) => Response.json({ code }, { status, headers: noStore });
export async function POST(request: Request) {
  const origin = request.headers.get("origin");
  if (request.headers.get("sec-fetch-site") === "cross-site") return failure(403, "INVALID_ORIGIN");
  if (origin) {
    try { if (new URL(origin).host !== (request.headers.get("host") || new URL(request.url).host)) return failure(403, "INVALID_ORIGIN"); }
    catch { return failure(403, "INVALID_ORIGIN"); }
  }
  if (new URL(request.url).search || request.headers.get("content-type")?.split(";")[0].trim() !== "application/json") return failure(400, "INVALID_ARGUMENT");
  const ip = request.headers.get("x-client-ip")?.trim() || "";
  if (!isIP(ip) || ip.includes("%")) return failure(400, "INVALID_CLIENT_IP");
  // Bound the body while streaming, including requests without Content-Length.
  const reader = request.body?.getReader();
  if (!reader) return failure(400, "INVALID_ARGUMENT");
  let length = 0;
  const chunks: Uint8Array[] = [];
  let timedOut = false;
  const timer = setTimeout(() => { timedOut = true; void reader.cancel().catch(() => {}); }, 5000);
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (timedOut) return failure(408, "REQUEST_TIMEOUT");
      if (done) break;
      length += value.byteLength;
      if (length > 4096) { await reader.cancel(); return failure(413, "BODY_TOO_LARGE"); }
      chunks.push(value);
    }
  } catch { return failure(400, "INVALID_ARGUMENT"); }
  finally { clearTimeout(timer); reader.releaseLock(); }
  let contact: string;
  try {
    const body = JSON.parse(Buffer.concat(chunks).toString("utf8"));
    if (!body || typeof body.contact !== "string" || Object.keys(body).some((k) => k !== "contact") || body.contact.length > 254) return failure(400, "INVALID_ARGUMENT");
    contact = body.contact;
  } catch { return failure(400, "INVALID_ARGUMENT"); }
  try {
    const response = await fetch(`${process.env.BACKEND_URL || "http://127.0.0.1:8080"}/api/v1/leads`, {
      method: "POST", cache: "no-store", credentials: "omit",
      headers: { "Content-Type": "application/json", "X-Client-IP": ip },
      body: JSON.stringify({ contact }), signal: AbortSignal.timeout(12000),
    });
    const headers: Record<string, string> = { ...noStore };
    if (response.status === 429) headers["Retry-After"] = "600";
    return Response.json(await response.json(), { status: response.status, headers });
  } catch { return failure(503, "SERVICE_UNAVAILABLE"); }
}
