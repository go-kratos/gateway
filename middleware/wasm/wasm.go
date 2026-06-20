package wasm

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func init() {
	middleware.Register("wasm", Middleware)
}

var (
	compiledModule wazero.CompiledModule
	runtime        wazero.Runtime
	mu             sync.Mutex
)

// WasmPath can be overwritten in tests
var WasmPath = "guest/guest.wasm"

type requestInfo struct {
	req     *http.Request
	blocked bool
}

func loadWasmModule(ctx context.Context, path string) error {
	mu.Lock()
	defer mu.Unlock()
	if compiledModule != nil {
		return nil
	}

	r := wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Define our ABI:
	// get_uri(ptr uint32, limit uint32) uint32
	// block_request()
	_, err := r.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, ptr uint32, limit uint32) uint32 {
			reqInfo := ctx.Value("req").(*requestInfo)
			uri := reqInfo.req.URL.RequestURI()
			if limit >= uint32(len(uri)) {
				m.Memory().Write(ptr, []byte(uri))
				return uint32(len(uri))
			}
			return 0
		}).Export("get_uri").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module) {
			reqInfo := ctx.Value("req").(*requestInfo)
			reqInfo.blocked = true
		}).Export("block_request").
		Instantiate(ctx)

	if err != nil {
		return err
	}

	wasmBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	compiledModule, err = r.CompileModule(ctx, wasmBytes)
	if err != nil {
		return err
	}
	runtime = r
	return nil
}

// Middleware executes a dynamic WebAssembly module to process requests.
func Middleware(c *config.Middleware) (middleware.Middleware, error) {
	return func(next http.RoundTripper) http.RoundTripper {
		return middleware.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			ctx := context.Background()
			
			err := loadWasmModule(ctx, WasmPath)
			if err != nil {
				panic(err)
			}

			info := &requestInfo{req: req, blocked: false}
			reqCtx := context.WithValue(ctx, "req", info)

			// Instantiate automatically executes _start -> main()
			mod, err := runtime.InstantiateModule(reqCtx, compiledModule, wazero.NewModuleConfig().WithName(""))
			if err != nil {
				// A clean exit in WASI throws an error with exit_code(0)
				if !strings.Contains(err.Error(), "exit_code(0)") {
					panic(err)
				}
			}
			if mod != nil {
				mod.Close(reqCtx)
			}

			// If the Wasm plugin flagged this as blocked, return 403 Forbidden
			if info.blocked {
				return &http.Response{
					Status:     http.StatusText(http.StatusForbidden),
					StatusCode: http.StatusForbidden,
					Header:     http.Header{},
					Body:       io.NopCloser(&bytes.Buffer{}),
				}, nil
			}

			return next.RoundTrip(req)
		})
	}, nil
}
