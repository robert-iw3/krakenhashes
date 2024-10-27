package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ZerkerEOD/hashdom/hashdom-backend/internal/database"
	"github.com/ZerkerEOD/hashdom/hashdom-backend/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Change this line to use user.User
	user, err := database.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// The rest of the function remains the same
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := jwt.GenerateToken(user.ID)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	if err := database.StoreToken(user.ID, token); err != nil {
		http.Error(w, "Error storing token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(time.Hour * 24 * 7 / time.Second), // 1 week
	})

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err == nil {
		if err := database.RemoveToken(cookie.Value); err != nil {
			http.Error(w, "Error removing token", http.StatusInternalServerError)
			return
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusOK)
}

func CheckAuthHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]bool{"authenticated": false})
		return
	}

	_, err = jwt.ValidateToken(cookie.Value)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]bool{"authenticated": false})
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"authenticated": true})
}
