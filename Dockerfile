# syntax=docker/dockerfile:1
# Frontend build
FROM node:24.13.0-alpine AS frontendbuilder
WORKDIR /build
ENV PNPM_CACHE_FOLDER=.cache/pnpm/
ENV PUPPETEER_SKIP_DOWNLOAD=true
ENV CYPRESS_INSTALL_BINARY=0
COPY frontend/pnpm-lock.yaml frontend/package.json frontend/.npmrc ./
RUN npm install -g corepack && corepack enable && pnpm install --frozen-lockfile
COPY frontend/ ./
RUN echo '{"VERSION":"local"}' > src/version.json && pnpm run build

# Backend build
FROM golang:1.25-alpine AS apibuilder
RUN apk add --no-cache git build-base sqlite-dev
WORKDIR /go/src/task-board
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=frontendbuilder /build/dist ./frontend/dist
RUN go build -ldflags '-s -w' -o /task-board .

# Final minimal image
FROM alpine:3.21
WORKDIR /app/task-board
RUN apk add --no-cache ca-certificates tzdata && mkdir -p /app/task-board/files /db
COPY --from=apibuilder /task-board /app/task-board/task-board
ENV TASKBOARD_SERVICE_ROOTPATH=/app/task-board/
ENV TASKBOARD_DATABASE_PATH=/db/task-board.db
EXPOSE 3456
ENTRYPOINT ["/app/task-board/task-board"]
