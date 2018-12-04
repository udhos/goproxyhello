package main

import (
	"flag"
	//"fmt"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const (
	helloVersion = "0.0"
)

func main() {

	tls := true

	log.Printf("version=%s runtime=%s GOMAXPROCS=%d", helloVersion, runtime.Version(), runtime.GOMAXPROCS(0))

	var target, listen, key, cert string
	var disableKeepalive, disableTLS bool

	flag.StringVar(&key, "key", "key.pem", "TLS key file")
	flag.StringVar(&cert, "cert", "cert.pem", "TLS cert file")
	flag.StringVar(&listen, "listen", ":8080", "listen address")
	flag.StringVar(&target, "target", "http://localhost", "target address")
	flag.BoolVar(&disableKeepalive, "disableKeepalive", false, "disable keepalive, only on listener")
	flag.BoolVar(&disableTLS, "disableTLS", false, "disable TLS")
	flag.Parse()

	hostname, errHost := os.Hostname()
	if errHost != nil {
		hostname = "unknown-host"
		log.Printf("failure finding hostname: %v", errHost)
	}
	log.Printf("hostname: %s", hostname)

	keepalive := !disableKeepalive
	log.Print("keepalive: ", keepalive)

	if disableTLS {
		log.Printf("disabling TLS from command-line switch: -disableTLS")
		tls = false
	} else {
		if !fileExists(key) {
			log.Printf("TLS key file not found: %s - disabling TLS", key)
			tls = false
		}
		if !fileExists(cert) {
			log.Printf("TLS cert file not found: %s - disabling TLS", cert)
			tls = false
		}
	}

	headers := map[string]struct{}{}
	headers["authorization"] = struct{}{}
	headers["content-type"] = struct{}{}
	headers["accept"] = struct{}{}
	headers["expect"] = struct{}{}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { rootHandler(w, r, target, hostname, headers) })

	if tls {
		log.Printf("forwarding HTTPS from TCP %s to %s", listen, target)
		if err := listenAndServeTLS(listen, cert, key, nil, keepalive); err != nil {
			log.Fatalf("listenAndServeTLS: %s: %v", listen, err)
		}
		return
	}

	log.Printf("forwarding HTTP from TCP %s to %s", listen, target)
	if err := listenAndServe(listen, nil, keepalive); err != nil {
		log.Fatalf("listenAndServe: %s: %v", listen, err)
	}
}

func listenAndServe(addr string, handler http.Handler, keepalive bool) error {
	server := &http.Server{Addr: addr, Handler: handler}
	server.SetKeepAlivesEnabled(keepalive)
	return server.ListenAndServe()
}

func listenAndServeTLS(addr, certFile, keyFile string, handler http.Handler, keepalive bool) error {
	server := &http.Server{Addr: addr, Handler: handler}
	server.SetKeepAlivesEnabled(keepalive)
	return server.ListenAndServeTLS(certFile, keyFile)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func rootHandler(w http.ResponseWriter, r *http.Request, target, hostname string, headers map[string]struct{}) {
	log.Printf("BEGIN TLS=%v %s host=%s path=%s query=%s from=%s to=%s", r.TLS != nil, r.Method, r.Host, r.URL.Path, r.URL.RawQuery, r.RemoteAddr, target)
	work(w, r, target, hostname, headers)
	log.Printf("END TLS=%v %s host=%s path=%s query=%s from=%s to=%s", r.TLS != nil, r.Method, r.Host, r.URL.Path, r.URL.RawQuery, r.RemoteAddr, target)
}

type readAccount struct {
	reader io.Reader
	size   int64
	err    error
}

func (r *readAccount) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	//log.Printf("read: %d %v", n, err)
	r.size += int64(n)
	r.err = err
	return
}

func work(w http.ResponseWriter, r *http.Request, target, hostname string, headers map[string]struct{}) {

	showHeader("original request", r.Header)

	via := "1.1 " + hostname

	foundVia := findHeader(r.Header, "Via", via)
	if foundVia {
		log.Printf("request loop found: %s: %s", "Via", via)
		http.Error(w, "loop detected from via header", http.StatusLoopDetected)
		return
	}

	tls := strings.HasPrefix(strings.ToLower(target), "https://")

	log.Printf("trying: TLS=%v %s %s %s %s", tls, r.Method, target, r.URL.Path, r.URL.RawQuery)

	u := target + r.URL.Path + "?" + r.URL.RawQuery

	log.Printf("trying: TLS=%v %s", tls, u)

	bodyReader := &readAccount{reader: r.Body}

	req, errReq := http.NewRequest(r.Method, u, bodyReader)
	if errReq != nil {
		log.Printf("request error: %v", errReq)
		http.Error(w, errReq.Error(), http.StatusServiceUnavailable)
		return
	}

	log.Printf("request: TLS=%v %s %s %s %s %s", tls, req.Method, req.URL.Scheme, req.Host, req.URL.Path, req.URL.RawQuery)

	log.Printf("%s request body: size=%d error: %v", r.Method, bodyReader.size, bodyReader.err)

	copyHeader("orig-to-forward", headers, req.Header, r.Header)

	if !foundVia {
		log.Printf("setting header: %s: %s", "Via", via)
		req.Header.Add("Via", via)
	}

	showHeader("forward request", req.Header)

	c := httpClient(tls)

	resp, errDo := c.Do(req)
	if errDo != nil {
		log.Printf("call error: %v", errDo)
		http.Error(w, errDo.Error(), http.StatusServiceUnavailable)
		return
	}

	showHeader("response", resp.Header)

	copyHeaderAll(w.Header(), resp.Header) // copy headers

	log.Printf("response status: %d", resp.StatusCode)
	w.WriteHeader(resp.StatusCode) // copy status

	n, errCopy := io.Copy(w, resp.Body) // copy body

	log.Printf("response body: size=%d error: %v", n, errCopy)

	resp.Body.Close()
}

func findHeader(h http.Header, key, value string) bool {
	lowK := strings.ToLower(key)
	lowV := strings.ToLower(value)
	for k, vv := range h {
		if lowK != strings.ToLower(k) {
			continue
		}
		for _, v := range vv {
			if lowV == strings.ToLower(v) {
				return true
			}
		}
	}
	return false
}

func showHeader(label string, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			log.Printf("%s: header: %s: %s", label, k, v)
		}
	}
}

func copyHeader(label string, keys map[string]struct{}, dst, src http.Header) {
	for k, vv := range src {
		lowK := strings.ToLower(k)
		if _, found := keys[lowK]; !found {
			continue
		}
		for _, v := range vv {
			log.Printf("%s: copy header: %s: %s", label, k, v)
			dst.Add(k, v)
		}
	}
}

func copyHeaderAll(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

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
	/*
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
	*/
	tr := &http.Transport{}
	if tls {
		tr.TLSClientConfig = tlsConfig()
	}
	return &http.Client{
		Transport: tr,
		//Timeout:   15 * time.Second,
	}
}
