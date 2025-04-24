package handlers

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Amirali-Amirifar/yeetcode/backend/config"
	"github.com/Amirali-Amirifar/yeetcode/backend/db"
	"github.com/Amirali-Amirifar/yeetcode/backend/utils/jwt"
	"golang.org/x/crypto/bcrypt"
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

func SignUpHandler(c *gin.Context) {
	log.Println("SignUpHandler reached")
	username := c.PostForm("username")
	password := c.PostForm("password")
	passwordRepeat := c.PostForm("password_repeat")

	redirectErr := func(msg string) {
		c.Redirect(http.StatusTemporaryRedirect, "/signup?err="+url.QueryEscape(msg))
	}

	redirectErr("Username")
	log.Printf("Received signup request - Username: %s", username)

	// Validate form data
	if username == "" || password == "" || passwordRepeat == "" {
		log.Println("Empty fields detected")
		redirectErr("Missing fields")
		return
	}

	// Validate email format
	if !strings.Contains(username, "@") {
		log.Println("Invalid email format - missing @")
		redirectErr("Please enter a valid email address")
		return
	}

	// Check password match
	if password != passwordRepeat {
		log.Println("Passwords do not match")
		redirectErr("Password does not match")
		return
	}

	// Check password strength
	if !IsValidPassword(password) {
		log.Println("Password too short")
		redirectErr("Password too short")
		return
	}

	var user db.User
	err := db.DB.Where("username = ?", username).First(&user).Error
	log.Printf("Checking username: %s, Error: %v", username, err)

	// Only return error if the error is not "record not found"
	if err != nil && err.Error() != "record not found" {
		log.Printf("Database error checking username: %v", err)
		redirectErr("Something went wrong, try again later")
		return
	}

	// If no error, user exists
	if err == nil {
		log.Println("Username already exists")
		redirectErr("Username already exists")
		return
	}

	log.Println("Hashing password...")
	hashedPassword, err := HashPassword(password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		redirectErr("Something went wrong, try again later - 2")
		return
	}

	log.Println("Creating new user...")
	newUser := db.User{
		Username: username,
		Password: hashedPassword,
		Role:     config.DefaultRole,
	}

	if err := db.DB.Create(&newUser).Error; err != nil {
		log.Printf("Error saving user to database: %v", err)
		redirectErr("Something went wrong, try again later - 3")
		return
	}

	log.Println("User created successfully, generating token...")
	// Generate token and set cookie
	token, err := jwt.GenerateSecureToken(newUser.Id, newUser.Role)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		c.HTML(http.StatusInternalServerError, "signup.gohtml", gin.H{
			"title": "Signup",
			"page":  "Signup",
			"err":   "Error creating session",
		})
		return
	}

	log.Println("Setting session cookie and redirecting...")
	c.SetCookie("session_token", token, 24*3600, "/", "", true, true)
	c.Redirect(http.StatusFound, "/")
}

func LoginHandler(c *gin.Context) {
	username := c.PostForm("email")
	password := c.PostForm("password")

	var user db.User
	if err := db.DB.Where("username = ?", username).First(&user).Error; err != nil {
		c.Redirect(http.StatusFound, "/login?err=wrong-credentials")
		return
	}

	if !ComparePasswords(user.Password, password) {
		c.Redirect(http.StatusFound, "/login?err=wrong-credentials")
		return
	}

	token, err := jwt.GenerateSecureToken(user.Id, user.Role)
	if err != nil {
		c.Redirect(http.StatusFound, "/login?err=token-error")
		return
	}

	// Set cookie with proper settings
	c.SetCookie(
		"session_token", // name
		token,           // value
		24*3600,         // max age in seconds (24 hours)
		"/",             // path
		"",              // domain
		true,            // secure
		true,            // httpOnly
	)
	c.Redirect(http.StatusFound, "/")
}

func LogoutHandler(c *gin.Context) {
	c.SetCookie("session_token", "", -1, "/", "", false, false)
	c.Redirect(http.StatusFound, "/signup")
}

func loginUser(c *gin.Context) {
	// Check if user is already logged in
	cookie, err := c.Cookie("session_token")
	if err == nil {
		if _, _, err := jwt.ParseToken(cookie); err == nil {
			c.Redirect(http.StatusFound, "/")
			return
		}
	}

	username := c.PostForm("email")
	password := c.PostForm("password")

	var user db.User
	if err := db.DB.Where("username = ?", username).First(&user).Error; err != nil {
		c.Redirect(http.StatusFound, "/login?err=wrong-credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		c.Redirect(http.StatusFound, "/login?err=wrong-credentials")
		return
	}

	token, err := jwt.GenerateSecureToken(user.Id, user.Role)
	if err != nil {
		c.Redirect(http.StatusFound, "/login?err=token-error")
		return
	}

	c.SetCookie("session_token", token, 24*3600, "/", "", true, true)
	c.Redirect(http.StatusFound, "/")
}

func comparePasswords(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}
