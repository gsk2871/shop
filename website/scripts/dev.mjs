// The local ingress overwrites IP headers exactly as the production proxy does.
// Next listens privately behind it, so browser-supplied headers cannot choose
// a visitor country. This also forwards the development HMR websocket.
import http from "node:http";
import net from "node:net";
import { spawn } from "node:child_process";
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
const upstreamPort = 3001;
const next = spawn(
  process.execPath,
  [
    require.resolve("next/dist/bin/next"),
    "dev",
    "--hostname",
    "127.0.0.1",
    "--port",
    String(upstreamPort),
  ],
  { stdio: "inherit" },
);
function headers(request) {
  const ip = (request.socket.remoteAddress || "127.0.0.1").replace(
    /^::ffff:/,
    "",
  );
  return {
    ...request.headers,
    "x-client-ip": ip,
    "x-forwarded-for": ip,
    "x-real-ip": ip,
    "x-forwarded-proto": "http",
  };
}
const proxy = http.createServer((request, response) => {
  const upstream = http.request(
    {
      hostname: "127.0.0.1",
      port: upstreamPort,
      method: request.method,
      path: request.url,
      headers: headers(request),
    },
    (incoming) => {
      response.writeHead(incoming.statusCode || 502, incoming.headers);
      incoming.pipe(response);
    },
  );
  upstream.on("error", () => {
    if (!response.headersSent) response.writeHead(503);
    response.end("Website is starting. Please try again shortly.");
  });
  request.on("aborted", () => upstream.destroy());
  request.pipe(upstream);
});
proxy.on("upgrade", (request, socket, head) => {
  const upstream = net.connect(upstreamPort, "127.0.0.1", () => {
    const lines = Object.entries(headers(request)).flatMap(([key, value]) =>
      value === undefined
        ? []
        : [`${key}: ${Array.isArray(value) ? value.join(", ") : value}`],
    );
    upstream.write(
      `${request.method} ${request.url} HTTP/${request.httpVersion}\r\n${lines.join("\r\n")}\r\n\r\n`,
    );
    if (head.length) upstream.write(head);
    socket.pipe(upstream);
    upstream.pipe(socket);
  });
  upstream.on("error", () => socket.destroy());
  socket.on("error", () => upstream.destroy());
  socket.on("close", () => upstream.destroy());
});
proxy.on("error", (error) => {
  console.error(error.message);
  next.kill("SIGTERM");
  process.exitCode = 1;
});
proxy.listen(3000, "127.0.0.1", () =>
  console.log("Website ingress: http://127.0.0.1:3000"),
);
function stop() {
  proxy.close();
  next.kill("SIGTERM");
}
process.on("SIGINT", stop);
process.on("SIGTERM", stop);
next.on("exit", (code) => {
  proxy.close();
  process.exitCode = process.exitCode || code || 0;
});
