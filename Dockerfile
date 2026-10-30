FROM node:24-alpine AS web
WORKDIR /src/webapp
COPY webapp/package*.json ./
RUN npm ci
COPY webapp/ ./
COPY cmd/vpet/static/index.html /src/cmd/vpet/static/index.html
RUN npm run build

FROM golang:1.25.9-alpine AS go
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=web /src/cmd/vpet/static/main.js ./cmd/vpet/static/main.js
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/vpet ./cmd/vpet
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/vpet-migrate ./cmd/vpet-migrate

FROM alpine:3.22
RUN addgroup -S vpet && adduser -S -G vpet vpet
COPY --from=go /out/vpet /usr/local/bin/vpet
COPY --from=go /out/vpet-migrate /usr/local/bin/vpet-migrate
USER vpet
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/vpet"]
