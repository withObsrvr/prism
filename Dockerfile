# ──────────────────────────────────────────────
# Stage 1: Build
# ──────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git nodejs npm

# Install templ CLI
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1001

WORKDIR /src

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Generate templ files
RUN templ generate

# Build Tailwind CSS
RUN npm install tailwindcss @tailwindcss/cli && \
    npx @tailwindcss/cli -i web/static/css/input.css -o web/static/css/app.css --minify

# Build binary
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w \
      -X github.com/withObsrvr/prism/cmd/prism.version=${VERSION} \
      -X github.com/withObsrvr/prism/cmd/prism.commit=${COMMIT} \
      -X github.com/withObsrvr/prism/cmd/prism.date=${DATE}" \
    -o /bin/prism .

# ──────────────────────────────────────────────
# Stage 2: Runtime
# ──────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/prism /usr/local/bin/prism
COPY --from=builder /src/web/static /app/web/static

ENV PRISM_PORT=8080
EXPOSE 8080

ENTRYPOINT ["prism"]
CMD ["serve"]
