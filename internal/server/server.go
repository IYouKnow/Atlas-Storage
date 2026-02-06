package server

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
				// 1. Log Noise Suppression
				// Suppress 404 errors for common Windows system files
				if os.IsNotExist(err) {
					base := strings.ToLower(filepath.Base(r.URL.Path))
					switch base {
					case "desktop.ini", "autorun.inf", "thumbs.db", "folder.jpg":
						return
					}
				}
				log.Printf("WebDAV Error: %s %s: %v", r.Method, r.URL.Path, err)
			}
		},
	}

	// Chain middlewares: Auth -> MimeFix -> Quota -> WebDAV
	handler := s.authMiddleware(s.mimeMiddleware(s.quotaMiddleware(webdavHandler)))

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
		ext := filepath.Ext(r.URL.Path)
		if t := mime.TypeByExtension(ext); t != "" {
			w.Header().Set("Content-Type", t)
		}
		next.ServeHTTP(w, r)
	})
}

// responseBuffer captures the response to allow modification.
type responseBuffer struct {
	http.ResponseWriter
	body *bytes.Buffer
	code int
}

func (w *responseBuffer) WriteHeader(code int) {
	w.code = code
}

func (w *responseBuffer) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

// quotaMiddleware implements RFC 4331 by injecting quota properties into PROPFIND responses.
func (s *Server) quotaMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 2. Disk Space Reporting
		// Intercept PROPFIND on the root path
		if r.Method == "PROPFIND" && r.URL.Path == "/" {
			rb := &responseBuffer{
				ResponseWriter: w,
				body:           new(bytes.Buffer),
				code:           http.StatusOK, // Default
			}

			next.ServeHTTP(rb, r)

			// If successful PROPFIND (MultiStatus), inject quota
			if rb.code == http.StatusMultiStatus {
				bodyStr := rb.body.String()

				// Ensure we use absolute path for Statfs
				absPath, _ := filepath.Abs(s.DataDir) // ignoring error, default to s.DataDir if fail
				if absPath == "" {
					absPath = s.DataDir
				}

				// Calculate quota
				free, used, err := getDiskUsage(absPath)
				total := free + used

				// Explicit Debug Logs as requested
				log.Printf("DEBUG QUOTA: Path=%s | Total=%d | Free=%d", absPath, total, free)

				if err == nil {
					// Inject properties before the first closing </D:prop>
					// Ensure strict namespacing. We assume the response uses 'D:' prefix for DAV:
					// If the webdav library changes prefixes, this might need adjustment, but D: is standard for x/net/webdav.
					quotaProps := fmt.Sprintf("<D:quota-available-bytes>%d</D:quota-available-bytes><D:quota-used-bytes>%d</D:quota-used-bytes>", free, used)
					if idx := strings.Index(bodyStr, "</D:prop>"); idx != -1 {
						bodyStr = bodyStr[:idx] + quotaProps + bodyStr[idx:]
					}
				}

				// Copy headers from the captured response (WebDAV sets these)
				// Note: Since we didn't call WriteHeader on w, we can set them now.
				// However, next.ServeHTTP logic might have tried to set headers on rb.
				// Since rb embeds http.ResponseWriter, calling rb.Header() calls w.Header().
				// So headers set by webdav handler are ALREADY in w.

				// We just need to update Content-Length if it changed
				w.Header().Set("Content-Length", fmt.Sprint(len(bodyStr)))
				w.WriteHeader(rb.code)
				w.Write([]byte(bodyStr))
				return
			}

			// If not MultiStatus or we shouldn't modify it, just flush buffer
			w.WriteHeader(rb.code)
			w.Write(rb.body.Bytes())
			return
		}

		next.ServeHTTP(w, r)
	})
}
