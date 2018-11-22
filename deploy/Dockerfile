FROM golang:1-alpine as builder
RUN mkdir /build
ENV BUILD_FROM="/go/src/github.com/sintell/lisariy-server/cmd/lisariy-webapp/main.go"
ADD ./cmd /go/src/github.com/sintell/lisariy-server/cmd
ADD ./internal /go/src/github.com/sintell/lisariy-server/internal
ADD ./vendor /go/src/github.com/sintell/lisariy-server/vendor
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /build/main ${BUILD_FROM}

FROM codechimpio/vips-alpine

RUN adduser -S -D -H -h /app lisariy
RUN mkdir /var/log/lisariy-webapp
RUN mkdir -p /etc/lisariy-webapp/config
RUN mkdir -p /var/lib/lisariy-webapp/images
RUN chown lisariy /var/log/lisariy-webapp
USER lisariy
COPY --from=builder /build/main /app/
WORKDIR /app
CMD ["./main", "-config", "/etc/lisariy-webapp/config/config.json"]
