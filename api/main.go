package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("my_secret_key") // TODO: Replace key

var db *sql.DB

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

type Media struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	CreatedAt time.Time `json:"created_at"`
	MediaName string    `json:"media_name"`
}

type Claims struct {
	UserID int `json:"user_id"`
	jwt.StandardClaims
}

func connectToDB(dsn string) (*sql.DB, error) {
	var db *sql.DB
	var err error
	maxRetries := 10

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			log.Printf("Error opening DB connection: %v", err)
		} else {
			// Try pinging the database to ensure the connection is live.
			err = db.Ping()
			if err == nil {
				log.Println("Successfully connected to the database!")
				return db, nil
			}
		}
		log.Printf("MySQL not ready yet (attempt %d/%d). Retrying in 2 seconds...", i+1, maxRetries)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("could not connect to database after %d attempts: %v", maxRetries, err)
}

func main() {
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlPort := os.Getenv("MYSQL_PORT")
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPassword := os.Getenv("MYSQL_PASSWORD")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		mysqlUser, mysqlPassword, mysqlHost, mysqlPort, mysqlDatabase)

	var err error
	db, err = connectToDB(dsn)
	if err != nil {
		log.Fatal("Error initializing database:", err)
	}
	defer db.Close()

	if err := initDB(); err != nil {
		log.Fatal("Error initializing database:", err)
	}

	r := mux.NewRouter()
	r.Use(corsMiddleware)

	r.HandleFunc("/register", handleRegister).Methods("POST")
	r.HandleFunc("/login", handleLogin).Methods("POST")

	s := r.PathPrefix("/").Subrouter()
	s.Use(authMiddleware)
	s.HandleFunc("/media", handleMediaUpload).Methods("POST")
	s.HandleFunc("/media", handleGetMedia).Methods("GET")
	s.HandleFunc("/media/{id}", handleGetMediaByID).Methods("GET")
	s.HandleFunc("/media/{id}", handleDeleteMedia).Methods("DELETE")

	port := "8080"
	log.Printf("API listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func initDB() error {
	userTable := `CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(255) NOT NULL UNIQUE,
		password VARCHAR(255) NOT NULL
	);`
	if _, err := db.Exec(userTable); err != nil {
		return err
	}

	mediaTable := `CREATE TABLE IF NOT EXISTS media (
		id INT AUTO_INCREMENT PRIMARY KEY,
		user_id INT NOT NULL,
		filename VARCHAR(255) NOT NULL,
		mime_type VARCHAR(100) NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		media_name VARCHAR(255) NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`
	_, err := db.Exec(mediaTable)
	return err
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error processing password", http.StatusInternalServerError)
		return
	}

	res, err := db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", creds.Username, string(hashedPassword))
	if err != nil {
		http.Error(w, "Error creating user: "+err.Error(), http.StatusInternalServerError)
		return
	}
	userID, _ := res.LastInsertId()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":  userID,
		"username": creds.Username,
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var user User
	err := db.QueryRow("SELECT id, password FROM users WHERE username = ?", creds.Username).Scan(&user.ID, &user.Password)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: user.ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

		if r.Method == "OPTIONS" {
			http.Error(w, "No Content", http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func handleMediaUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	// Limit upload size (here: 10 MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		http.Error(w, "Error generating file name", http.StatusInternalServerError)
		return
	}
	ext := filepath.Ext(header.Filename)
	filename := base64.URLEncoding.EncodeToString(randomBytes) + ext

	mediaPath := "/media/" + filename
	dst, err := os.Create(mediaPath)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	if _, err = io.Copy(dst, file); err != nil {
		http.Error(w, "Error writing file", http.StatusInternalServerError)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	res, err := db.Exec("INSERT INTO media (user_id, filename, mime_type, media_name) VALUES (?, ?, ?, ?)", userID, filename, mimeType, header.Filename)
	if err != nil {
		http.Error(w, "Error saving media record", http.StatusInternalServerError)
		return
	}
	mediaID, _ := res.LastInsertId()

	fsBaseURL := os.Getenv("FS_BASE_URL")
	mediaURL := fmt.Sprintf("%s/%s", fsBaseURL, filename)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"media_id":   mediaID,
		"filename":   filename,
		"mime_type":  mimeType,
		"url":        mediaURL,
		"media_name": header.Filename,
	})
}

func handleGetMedia(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query("SELECT id, filename, mime_type, created_at, media_name FROM media WHERE user_id = ?", userID)
	if err != nil {
		http.Error(w, "Error retrieving media", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	medias := []map[string]interface{}{}
	for rows.Next() {
		var m Media
		if err := rows.Scan(&m.ID, &m.Filename, &m.MimeType, &m.CreatedAt, &m.MediaName); err != nil {
			continue
		}
		medias = append(medias, map[string]interface{}{
			"id":         m.ID,
			"mime_type":  m.MimeType,
			"created_at": m.CreatedAt,
			"media_name": m.MediaName,
		})
	}

	json.NewEncoder(w).Encode(medias)
}

func handleGetMediaByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}
	fsBaseURL := os.Getenv("FS_BASE_URL")

	vars := mux.Vars(r)
	mediaIDStr, exists := vars["id"]
	if !exists {
		http.Error(w, "Media ID is required", http.StatusBadRequest)
		return
	}
	mediaID, err := strconv.Atoi(mediaIDStr)
	if err != nil {
		http.Error(w, "Invalid media ID", http.StatusBadRequest)
		return
	}

	var m Media
	rows, err := db.Query("SELECT id, filename, mime_type, created_at, media_name FROM media WHERE id = ? AND user_id = ?", mediaID, userID)
	if err != nil {
		http.Error(w, "Error retrieving media", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	if !rows.Next() {
		http.Error(w, "Media not found or unauthorized", http.StatusNotFound)
		return
	}

	if err := rows.Scan(&m.ID, &m.Filename, &m.MimeType, &m.CreatedAt, &m.MediaName); err != nil {
		http.Error(w, "Error scanning media", http.StatusInternalServerError)
		return
	}

	mediaURL := fmt.Sprintf("%s/%s", fsBaseURL, m.Filename)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         m.ID,
		"filename":   m.Filename,
		"mime_type":  m.MimeType,
		"created_at": m.CreatedAt,
		"url":        mediaURL,
		"media_name": m.MediaName,
	})
}

func handleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}
	vars := mux.Vars(r)
	mediaIDStr, exists := vars["id"]
	if !exists {
		http.Error(w, "Media ID is required", http.StatusBadRequest)
		return
	}
	mediaID, err := strconv.Atoi(mediaIDStr)
	if err != nil {
		http.Error(w, "Invalid media ID", http.StatusBadRequest)
		return
	}

	var filename string
	if err := db.QueryRow("SELECT filename FROM media WHERE id = ? AND user_id = ?", mediaID, userID).Scan(&filename); err != nil {
		http.Error(w, "Media not found or unauthorized", http.StatusNotFound)
		return
	}

	if _, err := db.Exec("DELETE FROM media WHERE id = ?", mediaID); err != nil {
		http.Error(w, "Error deleting media", http.StatusInternalServerError)
		return
	}

	mediaPath := "/media/" + filename
	if err := os.Remove(mediaPath); err != nil {
		log.Println("Error deleting file:", err)
	}

	w.WriteHeader(http.StatusNoContent)
}
