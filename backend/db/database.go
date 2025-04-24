package db

import (
	"fmt"
	"log"
	"time"

	"github.com/Amirali-Amirifar/yeetcode/backend/config"
	"github.com/Amirali-Amirifar/yeetcode/backend/utils/roles"
	"golang.org/x/crypto/bcrypt"

	_ "golang.org/x/crypto/bcrypt"
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
	// Relationships
	Questions   []*Question   `gorm:"foreignKey:OwnerId"`
	Submissions []*Submission `gorm:"foreignKey:UserId"`
	Stats       *UserStats    `gorm:"foreignKey:UserId"`
}

type Question struct {
	Id          uint   `gorm:"primaryKey"`
	Title       string `gorm:"not null"`
	Statement   string `gorm:"not null"`
	TimeLimit   int    `gorm:"not null"`        // milliseconds
	MemoryLimit int    `gorm:"not null"`        // megabytes
	Difficulty  string `gorm:"not null"`        // 'easy', 'medium', 'hard'
	Status      string `gorm:"default:'draft'"` // 'draft' or 'published'
	OwnerId     uint   `gorm:"not null"`        // creator of the question
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
	// Relationships
	Owner       *User         `gorm:"foreignKey:OwnerId"`
	Submissions []*Submission `gorm:"foreignKey:QuestionId"`
	TestCases   []*TestCase   `gorm:"foreignKey:QuestionId"`
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
	// Relationships
	User     *User     `gorm:"foreignKey:UserId"`
	Question *Question `gorm:"foreignKey:QuestionId"`
}

type UserStats struct {
	Id          uint    `gorm:"primaryKey"`
	UserId      uint    `gorm:"not null"`
	SolvedCount int     `gorm:"default:0"`
	SuccessRate float64 `gorm:"default:0.0"`
	UpdatedAt   time.Time
	// Relationships
	User *User `gorm:"foreignKey:UserId"`
}

type TestCase struct {
	Id         uint   `gorm:"primaryKey"`
	QuestionId uint   `gorm:"not null"`
	Input      string `gorm:"not null"`
	Output     string `gorm:"not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	// Relationships
	Question *Question `gorm:"foreignKey:QuestionId"`
}

// createInitialAdmin creates the initial admin user if it doesn't exist
func createInitialAdmin() error {
	var admin User
	result := DB.Where("username = ?", "admin@admin.com").First(&admin)

	if result.Error == nil {
		// Admin already exists
		return nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return result.Error
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("12345678"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Create admin user
	admin = User{
		Username: "admin@admin.com",
		Password: string(hashedPassword),
		Role:     roles.RoleAdmin,
	}

	if err := DB.Create(&admin).Error; err != nil {
		return err
	}

	log.Println("Initial admin user created successfully")
	return nil
}

// Init initializes the database and runs the migrations
func Init() *gorm.DB {
	conf := config.GetConfig()
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		conf.DbHost, conf.DbPort, conf.DbUser, conf.DbPassword, conf.DbName, conf.DbSSLMode)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	err = DB.AutoMigrate(&User{}, &Question{}, &Submission{}, &UserStats{}, &TestCase{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Create initial admin user
	if err := createInitialAdmin(); err != nil {
		log.Fatal("Failed to create initial admin user:", err)
	}

	return DB
}
