# syntax=docker/dockerfile:1

FROM node:24-alpine AS web
ARG NPM_REGISTRY=https://registry.npmmirror.com
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm install --no-audit --no-fund --registry=${NPM_REGISTRY}
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend
ARG GOPROXY=https://goproxy.cn,direct
RUN apk add --no-cache protobuf protobuf-dev ca-certificates
ENV CGO_ENABLED=0
ENV GOPROXY=${GOPROXY}
WORKDIR /src
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download
COPY proto ./proto
COPY backend ./backend
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11 \
 && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1
RUN protoc -I proto --go_out=backend --go_opt=module=backend \
      --go-grpc_out=backend --go-grpc_opt=module=backend \
      proto/host/v1/host.proto

FROM backend AS server-build
RUN rm -rf backend/internal/server/webdist && mkdir -p backend/internal/server/webdist
COPY --from=web /src/frontend/dist ./backend/internal/server/webdist
RUN cd backend && go build -trimpath -ldflags "-s -w" -o /out/omnitask-server ./cmd/server

FROM backend AS host-build
RUN cd backend && go build -trimpath -ldflags "-s -w" -o /out/omnitask-host ./cmd/host

FROM python:3.12-slim AS runtime
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tzdata git build-essential \
 && rm -rf /var/lib/apt/lists/*
ENV TZ=Asia/Shanghai
WORKDIR /app

FROM runtime AS server
COPY deploy/docker-compose/entrypoint.sh /app/entrypoint.sh
COPY --from=server-build /out/omnitask-server /app/omnitask-server
RUN chmod +x /app/entrypoint.sh
EXPOSE 8080 9000
ENTRYPOINT ["/app/entrypoint.sh", "/config-template/config.yml", "/app/omnitask-server"]

FROM runtime AS host
COPY deploy/docker-compose/entrypoint.sh /app/entrypoint.sh
COPY --from=host-build /out/omnitask-host /app/omnitask-host
COPY sdk/python /sdk
RUN chmod +x /app/entrypoint.sh
ENV PYTHONPATH=/sdk
VOLUME ["/data"]
EXPOSE 50051
ENTRYPOINT ["/app/entrypoint.sh", "/config-template/config.yml", "/app/omnitask-host"]
