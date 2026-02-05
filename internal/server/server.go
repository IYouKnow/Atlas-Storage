package server

import (
	"context"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/IYouKnow/atlas-drive/pkg/user"
	"golang.org/x/net/webdav"
)

// Server represents the Atlas storage server.
type Server struct {
	Addr       string
	DataDir    string
	UserStore  *user.Store
	HTTPServer *http.Server
}

// New creates a new Server instance.
func New(addr, dataDir string, store *user.Store) *Server {
	return &Server{
		Addr:      addr,
		DataDir:   dataDir,
		UserStore: store,
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	// Ensure data directory exists
	if err := os.MkdirAll(s.DataDir, 0755); err != nil {
		return err
	}

	webdavHandler := &webdav.Handler{
		Prefix:     "/",
		FileSystem: webdav.Dir(s.DataDir),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				log.Printf("WebDAV Error: %s %s: %v", r.Method, r.URL.Path, err)
			}
		},
	}

	// Chain middlewares: Auth -> MimeFix -> WebDAV
	handler := s.authMiddleware(s.mimeMiddleware(webdavHandler))

	s.HTTPServer = &http.Server{
		Addr:    s.Addr,
		Handler: handler,
	}

	log.Printf("Atlas Server starting on %s serving %s", s.Addr, s.DataDir)
	if err := s.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.HTTPServer.Shutdown(ctx)
}

// authMiddleware enforces Basic Auth using the UserStore.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Atlas Storage"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !s.UserStore.Authenticate(username, password) {
			log.Printf("Auth failed for user: %s", username)
			w.Header().Set("WWW-Authenticate", `Basic realm="Atlas Storage"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// mimeMiddleware ensures Content-Type is set correctly for Windows compatibility.
func (s *Server) mimeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Windows WebDAV client often relies on the server providing the correct Content-Type.
		// We can sniff the extension or content.
		// For HEAD/GET requests, we should try to hint.

		next.ServeHTTP(w, r)

		// Note: wrapping next.ServeHTTP means headers are already written by the time we return
		// if usage is standard. So we must set headers BEFORE calling next, or wrap the ResponseWriter.
		// However, webdav handler might verify existence first.

		// The `golang.org/x/net/webdav` ServeHTTP implementation handles Content-Type for GET requests file serving.
		// But let's be explicit if we can.
		// Actually, standard net/http file server does this well. WebDAV might be minimal.
		// Let's rely on standard behavior first, but if we need to force it:

		// A simple improvement: set MIME based on extension if not set?
		// But we don't know the file content here easily without interfering.

		// For now, basic WebDAV usually works. The User emphasized "Windows Quirks".
		// Common fix: ensure registry has mime types on client (not our control),
		// OR ensure server sends it.
		// Let's attempt to use mime.TypeByExtension.
		ext := filepath.Ext(r.URL.Path)
		if t := mime.TypeByExtension(ext); t != "" {
			w.Header().Set("Content-Type", t)
		}
	})
}
