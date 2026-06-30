# Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/client
COPY client/package.json client/package-lock.json ./
RUN npm ci
COPY client/ .
RUN npm run build

# Build Go binary
FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY server/ ./server/
COPY --from=frontend /app/client/dist ./server/cmd/client/dist/
WORKDIR /app/server
RUN go build -o /app/davechat ./cmd/main.go

# Runtime
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=backend /app/davechat .
EXPOSE 8080
CMD ["./davechat"]
