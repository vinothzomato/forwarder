########################################
## Build Stage
########################################
FROM golang:1.14.9-alpine as builder

# add a label to clean up later
LABEL stage=intermediate

# install required packages
RUN apk add --no-cache git tzdata

# setup the working directory
WORKDIR /go/src

# add source code
ADD . .

# build the source
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(./version.sh)" -o forwarder-linux-amd64

########################################
## Production Stage
########################################
FROM scratch

# set working directory
WORKDIR /root

# copy required files from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /go/src/forwarder-linux-amd64 ./forwarder-linux-amd64

ENTRYPOINT ["./forwarder-linux-amd64"]