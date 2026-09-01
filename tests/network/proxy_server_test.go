package network_test

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/pi-sandbox/pi/pkg/network"
)

func permissiveResolver(_ string) (*network.Policy, error) {
	return &network.Policy{Mode: network.ModeOpen}, nil // no DenyList: loopback reachable
}

func denyAllResolver(_ string) (*network.Policy, error) {
	return &network.Policy{Mode: network.ModeNone}, nil
}

func proxyClient(t *testing.T, proxyAddr, sandboxID string) *http.Client {
	t.Helper()
	u, err := url.Parse("http://" + sandboxID + ":tok@" + proxyAddr)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{Proxy: http.ProxyURL(u)},
	}
}

func TestProxyServer_AllowedHost_ForwardsRequest(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "hello from upstream")
	}))
	defer backend.Close()

	proxy := httptest.NewServer(network.NewProxyServer(permissiveResolver, nil))
	defer proxy.Close()

	resp, err := proxyClient(t, proxy.Listener.Addr().String(), "sbx-1").Get(backend.URL)
	if err != nil {
		t.Fatalf("proxied GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "hello from upstream" {
		t.Fatalf("got %d %q", resp.StatusCode, body)
	}
}

func TestProxyServer_InjectsCredentialsIntoApprovedForward(t *testing.T) {
	var gotAuth string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer backend.Close()

	proxy := network.NewProxyServer(permissiveResolver, nil)
	proxy.SetCredentialInjector(func(host string) []network.HeaderInjection {
		return []network.HeaderInjection{{Name: "Authorization", Value: "Basic injected"}}
	})
	ps := httptest.NewServer(proxy)
	defer ps.Close()

	// Client sends its own Authorization — the proxy must overwrite it.
	req, _ := http.NewRequest("GET", backend.URL, nil)
	req.Header.Set("Authorization", "Basic smuggled")
	resp, err := proxyClientDo(t, ps.Listener.Addr().String(), "sbx-1", req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "Basic injected" {
		t.Fatalf("upstream Authorization = %q, want proxy-injected value", gotAuth)
	}
}

func TestProxyServer_NoInjector_NoAuthAdded(t *testing.T) {
	var gotAuth string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer backend.Close()

	ps := httptest.NewServer(network.NewProxyServer(permissiveResolver, nil))
	defer ps.Close()

	resp, err := proxyClient(t, ps.Listener.Addr().String(), "sbx-1").Get(backend.URL)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if gotAuth != "" {
		t.Fatalf("unexpected Authorization %q with no injector", gotAuth)
	}
}

func proxyClientDo(t *testing.T, proxyAddr, sandboxID string, req *http.Request) (*http.Response, error) {
	t.Helper()
	return proxyClient(t, proxyAddr, sandboxID).Do(req)
}

func TestProxyServer_DeniedHost_403(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer backend.Close()

	proxy := httptest.NewServer(network.NewProxyServer(denyAllResolver, nil))
	defer proxy.Close()

	resp, err := proxyClient(t, proxy.Listener.Addr().String(), "sbx-1").Get(backend.URL)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got %d", resp.StatusCode)
	}
}

func TestProxyServer_UnknownSandbox_403(t *testing.T) {
	resolver := func(_ string) (*network.Policy, error) { return nil, fmt.Errorf("not found") }
	proxy := httptest.NewServer(network.NewProxyServer(resolver, nil))
	defer proxy.Close()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer backend.Close()

	resp, err := proxyClient(t, proxy.Listener.Addr().String(), "ghost").Get(backend.URL)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got %d", resp.StatusCode)
	}
}

func TestProxyServer_MissingProxyAuth_407(t *testing.T) {
	proxy := httptest.NewServer(network.NewProxyServer(permissiveResolver, nil))
	defer proxy.Close()

	// Raw absolute-URI proxy request with no Proxy-Authorization header.
	c, err := net.Dial("tcp", proxy.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	fmt.Fprint(c, "GET http://example.com/ HTTP/1.1\r\nHost: example.com\r\n\r\n")
	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusProxyAuthRequired {
		t.Fatalf("want 407, got %d", resp.StatusCode)
	}
}

func TestProxyServer_ConnectAllowed_Tunnels(t *testing.T) {
	// Raw TCP echo upstream.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		io.Copy(conn, conn)
	}()

	proxy := httptest.NewServer(network.NewProxyServer(permissiveResolver, nil))
	defer proxy.Close()

	c, err := net.Dial("tcp", proxy.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	auth := base64.StdEncoding.EncodeToString([]byte("sbx-1:tok"))
	fmt.Fprintf(c, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: Basic %s\r\n\r\n",
		ln.Addr(), ln.Addr(), auth)

	br := bufio.NewReader(c)
	status, err := br.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if want := "200"; !contains(status, want) {
		t.Fatalf("CONNECT status = %q, want 200", status)
	}
	// Consume the rest of the response headers (blank line).
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "\r\n" {
			break
		}
	}

	fmt.Fprint(c, "ping")
	buf := make([]byte, 4)
	if _, err := io.ReadFull(br, buf); err != nil {
		t.Fatalf("tunnel read: %v", err)
	}
	if string(buf) != "ping" {
		t.Fatalf("tunnel echo = %q", buf)
	}
}

func TestProxyServer_ConnectDeniedHost_403(t *testing.T) {
	proxy := httptest.NewServer(network.NewProxyServer(denyAllResolver, nil))
	defer proxy.Close()

	c, err := net.Dial("tcp", proxy.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	auth := base64.StdEncoding.EncodeToString([]byte("sbx-1:tok"))
	fmt.Fprintf(c, "CONNECT example.com:443 HTTP/1.1\r\nHost: example.com:443\r\nProxy-Authorization: Basic %s\r\n\r\n", auth)

	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got %d", resp.StatusCode)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
