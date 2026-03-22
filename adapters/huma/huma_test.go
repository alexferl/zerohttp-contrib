package huma

import (
	"bytes"
	"context"
	"crypto/tls"
	"embed"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContext(t *testing.T) {
	op := &huma.Operation{
		Method: http.MethodGet,
		Path:   "/test/{id}",
	}

	t.Run("creates context with operation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test/123", nil)
		rec := httptest.NewRecorder()

		ctx := NewContext(op, req, rec)
		require.NotNil(t, ctx)

		assert.Equal(t, op, ctx.Operation())
		assert.Equal(t, http.MethodGet, ctx.Method())
	})
}

func TestZerohttpContext_Operation(t *testing.T) {
	op := &huma.Operation{Method: http.MethodPost, Path: "/api"}
	ctx := &zerohttpContext{op: op}

	assert.Equal(t, op, ctx.Operation())
}

func TestZerohttpContext_Context(t *testing.T) {
	baseCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(baseCtx)
	ctx := &zerohttpContext{r: req}

	assert.Equal(t, baseCtx, ctx.Context())
}

func TestZerohttpContext_Method(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx := &zerohttpContext{r: req}

	assert.Equal(t, http.MethodPost, ctx.Method())
}

func TestZerohttpContext_Host(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
	ctx := &zerohttpContext{r: req}

	assert.Equal(t, "example.com", ctx.Host())
}

func TestZerohttpContext_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	ctx := &zerohttpContext{r: req}

	assert.Equal(t, "192.168.1.1:1234", ctx.RemoteAddr())
}

func TestZerohttpContext_URL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com:8080/path?key=value", nil)
	ctx := &zerohttpContext{r: req}

	url := ctx.URL()
	assert.Equal(t, "/path", url.Path)
	assert.Equal(t, "key=value", url.RawQuery)
}

func TestZerohttpContext_Param(t *testing.T) {
	t.Run("param without escaping", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test/simple", nil)
		req.SetPathValue("id", "simple")
		ctx := &zerohttpContext{r: req}

		assert.Equal(t, "simple", ctx.Param("id"))
	})

	t.Run("param with escaping", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test/hello%20world", nil)
		req.URL.RawPath = "/test/hello%20world"
		req.SetPathValue("id", "hello world")
		ctx := &zerohttpContext{r: req}

		assert.Equal(t, "hello world", ctx.Param("id"))
	})

	t.Run("param with invalid escape returns original", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test/invalid", nil)
		req.URL.RawPath = "/test/invalid%ZZ"
		req.SetPathValue("id", "invalid%ZZ")
		ctx := &zerohttpContext{r: req}

		assert.Equal(t, "invalid%ZZ", ctx.Param("id"))
	})
}

func TestZerohttpContext_Query(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?foo=bar&baz=qux", nil)
	ctx := &zerohttpContext{r: req}

	assert.Equal(t, "bar", ctx.Query("foo"))
	assert.Equal(t, "qux", ctx.Query("baz"))
	assert.Equal(t, "", ctx.Query("missing"))
}

func TestZerohttpContext_Header(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom", "value")
	ctx := &zerohttpContext{r: req}

	assert.Equal(t, "value", ctx.Header("X-Custom"))
	assert.Equal(t, "", ctx.Header("Missing"))
}

func TestZerohttpContext_EachHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-First", "value1")
	req.Header.Add("X-First", "value2")
	req.Header.Set("X-Second", "value3")
	ctx := &zerohttpContext{r: req}

	headers := make(map[string][]string)
	ctx.EachHeader(func(name, value string) {
		headers[name] = append(headers[name], value)
	})

	assert.Contains(t, headers, "X-First")
	assert.Contains(t, headers, "X-Second")
	assert.Len(t, headers["X-First"], 2)
}

func TestZerohttpContext_BodyReader(t *testing.T) {
	body := strings.NewReader("test body")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	ctx := &zerohttpContext{r: req}

	reader := ctx.BodyReader()
	require.NotNil(t, reader)

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "test body", string(content))
}

