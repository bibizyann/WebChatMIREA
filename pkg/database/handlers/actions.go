package handlers

import (
	"WebChatMIREA/pkg/database"
	"crypto/rand"
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/smtp"
	"time"
)

func SignUp(c *gin.Context) {
	var body struct {
		Email    string
		Password string
		Username string
	}

	if c.Bind(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse body"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to hash password"})
		return
	}
	user := database.Users{Email: body.Email, PasswordHash: string(hash), Username: body.Username}
	result := database.DB.Create(&user)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate a JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.Id,
		"exp": time.Now().Add(time.Hour * 24 * 30).Unix(),
	})

	var tokenString string
	tokenString, err = token.SignedString([]byte("SECRET"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create token"})
		return
	}

	// send it back
	c.SetSameSite(http.SameSiteLaxMode)
	// secure = True при деплое
	c.SetCookie("Authorization", tokenString, 3600*24*30, "", "", true, true)
	c.JSON(http.StatusOK, gin.H{})
}

func Login(c *gin.Context) {
	// Get the email and pass off req body
	var body struct {
		Email    string
		Password string
	}

	if c.Bind(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse body"})
		return
	}

	// Look up requested user
	var user database.Users
	database.DB.First(&user, "email = ?", body.Email)

	if user.Id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email or password"})
		return
	}

	// Compare sent in pass with saved user pass hash
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email or password"})
		return
	}
	// Generate a JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.Id,
		"exp": time.Now().Add(time.Hour * 24 * 30).Unix(),
	})

	var tokenString string
	tokenString, err = token.SignedString([]byte("SECRET"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create token"})
		return
	}

	// send it back
	c.SetSameSite(http.SameSiteLaxMode)
	// secure = True при деплое
	c.SetCookie("Authorization", tokenString, 3600*24*30, "", "", true, true)
	c.JSON(http.StatusOK, gin.H{})
}

func Validate(c *gin.Context) {
	user, _ := c.Get("user")

	// user.(database.Users).Id - how to use user

	c.JSON(200, gin.H{
		"message": user,
	})
}

func Logout(c *gin.Context) {
	c.SetCookie("Authorization", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func SendEmail(c *gin.Context, email string, password string) error {
	to := []string{email}

	host := "smtp.mailersend.net"
	port := "587"
	address := host + ":" + port

	// auth info
	auth := smtp.PlainAuth("", "MS_QqZqCB@trial-3yxj6ljr9d54do2r.mlsender.net", "HvpyrwqhnZMXZGZE", host)
	msg := []byte("To: " + email + "\r\n" +
		"Subject: Your New Password\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n" +
		"New password: " + password + "\r\n")

	from := "MS_QqZqCB@trial-3yxj6ljr9d54do2r.mlsender.net"
	err := smtp.SendMail(address, auth, from, to, msg)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to send email"})
		return err
	}
	c.JSON(http.StatusOK, gin.H{})
	return nil
}

func GetUserByEmail(c *gin.Context, email string) (*database.Users, error) {
	var user database.Users
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return nil, err
	}
	return &user, nil
}

// Генерация случайного пароля
func generateRandomPassword(length int) (string, error) {
	// Создаем байтовый массив заданной длины
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Преобразуем байты в строку в формате base64
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

func NewRandomPassword(c *gin.Context) (pass string, hashedPassword string, err error) {
	// Генерация случайного пароля длиной 12 символов
	password, err := generateRandomPassword(12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate password"})
		return "", "", err
	}

	// Хэширование пароля
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to hash password"})
		return "", "", err
	}

	// Возвращаем хэшированный пароль
	return password, string(hash), nil
}

// PasswordRecoveryPost Восстановление пароля
func PasswordRecoveryPost(c *gin.Context) {
	var body struct {
		Email string
	}

	if c.Bind(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse body"})
		return
	}

	// check if user exists in DB
	user, err := GetUserByEmail(c, body.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to send recovery email"})
		return
	}

	// Create new pass + update pass in DB
	pass, passHash, err := NewRandomPassword(c)
	user.PasswordHash = passHash
	database.DB.Save(&user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to generate new password"})
		return
	}
	err = SendEmail(c, body.Email, pass)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to send recovery email"})
		return
	}

}
