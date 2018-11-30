package main

import (
	//"bytes"
	"crypto/tls"
	//"fmt"
	//"io"
	//"io/ioutil"
	"net"
	"net/http"
	"time"
)

func tlsConfig() *tls.Config {
	return &tls.Config{
		//CipherSuites:             []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA, tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA},
		PreferServerCipherSuites: true,
		InsecureSkipVerify:       true,
		//MaxVersion:               tls.VersionTLS11,
		//MinVersion:               tls.VersionTLS11,
	}
}

func httpClient(tls bool) *http.Client {
	tr := &http.Transport{
		//TLSClientConfig:    tlsConfig(),
		DisableCompression: true,
		DisableKeepAlives:  true,
		Dial: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 10 * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if tls {
		tr.TLSClientConfig = tlsConfig()
	}
	return &http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
	}
}
