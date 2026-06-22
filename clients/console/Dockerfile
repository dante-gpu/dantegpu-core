# Multi-stage build for the DanteGPU console. Vite bakes VITE_* env at build
# time, so the gateway origin is passed as a build arg. The result is static
# assets served by nginx with SPA fallback.

FROM node:22-alpine AS build
WORKDIR /app

# Install deps first for better layer caching.
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY . .

# Gateway origin baked into the bundle. Override at build time:
#   docker build --build-arg VITE_API_BASE_URL=https://api.dantegpu.com .
ARG VITE_API_BASE_URL=http://localhost:8080
ARG VITE_SOLANA_CLUSTER=devnet
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_SOLANA_CLUSTER=$VITE_SOLANA_CLUSTER
RUN npm run build

FROM nginx:1.27-alpine AS runtime
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://localhost/ >/dev/null 2>&1 || exit 1
CMD ["nginx", "-g", "daemon off;"]
