package auth

import (
	"log"
	"net/http"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"db/database"
	"utils/jwt"
	"config"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Error hashing password:", err)
		return "", err
	}
	return string(hashedPassword), nil
}

func ComparePasswords(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

func IsValidPassword(password string) bool {
	return len(password) >= 8
}

func CheckValidToken(r *http.Request) (uint, string, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return 0, "", errors.New("no token found")
	}

	userId, role, err := jwt.ParseToken(cookie.Value)
	if err != nil {
		return 0, "", errors.New("invalid token")
	}

	return userId, role, nil
}

func SignUpHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	var user database.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err == nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	newUser := database.User{
		Username: username,
		Password: hashedPassword,
		Role:     config.DEFAULT_ROLE,
	}

	if err := database.DB.Create(&newUser).Error; err != nil {
		http.Error(w, "Error saving user to the database", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "User %s successfully registered!", username)
}


func LoginHandler(w http.ResponseWriter, r *http.Request) {
	userId, role, err := CheckValidToken(r)
	if err == nil {
		fmt.Fprintf(w, "Already logged in as user %d with role %s.", userId, role)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var user database.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !ComparePasswords(user.Password, password) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := jwt.GenerateSecureToken(user.Id, user.Role)
	if err != nil {
		http.Error(w, "Unable to generate token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,           
		Path:     "/",
		HttpOnly: true,            
		Secure:   true,            
		SameSite: http.SameSiteStrictMode,
	})

	fmt.Fprintf(w, "Login successful.")
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().Add(-time.Hour),
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

