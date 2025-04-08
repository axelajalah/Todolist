package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
)

type Todo struct {
	ID        string `json:"id"`
	Judul     string `json:"judul"`
	Deskripsi string `json:"deskripsi"`
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func RegisterTodoRoutes(router *mux.Router) {
	router.HandleFunc("/todos", authMiddleware(getTodos)).Methods("GET")
	router.HandleFunc("/todos", authMiddleware(createTodo)).Methods("POST")
	router.HandleFunc("/todos/{id}", authMiddleware(updateTodo)).Methods("PUT")
	router.HandleFunc("/todos/{id}", authMiddleware(deleteTodo)).Methods("DELETE")
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		log.Printf("Authenticated user ID: %s", claims.UserID)
		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func getTodos(w http.ResponseWriter, r *http.Request) {
	// Get userID from context
	ctxUserID := r.Context().Value("userID")
	if ctxUserID == nil {
		log.Println("Error: userID not found in context")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get pagination parameters
	page := 1
	limit := 10
	if p := r.URL.Query().Get("page"); p != "" {
		if pInt, err := strconv.Atoi(p); err == nil && pInt > 0 {
			page = pInt
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if lInt, err := strconv.Atoi(l); err == nil && lInt > 0 && lInt <= 100 {
			limit = lInt
		}
	}
	offset := (page - 1) * limit

	// Get total count
	var total int
	err := DB.QueryRow("SELECT COUNT(*) FROM todo").Scan(&total)
	if err != nil {
		log.Printf("Count query failed: %v", err)
		http.Error(w, "Failed to count todos", http.StatusInternalServerError)
		return
	}

	// Query database with pagination
	query := "SELECT id, judul, deskripsi FROM todo LIMIT ? OFFSET ?"
	rows, err := DB.Query(query, limit, offset)
	if err != nil {
		log.Printf("Database query failed. Query: %s, Error: %v", query, err)
		http.Error(w, "Failed to fetch todos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Process results
	var todos []Todo
	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.ID, &todo.Judul, &todo.Deskripsi); err != nil {
			log.Printf("Row scan failed. Error: %v", err)
			http.Error(w, "Failed to process todos", http.StatusInternalServerError)
			return
		}
		todos = append(todos, todo)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error: %v", err)
		http.Error(w, "Failed to process todos", http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"data":  todos,
		"page":  page,
		"limit": limit,
		"total": total,
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("JSON encoding failed. Error: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	// Check content type is multipart/form-data
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		log.Printf("Invalid content type: %s", contentType)
		http.Error(w, "Content-Type must be multipart/form-data", http.StatusBadRequest)
		return
	}

	// Parse multipart form data
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB limit
		log.Printf("Multipart form parse error: %v", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	judul := r.FormValue("judul")
	deskripsi := r.FormValue("deskripsi")

	log.Printf("Received form data - judul: '%s', deskripsi: '%s'", judul, deskripsi)
	log.Printf("Full form data: %+v", r.PostForm)

	if judul == "" {
		log.Println("Error: judul is empty")
		http.Error(w, "Judul cannot be empty", http.StatusBadRequest)
		return
	}
	if deskripsi == "" {
		log.Println("Error: deskripsi is empty")
		http.Error(w, "Deskripsi cannot be empty", http.StatusBadRequest)
		return
	}

	newTodo := Todo{
		Judul:     judul,
		Deskripsi: deskripsi,
	}

	// Create todo
	log.Printf("Executing INSERT: judul='%s', deskripsi='%s'", newTodo.Judul, newTodo.Deskripsi)

	result, err := DB.Exec("INSERT INTO todo (judul, deskripsi) VALUES (?, ?)",
		newTodo.Judul, newTodo.Deskripsi)
	if err != nil {
		log.Printf("Database INSERT error: %v", err)
		http.Error(w, "Failed to create todo", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Insert successful, rows affected: %d", rowsAffected)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newTodo)
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	// Check content type is multipart/form-data
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		log.Printf("Invalid content type: %s", contentType)
		http.Error(w, "Content-Type must be multipart/form-data", http.StatusBadRequest)
		return
	}

	// Parse multipart form data
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB limit
		log.Printf("Multipart form parse error: %v", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	judul := r.FormValue("judul")
	deskripsi := r.FormValue("deskripsi")

	log.Printf("Received update form data - judul: '%s', deskripsi: '%s'", judul, deskripsi)

	if judul == "" {
		log.Println("Error: judul is empty")
		http.Error(w, "Judul cannot be empty", http.StatusBadRequest)
		return
	}
	if deskripsi == "" {
		log.Println("Error: deskripsi is empty")
		http.Error(w, "Deskripsi cannot be empty", http.StatusBadRequest)
		return
	}

	// Verify todo exists
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM todo WHERE id = ?", id).Scan(&count)
	if err != nil || count == 0 {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	_, err = DB.Exec("UPDATE todo SET judul = ?, deskripsi = ? WHERE id = ?",
		judul, deskripsi, id)
	if err != nil {
		log.Printf("Database UPDATE error: %v", err)
		http.Error(w, "Failed to update todo", http.StatusInternalServerError)
		return
	}

	updatedTodo := Todo{
		ID:        id,
		Judul:     judul,
		Deskripsi: deskripsi,
	}

	json.NewEncoder(w).Encode(updatedTodo)
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	// Verify todo exists
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM todo WHERE id = ?", id).Scan(&count)
	if err != nil || count == 0 {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	_, err = DB.Exec("DELETE FROM todo WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("sampun dihapus mas"))
}
