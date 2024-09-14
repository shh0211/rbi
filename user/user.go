package user

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"rbi/models"
	"rbi/sqlite"
)

func RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/register", newUser).Methods(http.MethodPost)
	router.HandleFunc("/deactivate", deleteUser).Methods(http.MethodPost)
	router.HandleFunc("/login", userLogin).Methods(http.MethodGet)
	router.HandleFunc("/user/check", hasUsers).Methods(http.MethodGet)
}

var Db = sqlite.Db

func newUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Hash the password before saving
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)

	// Create new user in the database
	if err := Db.Create(&user).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User registered successfully",
	})
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find and delete the user by username
	if err := Db.Where("username = ?", user.Username).Delete(&models.User{}).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User deactivated successfully",
	})
}

// userLogin handles user login
func userLogin(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	var user models.User
	if err := Db.Where("username = ?", username).First(&user).Error; err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Compare the hashed password with the entered password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful",
	})
}

func hasUsers(w http.ResponseWriter, r *http.Request) {
	var count int64
	if err := Db.Model(&models.User{}).Count(&count).Error; err != nil {
		http.Error(w, "Failed to query user data", http.StatusInternalServerError)
		return
	}

	hasUsers := count > 0

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{
		"hasUsers": hasUsers,
	})
}
