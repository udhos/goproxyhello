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

    goproxyhello
