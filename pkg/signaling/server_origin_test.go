package signaling

import (
	"net/http/httptest"
	"testing"
)

func TestWebSocketOriginPolicy(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{name: "same host", origin: "https://app.example.com", host: "app.example.com", want: true},
		{name: "same host case insensitive", origin: "https://APP.example.com", host: "app.example.com", want: true},
		{name: "cross origin", origin: "https://attacker.example", host: "app.example.com", want: false},
		{name: "null origin", origin: "null", host: "app.example.com", want: false},
		{name: "malformed origin", origin: "://bad", host: "app.example.com", want: false},
		{name: "non-http scheme", origin: "javascript://app.example.com", host: "app.example.com", want: false},
		{name: "native client without origin", host: "app.example.com", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "https://"+tt.host+"/ws", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if got := upgrader.CheckOrigin(req); got != tt.want {
				t.Fatalf("CheckOrigin(%q, host %q) = %v, want %v", tt.origin, tt.host, got, tt.want)
			}
		})
	}
}
