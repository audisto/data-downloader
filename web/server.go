/*
IMPORTANT: all static files are embedded inside the final binary.
*/

package web

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	_ "github.com/audisto/data-downloader/web/statik" // compiled static files
	"github.com/gin-gonic/gin"
	"github.com/rakyll/statik/fs"
)

var (
	// EmbeddedFS Keep a reference to the embedded FileSystem holding our static files and templates
	EmbeddedFS http.FileSystem
	// TemplatesFSPrefix the path to template files *inside* the embedded FileSystem
	TemplatesFSPrefix = "/templates/"
	// TemplateFiles the list of template files to look for inside the embedded FileSystem
	TemplateFiles = [...]string{
		"footer.html", "head.html", "home.html", "login.html"}
)

func init() {
	var err error
	EmbeddedFS, err = fs.New()
	if err != nil {
		log.Fatal(err)
	}
}

// StartWebInterface -
func StartWebInterface(port uint, debug bool) {

	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}

	server := gin.New()
	server.SetHTMLTemplate(getTemplates())
	server.Use(Logger())
	server.Use(gin.Recovery())
	server.StaticFS("/static", EmbeddedFS)
	server.GET("/", homeHandler)
	server.GET("/login", loginHandler)
	banner :=
		`                   _ _     _        
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

func getTemplates() *template.Template {
	tmpl := template.New("")

	for _, tmpName := range TemplateFiles {

		templateFile, err := EmbeddedFS.Open(TemplatesFSPrefix + tmpName)
		if err != nil {
			log.Fatal(err)
		}
		fileBytes, err := ioutil.ReadAll(templateFile)
		if err != nil {
			log.Fatal(err)
		}
		tmpl, err = tmpl.New(tmpName).Parse(string(fileBytes))
		if err != nil {
			log.Fatal(err)
		}
	}

	return tmpl
}
