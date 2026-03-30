FROM golang:alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o af .

FROM golang:alpine AS seeder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o seed-tool ./seed/
RUN ./seed-tool -db /build/af.db -data /build/seed/data/

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /build/af .
COPY --from=seeder /build/af.db .
COPY public ./public
EXPOSE 3000
CMD ["./af"]