func TestZerohttpContext_GetMultipartForm(t *testing.T) {
	t.Run("valid multipart form", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		part, err := writer.CreateFormField("field")
		require.NoError(t, err)
		_, err = part.Write([]byte("value"))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		ctx := &zerohttpContext{r: req}

		form, err := ctx.GetMultipartForm()
		require.NoError(t, err)
		assert.NotNil(t, form)
		assert.Equal(t, "value", form.Value["field"][0])
	})
}

func TestZerohttpContext_SetStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx := &zerohttpContext{w: rec}

	ctx.SetStatus(http.StatusCreated)

	assert.Equal(t, http.StatusCreated, ctx.Status())
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestZerohttpContext_Status(t *testing.T) {
	ctx := &zerohttpContext{status: http.StatusAccepted}

	assert.Equal(t, http.StatusAccepted, ctx.Status())
}

func TestZerohttpContext_AppendHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx := &zerohttpContext{w: rec}

	ctx.AppendHeader("X-Custom", "value1")
	ctx.AppendHeader("X-Custom", "value2")

	assert.Equal(t, []string{"value1", "value2"}, rec.Header()["X-Custom"])
}

func TestZerohttpContext_SetHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx := &zerohttpContext{w: rec}

	ctx.SetHeader("X-Custom", "value1")
	ctx.SetHeader("X-Custom", "value2")

	assert.Equal(t, "value2", rec.Header().Get("X-Custom"))
}

func TestZerohttpContext_BodyWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx := &zerohttpContext{w: rec}

	writer := ctx.BodyWriter()
	require.NotNil(t, writer)

	_, err := writer.Write([]byte("response body"))
	require.NoError(t, err)

	assert.Equal(t, "response body", rec.Body.String())
}

func TestZerohttpContext_TLS(t *testing.T) {
	t.Run("with TLS", func(t *testing.T) {
		tlsState := &tls.ConnectionState{
			Version: tls.VersionTLS12,
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.TLS = tlsState
		ctx := &zerohttpContext{r: req}

		assert.Equal(t, tlsState, ctx.TLS())
	})

	t.Run("without TLS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := &zerohttpContext{r: req}

		assert.Nil(t, ctx.TLS())
	})
}

func TestZerohttpContext_Version(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Proto = "HTTP/2.0"
	req.ProtoMajor = 2
	req.ProtoMinor = 0
	ctx := &zerohttpContext{r: req}

	version := ctx.Version()
	assert.Equal(t, "HTTP/2.0", version.Proto)
	assert.Equal(t, 2, version.ProtoMajor)
	assert.Equal(t, 0, version.ProtoMinor)
}

func TestZerohttpContext_SetReadDeadline(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &zerohttpContext{r: req, w: rec}

	deadline := time.Now().Add(5 * time.Second)
	err := ctx.SetReadDeadline(deadline)
	// ResponseRecorder doesn't support SetReadDeadline so this will error
	assert.Error(t, err)
}

func TestUnwrap(t *testing.T) {
	t.Run("unwraps zerohttp context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		ctx := &zerohttpContext{op: &huma.Operation{}, r: req, w: rec}

		r, w := Unwrap(ctx)
		assert.Equal(t, req, r)
		assert.Equal(t, rec, w)
	})

	t.Run("unwraps wrapped context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		inner := &zerohttpContext{op: &huma.Operation{}, r: req, w: rec}
		wrapped := newWrappedContext(inner)

		r, w := Unwrap(wrapped)
		assert.Equal(t, req, r)
		assert.Equal(t, rec, w)
	})

	t.Run("panics for non-zerohttp context", func(t *testing.T) {
		assert.Panics(t, func() {
			Unwrap(&fakeContext{})
		})
	})
}

// wrappedContext wraps a huma.Context and delegates all methods
type wrappedContext struct {
	inner huma.Context
}

func newWrappedContext(inner huma.Context) *wrappedContext {
	return &wrappedContext{inner: inner}
}

func (w *wrappedContext) Unwrap() huma.Context {
	return w.inner
}

