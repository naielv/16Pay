# =====================================================
# Stage 1: Build SvelteKit frontend
# =====================================================
FROM node:22-alpine AS frontend-builder
WORKDIR /app

# Install dependencies (cached layer)
COPY package.json package-lock.json ./
RUN npm ci

# Build Paraglide first (needed by vite build)
COPY project.inlang ./project.inlang
COPY messages ./messages
RUN npx @inlang/paraglide-js compile --project ./project.inlang --outdir ./src/lib/paraglide || true

# Copy sources and build
COPY . .
RUN npm run build

# =====================================================
# Stage 2: Build PocketBase Go binary
# =====================================================
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Cache Go dependencies
COPY pocketbase/go.mod pocketbase/go.sum ./
RUN go mod download

# Copy migrations and source
COPY pocketbase/ ./

# Build a static binary
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o 16pay-pocketbase .

# =====================================================
# Stage 3: Runtime image
# =====================================================
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the PocketBase binary
COPY --from=backend-builder /app/16pay-pocketbase /app/16pay-pocketbase

# Copy the pre-built frontend from stage 1
COPY --from=frontend-builder /app/pocketbase/pb_public /app/pb_public

# Default environment variables
ENV PAY_PUBLIC_DIR=/app/pb_public
ENV PB_DATA_DIR=/app/pb_data

# Expose PocketBase default port
EXPOSE 8090

# Healthcheck
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8090/api/health || exit 1

# Run PocketBase
CMD ["/app/16pay-pocketbase", "serve", "--http=0.0.0.0:8090", "--dir=/app/pb_data"]