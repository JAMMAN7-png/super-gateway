package proxy

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

// DialerConfig controls HTTP client settings and proxy
type DialerConfig struct {
	ProxyURL  string
	Timeout   time.Duration
	KeepAlive time.Duration
}

// NewFastHTTPClient creates a fasthttp client with optional SOCKS5/HTTP proxy
func NewFastHTTPClient(cfg DialerConfig) *fasthttp.Client {
	c := &fasthttp.Client{
		ReadTimeout:         cfg.Timeout,
		WriteTimeout:        cfg.Timeout,
		MaxIdleConnDuration: cfg.KeepAlive,
		TLSConfig:           &tls.Config{InsecureSkipVerify: false},
	}

	if cfg.ProxyURL != "" {
		u, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			return c
		}
		switch u.Scheme {
		case "socks5", "socks5h":
			c.Dial = fasthttpproxy.FasthttpSocksDialer(cfg.ProxyURL)
		case "http", "https":
			c.Dial = fasthttpproxy.FasthttpHTTPDialer(cfg.ProxyURL)
		}
	}
	return c
}

// NewStandardHTTPClient creates a standard net/http client with proxy
func NewStandardHTTPClient(cfg DialerConfig) *http.Client {
	t := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     cfg.KeepAlive,
	}

	if cfg.ProxyURL != "" {
		if u, err := url.Parse(cfg.ProxyURL); err == nil {
			t.Proxy = http.ProxyURL(u)
		}
	}

	return &http.Client{Transport: t, Timeout: cfg.Timeout}
}
