FROM golang:alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

RUN apk update --no-cache && apk add --no-cache make

WORKDIR /build

ADD go.mod .
ADD go.sum .

RUN go mod download

COPY . .
RUN make go


FROM --platform=linux/amd64 alpine

ENV TZ Asia/Shanghai

WORKDIR /app
COPY --from=builder /build/k8s-char-device-plugin /app/

ENTRYPOINT ["/app/k8s-char-device-plugin"]
CMD ["-config", "/app/config.yaml"]