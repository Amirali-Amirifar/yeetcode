package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

type User struct {
	Id        uint   `gorm:"primaryKey"`
	Username  string `gorm:"unique;not null"`
	Password  string `gorm:"not null"`
	Role      string `gorm:"default:'user';not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Question struct {
	Id          uint   `gorm:"primaryKey"`
	Title       string `gorm:"not null"`
	Statement   string `gorm:"not null"`
	TimeLimit   int    `gorm:"not null"`        // milliseconds
	MemoryLimit int    `gorm:"not null"`        // megabytes
	Status      string `gorm:"default:'draft'"` // 'draft' or 'published'
	OwnerId     uint   `gorm:"not null"`        // creator of the question
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
}

type Submission struct {
	Id          uint   `gorm:"primaryKey"`
	Code        string `gorm:"not null"`
	Status      string `gorm:"default:'pending'"` // 'pending', 'compile_error', 'wrong_answer', etc.
	Output      string `gorm:""`
	Error       string `gorm:""`
	QuestionId  uint   `gorm:"not null"`
	UserId      uint   `gorm:"not null"`
	CreatedAt   time.Time
	ProcessedAt *time.Time
}

type UserStats struct {
	Id          uint    `gorm:"primaryKey"`
	UserId      uint    `gorm:"not null"`
	SolvedCount int     `gorm:"default:0"`
	SuccessRate float64 `gorm:"default:0.0"`
	UpdatedAt   time.Time
}

type TestCase struct {
	Id         uint   `gorm:"primaryKey"`
	QuestionId uint   `gorm:"not null"`
	Input      string `gorm:"not null"`
	Output     string `gorm:"not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Init initializes the database and runs the migrations
func Init() *gorm.DB {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	err = DB.AutoMigrate(&User{}, &Question{}, &Submission{}, &UserStats{}, &TestCase{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	return DB
}

