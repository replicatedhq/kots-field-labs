FROM golang:1.16 AS builder
COPY /go/src/ /go/src/
WORKDIR /go/src/
# RUN unset GOPATH
RUN go install /go/src/cmd/kots-field-labs/
# COPY app.go    .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kotslab /go/src/cmd/kots-field-labs/

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/cmd/kots-field-labs/ .
COPY container_scripts/create_labs.sh ./

# CMD ["./kotslab"]
ENTRYPOINT ["/bin/ash", "create_labs.sh"]