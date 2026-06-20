package wasm

import (
	"net/http"
	"os"
	"os/exec"
	"testing"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
)

func TestWasmMiddleware(t *testing.T) {
	// 1. Compile the guest Wasm module
	cmd := exec.Command("go", "build", "-o", "guest.wasm", "guest.go")
	cmd.Dir = "guest"
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to compile guest.wasm: %v\n%s", err, string(out))
	}
	defer os.Remove("guest/guest.wasm")

	// 2. Setup Middleware
	WasmPath = "guest/guest.wasm"
	c := &config.Middleware{Name: "wasm"}
	m, err := Middleware(c)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	next := middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	})
	tripper := m(next)

	// 3. Test Allowed Request
	reqAllowed, _ := http.NewRequest("GET", "http://example.com/api/users", nil)
	respAllowed, _ := tripper.RoundTrip(reqAllowed)
	if respAllowed.StatusCode != http.StatusOK {
		t.Fatalf("expected /api/users to be allowed (200), got %d", respAllowed.StatusCode)
	}

	// 4. Test Blocked Request (triggered by Wasm module)
	reqBlocked, _ := http.NewRequest("GET", "http://example.com/forbidden", nil)
	respBlocked, _ := tripper.RoundTrip(reqBlocked)
	if respBlocked.StatusCode != http.StatusForbidden {
		t.Fatalf("expected /forbidden to be blocked (403), got %d", respBlocked.StatusCode)
	}
}
