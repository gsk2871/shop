FROM node:22-bookworm-slim AS build
RUN npm install -g pnpm@9.15.0
WORKDIR /src
COPY shared/ shared/
COPY admin-fe/package.json admin-fe/pnpm-lock.yaml admin-fe/
WORKDIR /src/admin-fe
RUN pnpm install --frozen-lockfile
COPY admin-fe/ ./
ARG WEBSITE_URL
ENV UMI_APP_WEBSITE_URL=$WEBSITE_URL
RUN test -n "$WEBSITE_URL" && pnpm build
FROM nginx:stable-alpine
COPY --from=build /src/admin-fe/dist/ /usr/share/nginx/html/
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