func (w *wrappedContext) Operation() *huma.Operation             { return w.inner.Operation() }
func (w *wrappedContext) Context() context.Context               { return w.inner.Context() }
func (w *wrappedContext) Method() string                         { return w.inner.Method() }
func (w *wrappedContext) Host() string                           { return w.inner.Host() }
func (w *wrappedContext) RemoteAddr() string                     { return w.inner.RemoteAddr() }
func (w *wrappedContext) URL() url.URL                           { return w.inner.URL() }
func (w *wrappedContext) Param(name string) string               { return w.inner.Param(name) }
func (w *wrappedContext) Query(name string) string               { return w.inner.Query(name) }
func (w *wrappedContext) Header(name string) string              { return w.inner.Header(name) }
func (w *wrappedContext) EachHeader(cb func(name, value string)) { w.inner.EachHeader(cb) }
func (w *wrappedContext) BodyReader() io.Reader                  { return w.inner.BodyReader() }
func (w *wrappedContext) GetMultipartForm() (*multipart.Form, error) {
	return w.inner.GetMultipartForm()
}

func (w *wrappedContext) SetReadDeadline(deadline time.Time) error {
	return w.inner.SetReadDeadline(deadline)
}
func (w *wrappedContext) SetStatus(code int)              { w.inner.SetStatus(code) }
func (w *wrappedContext) Status() int                     { return w.inner.Status() }
func (w *wrappedContext) AppendHeader(name, value string) { w.inner.AppendHeader(name, value) }
func (w *wrappedContext) SetHeader(name, value string)    { w.inner.SetHeader(name, value) }
func (w *wrappedContext) BodyWriter() io.Writer           { return w.inner.BodyWriter() }
func (w *wrappedContext) TLS() *tls.ConnectionState       { return w.inner.TLS() }
func (w *wrappedContext) Version() huma.ProtoVersion      { return w.inner.Version() }

type fakeContext struct{}

func (f *fakeContext) Operation() *huma.Operation                 { return nil }
func (f *fakeContext) Context() context.Context                   { return nil }
func (f *fakeContext) Method() string                             { return "" }
func (f *fakeContext) Host() string                               { return "" }
func (f *fakeContext) RemoteAddr() string                         { return "" }
func (f *fakeContext) URL() url.URL                               { return url.URL{} }
func (f *fakeContext) Param(name string) string                   { return "" }
func (f *fakeContext) Query(name string) string                   { return "" }
func (f *fakeContext) Header(name string) string                  { return "" }
func (f *fakeContext) EachHeader(cb func(name, value string))     {}
func (f *fakeContext) BodyReader() io.Reader                      { return nil }
func (f *fakeContext) GetMultipartForm() (*multipart.Form, error) { return nil, nil }
func (f *fakeContext) SetReadDeadline(deadline time.Time) error   { return nil }
func (f *fakeContext) SetStatus(code int)                         {}
func (f *fakeContext) Status() int                                { return 0 }
func (f *fakeContext) AppendHeader(name, value string)            {}
func (f *fakeContext) SetHeader(name, value string)               {}
func (f *fakeContext) BodyWriter() io.Writer                      { return nil }
func (f *fakeContext) TLS() *tls.ConnectionState                  { return nil }
func (f *fakeContext) Version() huma.ProtoVersion                 { return huma.ProtoVersion{} }

// mockRouter implements zh.Router for testing
type mockRouter struct {
	routes map[string]map[string]http.HandlerFunc
}

func newMockRouter() *mockRouter {
	return &mockRouter{
		routes: make(map[string]map[string]http.HandlerFunc),
	}
}

