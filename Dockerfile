# --- Build ---
FROM golang:1.25 AS build

WORKDIR /src

# Download dependencies first to take advantage of Docker's layer cache
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/app

# --- Runtime ---
# distroless/static includes CA certificates (needed to call the gencat API over HTTPS)
# This part is the image that will create the containers in the server.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/api /app/api

EXPOSE 8080

# Use a non root user to run the docker
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
