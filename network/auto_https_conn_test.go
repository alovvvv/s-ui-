package network

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// A plaintext request arriving on the TLS port must be answered with a redirect
// the client can actually read. The response used to be a zero-valued
// http.Response, which serialises as "HTTP/0.0 307" -- curl rejects that with
// "unsupported HTTP version", so the redirect never worked at all.
func TestPlainRequestGetsAReadableRedirect(t *testing.T) {
	client, server := net.Pipe()
	conn := NewAutoHttpsConn(server)

	go func() {
		_, _ = io.WriteString(client, "GET /app/ HTTP/1.1\r\nHost: panel.example:2095\r\n\r\n")
	}()

	done := make(chan *http.Response, 1)
	errc := make(chan error, 1)
	go func() {
		resp, err := http.ReadResponse(bufio.NewReader(client), nil)
		if err != nil {
			errc <- err
			return
		}
		done <- resp
	}()

	// Drives readRequest, which writes the redirect on the other side.
	buf := make([]byte, 1)
	_, _ = conn.Read(buf)

	select {
	case err := <-errc:
		t.Fatalf("the client could not read the response: %v", err)
	case resp := <-done:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusTemporaryRedirect {
			t.Errorf("status = %d, want 307", resp.StatusCode)
		}
		if resp.ProtoMajor != 1 || resp.ProtoMinor != 1 {
			t.Errorf("proto = HTTP/%d.%d, want HTTP/1.1", resp.ProtoMajor, resp.ProtoMinor)
		}
		if got, want := resp.Header.Get("Location"), "https://panel.example:2095/app/"; got != want {
			t.Errorf("Location = %q, want %q", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the redirect")
	}
}

// A TLS handshake is not an HTTP request: the bytes have to be handed straight
// through to the TLS listener, not consumed.
func TestTLSBytesArePassedThrough(t *testing.T) {
	client, server := net.Pipe()
	conn := NewAutoHttpsConn(server)

	// The opening bytes of a TLS ClientHello.
	hello := []byte{0x16, 0x03, 0x01, 0x00, 0x05, 0x01, 0x00, 0x00, 0x01, 0x00}
	go func() {
		_, _ = client.Write(hello)
	}()

	got := make([]byte, 0, len(hello))
	buf := make([]byte, len(hello))
	deadline := time.Now().Add(5 * time.Second)
	for len(got) < len(hello) && time.Now().Before(deadline) {
		n, err := conn.Read(buf)
		got = append(got, buf[:n]...)
		if err != nil {
			break
		}
	}
	if string(got) != string(hello) {
		t.Errorf("read %v, want the handshake bytes %v", got, hello)
	}
}

// A peer that connects and sends nothing used to pin a goroutine and a file
// descriptor for the life of the process.
func TestSilentPeerDoesNotBlockForever(t *testing.T) {
	if testing.Short() {
		t.Skip("waits on the first-read deadline")
	}
	client, server := net.Pipe()
	defer client.Close()
	conn := NewAutoHttpsConn(server)

	done := make(chan struct{})
	go func() {
		buf := make([]byte, 1)
		_, _ = conn.Read(buf)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(firstReadTimeout + 10*time.Second):
		t.Fatalf("the read did not give up within %v", firstReadTimeout)
	}
}