func (m *mockRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handlers, ok := m.routes[r.Method]; ok {
		if handler, ok := handlers[r.URL.Path]; ok {
			handler(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

func (m *mockRouter) GET(path string, handler http.Handler, mw ...func(http.Handler) http.Handler) {
	m.addRoute(http.MethodGet, path, handler)
}

func (m *mockRouter) POST(path string, handler http.Handler, mw ...func(http.Handler) http.Handler) {
	m.addRoute(http.MethodPost, path, handler)
}

func (m *mockRouter) PUT(path string, handler http.Handler, mw ...func(http.Handler) http.Handler) {
	m.addRoute(http.MethodPut, path, handler)
}

func (m *mockRouter) DELETE(path string, handler http.Handler, mw ...func(http.Handler) http.Handler) {
	m.addRoute(http.MethodDelete, path, handler)
}

func (m *mockRouter) PATCH(path string, handler http.Handler, mw ...func(http.Handler) http.Handler) {
	m.addRoute(http.MethodPatch, path, handler)
}

func (m *mockRouter) HEAD(path string, handler http.Handler, mw ...func(http.Handler) http.Handler) {
	m.addRoute(http.MethodHead, path, handler)
}

func (m *mockRouter) OPTIONS(path string, handler http.Handler, mw ...func(http.Handler) http.Handler) {
	m.addRoute(http.MethodOptions, path, handler)
}

func (m *mockRouter) CONNECT(path string, handler http.Handler, mw ...func(http.Handler) http.Handler) {
	m.addRoute(http.MethodConnect, path, handler)
}
func (m *mockRouter) Use(mw ...func(http.Handler) http.Handler)                                   {}
func (m *mockRouter) Group(fn func(zh.Router))                                                    {}
func (m *mockRouter) NotFound(h http.Handler)                                                     {}
func (m *mockRouter) MethodNotAllowed(h http.Handler)                                             {}
func (m *mockRouter) Files(prefix string, embedFS embed.FS, dir string)                           {}
func (m *mockRouter) FilesDir(prefix, dir string)                                                 {}
func (m *mockRouter) Static(embedFS embed.FS, distDir string, fallback bool, apiPrefix ...string) {}
func (m *mockRouter) StaticDir(dir string, fallback bool, apiPrefix ...string)                    {}
func (m *mockRouter) ServeMux() *http.ServeMux                                                    { return nil }
func (m *mockRouter) Logger() log.Logger                                                          { return nil }
func (m *mockRouter) SetLogger(logger log.Logger)                                                 {}
func (m *mockRouter) Config() config.Config                                                       { return config.Config{} }
func (m *mockRouter) SetConfig(cfg config.Config)                                                 {}

func (m *mockRouter) addRoute(method, path string, handler http.Handler) {
	if m.routes[method] == nil {
		m.routes[method] = make(map[string]http.HandlerFunc)
	}
	m.routes[method][path] = func(w http.ResponseWriter, r *http.Request) { handler.ServeHTTP(w, r) }
}

// _ imports to satisfy interface requirements
var (
	_ zh.Router    = (*mockRouter)(nil)
	_ http.Handler = zh.HandlerFunc(nil)
)

func TestZerohttpAdapter_Handle(t *testing.T) {
	tests := []struct {
		name   string
		method string
	}{
		{"GET", http.MethodGet},
		{"POST", http.MethodPost},
		{"PUT", http.MethodPut},
		{"DELETE", http.MethodDelete},
		{"PATCH", http.MethodPatch},
		{"HEAD", http.MethodHead},
		{"OPTIONS", http.MethodOptions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newMockRouter()
			adapter := &zerohttpAdapter{router: router}

			op := &huma.Operation{Method: tt.method, Path: "/test"}
			called := false
			handler := func(ctx huma.Context) {
				called = true
				ctx.SetStatus(http.StatusOK)
			}

			adapter.Handle(op, handler)

			req := httptest.NewRequest(tt.method, "/test", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.True(t, called)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestZerohttpAdapter_ServeHTTP(t *testing.T) {
	router := newMockRouter()
	adapter := &zerohttpAdapter{router: router}

	router.GET("/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	adapter.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Hello", rec.Body.String())
}

func TestNewAdapter(t *testing.T) {
	router := newMockRouter()
	adapter := &zerohttpAdapter{router: router}
	assert.NotNil(t, adapter)
}

func TestNew(t *testing.T) {
	router := newMockRouter()
	adapter := &zerohttpAdapter{router: router}

	cfg := huma.Config{
		OpenAPI: &huma.OpenAPI{
			Info: &huma.Info{
				Title:   "Test API",
				Version: "1.0.0",
			},
		},
	}

	api := huma.NewAPI(cfg, adapter)
	assert.NotNil(t, api)
	assert.Equal(t, "Test API", api.OpenAPI().Info.Title)
	assert.Equal(t, "1.0.0", api.OpenAPI().Info.Version)
}
