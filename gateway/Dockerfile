FROM golang:alpine3.18 AS compiler
RUN apk --update --no-cache add git
WORKDIR /gateway

ADD go.mod go.sum ./
RUN go mod download

ADD . .
RUN CGO_ENABLED=0 go build -o /bin/goapp ./main.go

FROM scratch
COPY --from=compiler /bin/goapp /gateway
ENTRYPOINT ["/gateway"]
