package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/katierevinska/calculatorService/internal/auth"
)

type contextKey string

const UserIDKey contextKey = "userID"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := auth.ExtractTokenFromHeader(r)
		if err != nil {
			log.Printf("AuthMiddleware: Error extracting token: %v (Path: %s)", err, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized: " + err.Error()})
			return
		}

		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			log.Printf("AuthMiddleware: Error validating token: %v (Path: %s)", err, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized: Invalid token"})
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		log.Printf("AuthMiddleware: User %d authenticated for path %s", claims.UserID, r.URL.Path)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
