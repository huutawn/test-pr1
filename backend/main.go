package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	db        *sql.DB
	jwtSecret string
}

var app App

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type signupReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshReq struct {
	Token string `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Email        string `json:"email,omitempty"`
}

func main() {
	port := getEnv("PORT", "8080")
	secret := getEnv("JWT_SECRET", "default-secret-please-change")
	// Connect to PostgreSQL via DSN from environment
	dsn := postgresDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	if err := migrate(db); err != nil {
		log.Fatalf("failed to migrate db: %v", err)
	}

	app = App{db: db, jwtSecret: secret}
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", registerHandler)
	mux.HandleFunc("/auth/login", loginHandler)
	mux.HandleFunc("/auth/refresh", refreshHandler)
	mux.Handle("/me", authMiddleware(http.HandlerFunc(meHandler)))

	// Basic CORS for local testing; tighten for prod
	handler := corsMiddleware(mux)

	log.Printf("backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func migrate(db *sql.DB) error {
	// users table (PostgreSQL)
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
        id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        email TEXT UNIQUE NOT NULL,
        password_hash TEXT NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )`)
	if err != nil {
		return err
	}
	// refresh tokens table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS refresh_tokens (
        id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        token TEXT UNIQUE NOT NULL,
        user_id BIGINT NOT NULL,
        expires_at TIMESTAMPTZ NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users(id)
    )`)
	return err
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req signupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}
	if req.Email == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing fields"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	now := time.Now()
	var id int64
	err = app.db.QueryRow("INSERT INTO users(email, password_hash, created_at) VALUES($1,$2,$3) RETURNING id", req.Email, string(hash), now).Scan(&id)
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "user exists"})
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "email": req.Email})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var user User
	row := app.db.QueryRow("SELECT id, email, password_hash FROM users WHERE email=$1", req.Email)
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	accessToken, err := generateJWT(user.ID, user.Email, 15*time.Minute, app.jwtSecret)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	refreshToken, err := generateRandomToken(32)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = app.db.Exec("INSERT INTO refresh_tokens(token, user_id, expires_at) VALUES($1,$2,$3)", refreshToken, user.ID, time.Now().Add(7*24*time.Hour))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tokenResponse{AccessToken: accessToken, RefreshToken: refreshToken, Email: user.Email})
}

func refreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var userID int64
	var expiresAt time.Time
	err := app.db.QueryRow("SELECT user_id, expires_at FROM refresh_tokens WHERE token=$1", req.Token).Scan(&userID, &expiresAt)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if time.Now().After(expiresAt) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// fetch email for token claims
	var email string
	_ = app.db.QueryRow("SELECT email FROM users WHERE id=?", userID).Scan(&email)
	newToken, err := generateJWT(userID, email, 15*time.Minute, app.jwtSecret)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tokenResponse{AccessToken: newToken, Email: email})
}

type key int

const userKey key = 0

func meHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(userKey)
	if uid == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userID := uid.(int64)
	var email string
	row := app.db.QueryRow("SELECT email FROM users WHERE id=?", userID)
	if err := row.Scan(&email); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"id": userID, "email": email})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(app.jwtSecret), nil
		})
		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		sub, ok := claims["sub"].(float64)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userKey, int64(sub))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateJWT(userID int64, email string, ttl time.Duration, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   time.Now().Add(ttl).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func generateRandomToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func postgresDSN() string {
	host := getEnv("PG_HOST", "localhost")
	port := getEnv("PG_PORT", "5432")
	user := getEnv("PG_USER", "postgres")
	pass := getEnv("PG_PASSWORD", "")
	dbname := getEnv("PG_DB", "auth_demo")
	sslmode := getEnv("PG_SSLMODE", "disable")
	if pass != "" {
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", url.QueryEscape(user), url.QueryEscape(pass), host, port, dbname, sslmode)
	}
	return fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s", url.QueryEscape(user), host, port, dbname, sslmode)
}
