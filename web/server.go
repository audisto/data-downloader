package web

import (
	"fmt"

	_ "github.com/audisto/data-downloader/web/statik" // compiled static files
	"github.com/gin-gonic/gin"
)

// StartWebInterface -
func StartWebInterface(port uint, debug bool) {

	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}
	server := gin.New()
	server.Use(Logger())
	server.Use(gin.Recovery())
	server.LoadHTMLGlob("web/static/templates/*")
	server.Static("/static", "./web/static")
	server.GET("/", homeHandler)
	server.GET("/login", loginHandler)
	fmt.Print("here")
	banner := `                   _ _     _        
    /\            | (_)   | |       
   /  \  _   _  __| |_ ___| |_ ___  
  / /\ \| | | |/ _  | / __| __/ _ \ 
 / ____ \ |_| | (_| | \__ \ || (_) |
/_/    \_\__,_|\__,_|_|___/\__\___/ 

- server started: http://localhost:%d
`
	fmt.Printf(banner, port)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server.Run(addr) // listen and serve on 0.0.0.0:8080
}
