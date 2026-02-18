FROM node:20-slim AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.24-bookworm AS backend
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /app/frontend/dist ./cmd/server/dist
RUN CGO_ENABLED=0 go build -o /slic ./cmd/server

FROM debian:bookworm-slim
COPY --from=backend /slic /slic
COPY backend/slic.db /slic.db
EXPOSE 8080
CMD ["/slic"]
