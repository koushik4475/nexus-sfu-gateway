package signaling

import (
	"net/http/httptest"
	"testing"
)

func TestCheckSameOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{name: "no origin for native client", host: "gateway.example:8080", want: true},
		{name: "matching origin", origin: "https://gateway.example:8080", host: "gateway.example:8080", want: true},
		{name: "different host", origin: "https://attacker.example", host: "gateway.example", want: false},
		{name: "different port", origin: "https://gateway.example:9000", host: "gateway.example:8080", want: false},
		{name: "invalid origin", origin: "not a url", host: "gateway.example", want: false},
		{name: "unsupported scheme", origin: "javascript://gateway.example", host: "gateway.example", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://"+tt.host+"/ws", nil)
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if got := checkSameOrigin(r); got != tt.want {
				t.Fatalf("checkSameOrigin() = %v, want %v", got, tt.want)
			}
		})
	}
}
