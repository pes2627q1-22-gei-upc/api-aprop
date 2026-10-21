# --- Build ---
FROM golang:1.25 AS build

WORKDIR /src

# Download dependencies first to take advantage of Docker's layer cache
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/mcp ./cmd/mcp

# --- Runtime ---
# distroless/static includes CA certificates (needed to call the gencat API over HTTPS)
# This part is the image that will create the containers in the server.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/api /app/api
# The catalogue MCP server runs from the same image: --entrypoint /app/mcp
COPY --from=build /out/mcp /app/mcp

EXPOSE 8080 8081

# Use a non root user to run the docker
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
