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
	"regexp"
	"strconv"
	"strings"

	"github.com/IYouKnow/atlas-drive/pkg/user"
	"golang.org/x/net/webdav"
)

// Server represents the Atlas storage server.
type Server struct {
	Addr       string
	DataDir    string
	UserStore  *user.Store
	QuotaBytes uint64 // If > 0, WebDAV reports this as total quota (used = size of DataDir; available = quota - used).
	HTTPServer *http.Server
}

// New creates a new Server instance. quotaBytes is the advertised storage quota in bytes;
// 0 means report the underlying filesystem's free/used space (previous behaviour).
func New(addr, dataDir string, store *user.Store, quotaBytes uint64) *Server {
	return &Server{
		Addr:       addr,
		DataDir:    dataDir,
		UserStore:  store,
		QuotaBytes: quotaBytes,
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
	// Pre-compile regex to find the closing prop tag (handling namespaces like D:prop or d:prop or just prop)
	// We want to insert BEFORE this tag.
	// Common patterns: </D:prop>, </d:prop>, </prop>
	propEndRegex := regexp.MustCompile(`(</[a-zA-Z0-9_:]*prop>)`)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 2. Disk Space Reporting
		// We only care about PROPFIND on the root.
		if r.Method == "PROPFIND" && r.URL.Path == "/" {
			rb := &responseBuffer{
				ResponseWriter: w,
				body:           new(bytes.Buffer),
				code:           http.StatusOK, // Default to 200 if not set
			}

			// Pass the buffer to the real handler
			next.ServeHTTP(rb, r)

			// If the operation wasn't successful MultiStatus, just pass it through
			if rb.code != http.StatusMultiStatus {
				w.WriteHeader(rb.code)
				w.Write(rb.body.Bytes())
				return
			}

			// It is MultiStatus. Let's process the XML.
			bodyBytes := rb.body.Bytes()
			bodyStr := string(bodyBytes)

			// Determine which namespace prefix is used for quota properties.
			// Usually "D" or "d". The standard lib uses "D".
			// We can try to infer or just default to "D".
			// Let's use "D" but ensure xmlns:D="DAV:" is present.
			// Actually, we are injecting inside < prop > ... < /prop >
			// The surrounding tags define the namespace context.
			// To be safe, we will use <D:quota-...> and trust that <D:multistatus xmlns:D="DAV:"> is at the top.
			// Most clients (including Windows) are fine if we use the same prefix as the root element.

			// Calculate disk usage: either quota-based (share size) or filesystem-based
			var free, used uint64
			var err error
			if s.QuotaBytes > 0 {
				used, err = getDirUsedBytes(s.DataDir)
				if err == nil {
					if used > s.QuotaBytes {
						used = s.QuotaBytes
					}
					if s.QuotaBytes >= used {
						free = s.QuotaBytes - used
					}
				}
			} else {
				absPath, _ := filepath.Abs(s.DataDir)
				if absPath == "" {
					absPath = s.DataDir
				}
				free, used, err = getDiskUsage(absPath)
			}

			if err == nil {
				// Detect usage of namespace prefix for DAV: directly from the XML
				// Standard lib usually uses "D" or "d".
				prefix := "D"
				nsRegex := regexp.MustCompile(`xmlns:([a-zA-Z0-9_]+)="DAV:"`)
				if match := nsRegex.FindStringSubmatch(bodyStr); len(match) > 1 {
					prefix = match[1]
				}

				// Construct insertion string using the MATCHED prefix
				quotaXML := fmt.Sprintf(
					"<%s:quota-available-bytes>%d</%s:quota-available-bytes><%s:quota-used-bytes>%d</%s:quota-used-bytes>",
					prefix, free, prefix, prefix, used, prefix,
				)

				// Find insertion point: the first closing </...prop> tag.
				loc := propEndRegex.FindStringIndex(bodyStr)
				if loc != nil {
					// Insert before the tag
					start := loc[0]
					newBody := bodyStr[:start] + quotaXML + bodyStr[start:]
					bodyBytes = []byte(newBody)
				}
			} else {
				log.Printf("WebDAV Warning: failed to get disk usage: %v", err)
			}

			// Write the modified response
			// IMPORTANT: Update Content-Length to match new size
			w.Header().Set("Content-Length", strconv.Itoa(len(bodyBytes)))
			// Ensure Content-Type is set (webdav lib usually sets it, but good to ensure)
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			}

			// We do NOT need to set "DAV" or "Allow" headers manually because next.ServeHTTP (the library)
			// ALREADY set them on the underlying ResponseWriter (w) before writing to rb,
			// OR it set them on rb (headers map is shared if rb.ResponseWriter is w).
			// responseBuffer embeds http.ResponseWriter, so rb.Header() IS w.Header().
			// So headers are fine.

			w.WriteHeader(rb.code)
			w.Write(bodyBytes)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
