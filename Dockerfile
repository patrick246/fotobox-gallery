FROM golang:1.17 as builder
WORKDIR /app
COPY go.* ./
RUN go mod download -x
COPY . .
RUN mkdir -p bin && CGO_ENABLED=0 go build -o bin ./cmd/...

FROM gcr.io/distroless/static:nonroot
LABEL org.opencontainers.image.source=https://github.com/patrick246/fotobox-gallery
LABEL org.opencontainers.image.authors=patrick246
LABEL org.opencontainers.image.licenses=AGPL-3.0
COPY --from=builder /app/bin/fotobox-gallery /
USER nonroot
ENTRYPOINT ["/fotobox-gallery"]
