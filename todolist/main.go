package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	// Initialize database
	if err := InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer DB.Close()

	// Create router with logging
	router := mux.NewRouter()
	router.Use(loggingMiddleware)

	// Register routes
	RegisterUserRoutes(router)
	RegisterTodoRoutes(router)

	// Start server
	log.Println("Server starting on :8000")
	log.Fatal(http.ListenAndServe(":8000", router))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}
