FROM golang:1.25-alpine AS builder

ARG VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/ostretsov/labelsrv/internal/version.Version=${VERSION}" \
    -o /labelsrv ./cmd/labelsrv


FROM scratch

ARG VERSION=dev
ARG CREATED
ARG SOURCE=https://github.com/ostretsov/labelsrv

LABEL org.opencontainers.image.title="labelsrv" \
      org.opencontainers.image.description="Configuration-driven label rendering server" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${CREATED}" \
      org.opencontainers.image.source="${SOURCE}"

COPY --from=builder /labelsrv /labelsrv

VOLUME ["/labels", "/fonts"]

EXPOSE 8080

ENTRYPOINT ["/labelsrv"]
CMD ["serve", "--labels", "/labels", "--fonts", "/fonts"]
