# goproxyhello

Simple Go HTTP proxy

# Quick Start

## Build

    # make sure you have Go 1.11 or higher
    git clone https://github.com/udhos/goproxyhello ;# clone outside of GOPATH
    cd goproxyhello
    go install ./goproxyhello

## Run

If you want to use HTTPS, you will need a certificate:

    $ openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout key.pem -out cert.pem

Run:

    goproxyhello -target http://remote_host:remote_port

Example:

    $ goproxyhello -target http://localhost:8000
    2018/11/30 17:22:05 version=0.0 runtime=go1.11.2 GOMAXPROCS=1
    2018/11/30 17:22:05 keepalive: true
    2018/11/30 17:22:05 TLS key file not found: key.pem - disabling TLS
    2018/11/30 17:22:05 TLS cert file not found: cert.pem - disabling TLS
    2018/11/30 17:22:05 forwarding HTTP from TCP :8080 to http://localhost:8000

