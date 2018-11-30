package main

import (
	"flag"
	//"fmt"
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
	var disableKeepalive bool

	flag.StringVar(&key, "key", "key.pem", "TLS key file")
	flag.StringVar(&cert, "cert", "cert.pem", "TLS cert file")
	flag.StringVar(&listen, "listen", ":8080", "listen address")
	flag.StringVar(&target, "target", "http://localhost", "target address")
	flag.BoolVar(&disableKeepalive, "disableKeepalive", false, "disable keepalive")
	flag.Parse()

	keepalive := !disableKeepalive

	log.Print("keepalive: ", keepalive)

	if !fileExists(key) {
		log.Printf("TLS key file not found: %s - disabling TLS", key)
		tls = false
	}

	if !fileExists(cert) {
		log.Printf("TLS cert file not found: %s - disabling TLS", cert)
		tls = false
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { rootHandler(w, r, keepalive, target) })

	if tls {
		log.Printf("forwarding HTTPS from TCP %s to %s", listen, target)
		if err := listenAndServeTLS(listen, cert, key, nil, keepalive); err != nil {
			log.Fatalf("listenAndServeTLS: %s: %v", listen, err)
		}
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

func rootHandler(w http.ResponseWriter, r *http.Request, keepalive bool, target string) {
	log.Printf("%s host=%s path=%s query=%s from=%s to=%s", r.Method, r.Host, r.URL.Path, r.URL.RawQuery, r.RemoteAddr, target)

	showHeader("original request", r.Header)

	log.Printf("trying: %s %s %s %s", r.Method, target, r.URL.Path, r.URL.RawQuery)

	u := target + r.URL.Path + "?" + r.URL.RawQuery

	log.Printf("trying: %s", u)

	req, errReq := http.NewRequest(r.Method, u, r.Body)
	if errReq != nil {
		log.Printf("request error: %v", errReq)
		http.Error(w, errReq.Error(), http.StatusServiceUnavailable)
		return
	}
	//req.Header.Set("Content-Type", req.Header.Get("Content-Type"))
	copyHeader("Content-Type", req.Header, r.Header)
	copyHeader("Authorization", req.Header, r.Header)

	showHeader("forward request", r.Header)

	tls := strings.HasPrefix(target, "https://")

	c := httpClient(tls)

	resp, errDo := c.Do(req)
	if errDo != nil {
		log.Printf("call error: %v", errDo)
		http.Error(w, errDo.Error(), http.StatusServiceUnavailable)
		return
	}

	showHeader("response", resp.Header)

	copyHeaderAll(w.Header(), resp.Header) // copy headers

	log.Printf("status: %d", resp.StatusCode)
	w.WriteHeader(resp.StatusCode) // copy status

	io.Copy(w, resp.Body) // copy body

	resp.Body.Close()
}

func showHeader(label string, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			log.Printf("%s: %s: %s", label, k, v)
		}
	}
}

func copyHeader(key string, dst, src http.Header) {
	kk := strings.ToLower(key)
	for k, vv := range src {
		if kk != strings.ToLower(k) {
			continue
		}
		for _, v := range vv {
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
