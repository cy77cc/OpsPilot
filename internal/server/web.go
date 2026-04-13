package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	webui "github.com/cy77cc/OpsPilot/web"
	"github.com/gin-gonic/gin"
)

// RegisterWebStaticRoutes mounts the embedded frontend build in non-development environments.
func RegisterWebStaticRoutes(r *gin.Engine) {
	if config.IsDevelopment() {
		return
	}

	distFS, err := webui.SubDist()
	if err != nil {
		return
	}

	staticServer := http.FileServer(http.FS(distFS))
	r.GET("/assets/*filepath", gin.WrapH(staticServer))
	r.GET("/vite.svg", gin.WrapH(staticServer))
	r.GET("/favicon.ico", gin.WrapH(staticServer))

	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		indexFile, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "frontend not built"})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexFile)
	})
}
