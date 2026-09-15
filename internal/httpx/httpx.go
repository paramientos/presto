package httpx

import (
	"net"
	"net/http"
	"time"
)

// New builds a client whose connection pool is sized for the caller's worker
// count. The default of 2 idle connections per host throttles parallel fetches.
func New(timeout time.Duration, connsPerHost int) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          connsPerHost * 4,
		MaxIdleConnsPerHost:   connsPerHost,
		MaxConnsPerHost:       connsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		ForceAttemptHTTP2:     true,
	}

	return &http.Client{Timeout: timeout, Transport: transport}
}
