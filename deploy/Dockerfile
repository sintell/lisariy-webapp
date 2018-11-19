FROM golang:1-alpine as building
RUN mkdir build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o main .

FROM alpine
RUN adduser -S -D -H -h /app lisariy
USER lisariy
COPY --from=builder /build/main /app/
WORKDIR /app
CMD ["./main"]
