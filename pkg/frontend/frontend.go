package frontend

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/spf13/viper"
)

func Execute() {
	staticPath := viper.GetString("frontend.staticPath")
	port := viper.GetString("frontend.port")

	// Custom handler for CSS files
	http.HandleFunc("/static/css/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(staticPath+"htmx/static", r.URL.Path))
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	})

	// Serve other static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath+"htmx/static/"))))

	// Serve index.html from the htmx directory as the root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, staticPath+"htmx/index.html")
	})

	// Serve other files from the htmx directory
	http.Handle("/htmx/", http.StripPrefix("/htmx/", http.FileServer(http.Dir(staticPath+"htmx/"))))

	log.Printf("Frontend server listening on :%s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Frontend server error: %v", err)
	}
}
