# syntax=docker/dockerfile:1

# ── Stage 1: Build frontend assets ───────────────────────────────────────
FROM node:22-alpine AS assets

WORKDIR /app


# Copy only dependency files first → best layer caching
COPY nodejs/package*.json ./nodejs/

RUN cd nodejs \
  --mount=type=cache,target=/root/.npm && \
  npm ci --prefer-offline --no-audit --no-fund

#  copy the rest of nodejs dir files
COPY ./nodejs/*.js /app/nodejs/
COPY ./nodejs/src /app/nodejs/src
COPY ./static /app/static
COPY ./web /app/web

# Assuming your build script outputs to ./dist or similar
RUN cd nodejs && npm run build:webpack


# ── Stage 2: Build Go binary ─────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

COPY app/    ./app/
COPY types/  ./types/
COPY web/    ./web/
COPY main.go ./

# Copy already-built static assets
COPY --from=assets /app/static/ ./static/

RUN mkdir -p ./dist

RUN go tool templ generate

# Prefer static build if possible
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o ./dist/mandok ./main.go


# ── Stage 3: Minimal runtime ─────────────────────────────────────────────
# Use the Docker CLI image to run the docker commands
FROM docker:cli

# If using scratch + your app makes HTTPS calls → need certificates
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

COPY --from=builder /app/dist/mandok  ./mandok
# COPY --from=builder /app/static       ./static

# Optional: non-root user (scratch supports numeric uids)
# USER 10001:10001

ENV PORT=80
EXPOSE 80

VOLUME /app/projects

CMD ["./mandok"]