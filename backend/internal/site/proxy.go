// Package site serves deployed sites. Each deployment gets its own host - 20.localhost:3000
// for deployment 20 - so the site is served from the root of that host and the absolute
// asset paths a build emits ("/assets/index-abc.js") resolve correctly without the build
// needing to know where it was published.
package site

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sudhanshu042004/orcs/internal/container"
	"github.com/sudhanshu042004/orcs/internal/repository"
	"github.com/sudhanshu042004/orcs/internal/s3"
	"github.com/sudhanshu042004/orcs/internal/stack"
)

// baseDomain is the host deployed sites hang off, including the port the browser talks to.
func baseDomain() string {
	if domain := os.Getenv("SITE_BASE_DOMAIN"); domain != "" {
		return domain
	}
	return "localhost:3000"
}

// RuntimeHost is where a dynamic app's published port can be reached. The container ports
// are published on the docker host, which is loopback when the backend runs on that host
// and the host gateway when it runs in a container of its own.
func RuntimeHost() string {
	if host := os.Getenv("RUNTIME_HOST"); host != "" {
		return host
	}
	return "127.0.0.1"
}

func scheme() string {
	if s := os.Getenv("SITE_SCHEME"); s != "" {
		return s
	}
	return "http"
}

// URL is the address a deployment is served at.
func URL(depId int64) string {
	return fmt.Sprintf("%s://%d.%s", scheme(), depId, baseDomain())
}

// Describe reports how deployed sites are addressed. Used at startup.
func Describe() string {
	return fmt.Sprintf("%s://<deployment-id>.%s", scheme(), baseDomain())
}

// deploymentIdFromHost pulls the deployment id out of a request host. Only a leading
// all-digit label counts, so the API's own host ("localhost:3000") is left alone.
func deploymentIdFromHost(host string) (int64, bool) {
	// Host carries a port, and may be an IPv6 literal in brackets
	if h, _, found := strings.Cut(host, "]"); found {
		host = strings.TrimPrefix(h, "[")
	} else if h, _, found := strings.Cut(host, ":"); found {
		host = h
	}

	// An IP address also leads with digits - 127.0.0.1 is the API, not deployment 127
	if net.ParseIP(host) != nil {
		return 0, false
	}

	label, _, found := strings.Cut(host, ".")
	if !found || label == "" {
		return 0, false
	}

	depId, err := strconv.ParseInt(label, 10, 64)
	if err != nil || depId <= 0 {
		return 0, false
	}
	return depId, true
}

// objectKey maps a request path to a stored key under the deployment's prefix. The
// returned path is always inside that prefix, whatever the request asked for.
func objectKey(depId int64, requestPath string) string {
	clean := path.Clean("/" + strings.TrimPrefix(requestPath, "/"))
	if strings.HasSuffix(requestPath, "/") || clean == "/" {
		clean = path.Join(clean, "index.html")
	}
	return fmt.Sprintf("deployments/%d%s", depId, clean)
}

// target is what one deployment needs for its requests to be served: either a static
// prefix in object storage, or a live app to forward to.
type target struct {
	dynamic bool
	proxy   *httputil.ReverseProxy
	err     string // set when the deployment cannot be served at all
}

// Looking a deployment up means a database round trip, and a page load asks for every one
// of its assets, so the answer is held briefly.
var (
	targetsMu sync.Mutex
	targets   = map[int64]targetEntry{}
)

type targetEntry struct {
	target  target
	expires time.Time
}

const targetTTL = 5 * time.Second

func lookup(depId int64) target {
	targetsMu.Lock()
	if entry, ok := targets[depId]; ok && time.Now().Before(entry.expires) {
		targetsMu.Unlock()
		return entry.target
	}
	targetsMu.Unlock()

	resolved := resolve(depId)

	targetsMu.Lock()
	targets[depId] = targetEntry{target: resolved, expires: time.Now().Add(targetTTL)}
	targetsMu.Unlock()
	return resolved
}

func resolve(depId int64) target {
	dep, err := repository.GetPublicDeployment(depId)
	if err != nil {
		return target{err: "no site is deployed here"}
	}
	st, ok := stack.Get(dep.Stack)
	if !ok || st.Kind != stack.KindDynamic {
		return target{}
	}

	// Docker hands out a new host port every time a container starts, so the live mapping
	// wins over the one recorded at deploy time and a restarted app is picked back up
	hostPort := 0
	if containerId, err := repository.GetDeploymentContainer(depId); err == nil && containerId != "" {
		if port, err := container.HostPort(containerId, st.Port); err == nil {
			hostPort = port
		}
	}
	if hostPort == 0 {
		hostPort, _ = repository.GetDeploymentHostPort(depId)
	}
	if hostPort == 0 {
		return target{dynamic: true, err: "this app is not running"}
	}

	backend := &url.URL{Scheme: "http", Host: net.JoinHostPort(RuntimeHost(), strconv.Itoa(hostPort))}
	proxy := httputil.NewSingleHostReverseProxy(backend)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		fmt.Printf("[Site] deployment %d: %s\n", depId, err.Error())
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("this app is not responding"))
	}
	return target{dynamic: true, proxy: proxy}
}

// Forget drops a deployment's cached target, so a deployment that has just been deleted or
// redeployed is looked up again rather than served from a stale entry.
func Forget(depId int64) {
	targetsMu.Lock()
	delete(targets, depId)
	targetsMu.Unlock()
}

// Middleware serves a deployed site when the request arrives on a deployment host, and
// hands every other request to the API routes behind it.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		depId, ok := deploymentIdFromHost(c.Request.Host)
		if !ok {
			c.Next()
			return
		}
		c.Abort()

		t := lookup(depId)
		if t.err != "" {
			c.String(http.StatusBadGateway, t.err)
			return
		}

		// A running app handles its own routing and methods; a directory of files in
		// object storage can only answer reads
		if t.dynamic {
			t.proxy.ServeHTTP(c.Writer, c.Request)
			return
		}
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.String(http.StatusMethodNotAllowed, "a static site only answers GET")
			return
		}

		serve(c, depId)
	}
}

func serve(c *gin.Context, depId int64) {
	key := objectKey(depId, c.Request.URL.Path)

	obj, err := s3.GetObject(c.Request.Context(), key)
	if errors.Is(err, s3.ErrNotFound) {
		// A path with an extension is asking for a file that really is missing. Anything
		// else is most likely a client-side route, which the site's index.html handles.
		if path.Ext(key) != "" {
			c.String(http.StatusNotFound, "not found")
			return
		}
		key = objectKey(depId, "/")
		obj, err = s3.GetObject(c.Request.Context(), key)
	}

	if errors.Is(err, s3.ErrNotFound) {
		c.String(http.StatusNotFound, "no site is deployed here")
		return
	}
	if err != nil {
		fmt.Printf("[Site] deployment %d: could not read %s: %s\n", depId, key, err.Error())
		c.String(http.StatusBadGateway, "the site could not be loaded")
		return
	}
	defer obj.Body.Close()

	if obj.ETag != "" {
		c.Header("ETag", obj.ETag)
	}
	// Build assets carry a content hash in the name, so they can be cached hard. The
	// entry point cannot - it is what points at the current assets.
	if path.Ext(key) == ".html" {
		c.Header("Cache-Control", "no-cache")
	} else {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	}

	c.DataFromReader(http.StatusOK, obj.Size, obj.ContentType, obj.Body, nil)
}
