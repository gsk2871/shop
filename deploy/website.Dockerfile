FROM node:22-bookworm-slim AS build
RUN npm install -g pnpm@9.15.0
WORKDIR /src
COPY shared/ shared/
COPY website/package.json website/pnpm-lock.yaml website/
WORKDIR /src/website
RUN pnpm install --frozen-lockfile
COPY website/ ./
ENV BACKEND_URL=http://backend:8080 NEXT_TELEMETRY_DISABLED=1
RUN pnpm build && mkdir -p .next/standalone/public \
    && if [ -d public ]; then cp -a public/. .next/standalone/public/; fi \
    && cp -a .next/static .next/standalone/.next/static
FROM node:22-bookworm-slim
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 HOSTNAME=0.0.0.0 PORT=3000 BACKEND_URL=http://backend:8080
COPY --from=build --chown=node:node /src/website/.next/standalone/ ./
USER node
CMD ["node", "server.js"]
