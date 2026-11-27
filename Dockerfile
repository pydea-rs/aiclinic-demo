# =====================================================================
# AI-Clinic · Unified Render Production Dockerfile
# Builds both the React frontend and NestJS backend into a single
# self-contained service with an Nginx reverse-proxy on port 8080.
# =====================================================================

# ---------------------------------------------------------------------
# Stage 1: Build Frontend SPA (Vite React)
# ---------------------------------------------------------------------
FROM node:22-alpine AS client-build
WORKDIR /app/client

COPY client/package.json client/pnpm-lock.yaml client/pnpm-workspace.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile

COPY client/index.html client/vite.config.ts client/tsconfig*.json client/tailwind.config.js client/postcss.config.js ./
COPY client/src ./src

ARG VITE_APP_NAME=Ai-Clinic-Demo
ARG VITE_STRIP_AI_GREETING=true
ENV VITE_API_BASE_URL=/api
ENV VITE_WS_URL=
ENV VITE_APP_NAME=${VITE_APP_NAME}
ENV VITE_STRIP_AI_GREETING=${VITE_STRIP_AI_GREETING}

RUN pnpm run build

# ---------------------------------------------------------------------
# Stage 2: Build Backend API (NestJS + Fastify)
# ---------------------------------------------------------------------
FROM node:22-alpine AS server-build
WORKDIR /app/server

COPY server/package.json server/pnpm-lock.yaml server/pnpm-workspace.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile

COPY server/prisma ./prisma
RUN pnpm exec prisma generate

COPY server/tsconfig.json server/tsconfig.build.json server/nest-cli.json ./
COPY server/src ./src

RUN pnpm run build

# ---------------------------------------------------------------------
# Stage 3: Production Runtime (Node 22-slim + Nginx)
# ---------------------------------------------------------------------
FROM node:22-slim AS production

# Install nginx, openssl (required by Prisma), dumb-init, ca-certificates, and gettext-base
RUN apt-get update && apt-get install -y --no-install-recommends \
    dumb-init \
    openssl \
    ca-certificates \
    nginx \
    gettext-base \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app/server

# Install production dependencies for server
COPY server/package.json server/pnpm-lock.yaml server/pnpm-workspace.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile --prod

# Copy Prisma schema and generated client
COPY server/prisma ./prisma
RUN pnpm exec prisma generate

# Copy built server distribution
COPY --from=server-build /app/server/dist ./dist

# Copy built client distribution into Nginx webroot
COPY --from=client-build /app/client/dist /usr/share/nginx/html

# Create uploads folder
RUN mkdir -p /app/server/uploads

# Configure Nginx template supporting dynamic PORT, WebSockets, and unbuffered SSE
RUN mkdir -p /etc/nginx/templates

RUN cat << 'EOF' > /etc/nginx/templates/default.conf.template
server {
    listen ${PORT};
    server_name localhost;
    root /usr/share/nginx/html;
    index index.html;

    # Client-side SPA routing fallback
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Reverse proxy API requests to local NestJS backend on port 3000
    location /api/ {
        client_max_body_size 11m;
        proxy_pass http://127.0.0.1:3000/;
        proxy_http_version 1.1;

        # WebSocket support
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Standard proxy headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Keep WebSockets and SSE alive (prevent 60s idle timeout)
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;

        # Real-time streaming for SSE without intermediate buffering
        proxy_buffering off;
        proxy_cache off;
    }

    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml;
    gzip_min_length 256;
}
EOF

# Create startup script
RUN cat << 'EOF' > /app/start.sh
#!/bin/sh
set -e

# Default port to 8080 if PORT is not set by the platform
export PORT="${PORT:-8080}"
echo "==> Configuring Nginx on port ${PORT}..."
mkdir -p /etc/nginx/conf.d
envsubst '${PORT}' < /etc/nginx/templates/default.conf.template > /etc/nginx/conf.d/default.conf

echo "==> Running Prisma Database Migrations against Neon..."
cd /app/server
npx prisma migrate deploy

if [ "$SEED_DATABASE" = "true" ]; then
    echo "==> Seeding initial platform data..."
    npx prisma db seed
fi

echo "==> Starting NestJS Backend on internal port 3000..."
APP_PORT=3000 node dist/main &

echo "==> Starting Nginx Reverse Proxy on external port ${PORT}..."
exec nginx -g "daemon off;"
EOF

RUN chmod +x /app/start.sh

EXPOSE 8080

ENTRYPOINT ["dumb-init", "--"]
CMD ["/app/start.sh"]
