package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"pharmacy-api/internal/models"
	"pharmacy-api/internal/repositories"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/segmentio/kafka-go"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo    repositories.UserRepository
	jwtSecret   string
	kafkaWriter *kafka.Writer
	
}



func NewAuthService(userRepo repositories.UserRepository) *AuthService {
	brokersString := os.Getenv("KAFKA_BROKERS")
	if brokersString == "" {
		log.Fatalf("Error: KAFKA_BROKERS environment variable not set or empty")
	}
	brokers := strings.Split(brokersString, ",")
	log.Printf("Kafka Brokers: %v", brokers)

	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		log.Fatalf("Error: KAFKA_TOPIC environment variable not set")
	}


	// Создаем Kafka writer
	kafkaWriter := &kafka.Writer{
		Addr:      kafka.TCP(brokers...), // Используем Addr вместо Brokers
		Topic:     topic,
		Balancer:  &kafka.LeastBytes{},
	}

	return &AuthService{
		userRepo:    userRepo,
		jwtSecret:   os.Getenv("JWT_SECRET"),
		kafkaWriter: kafkaWriter,
	}
}

func (s *AuthService) Register(username, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &models.User{
		Username: username,
		Password: string(hashedPassword),
	}

	err = s.userRepo.Create(user)
	if err != nil {
		return err
	}

	// Отправка сообщения в Kafka после успешной регистрации
	err = s.SendRegistrationMessage(context.Background(), username)
	if err != nil {
		log.Printf("Failed to send registration message: %v", err)
	}

	return nil
}

func (s *AuthService) Login(username, password string) (string, error) {
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if user == nil {
		return "", errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	token, err := s.generateJWT(user)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) generateJWT(user *models.User) (string, error) {
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

type Claims struct {
	UserID   uint
	Username string
	jwt.RegisteredClaims
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func (s *AuthService) SendRegistrationMessage(ctx context.Context, username string) error {
	message := kafka.Message{
		Key:   []byte("registration"),
		Value: []byte(fmt.Sprintf("New user registered: %s", username)),
		Time:  time.Now(),
	}

	if s.kafkaWriter == nil {
		log.Println("Kafka writer is not initialized")
		return errors.New("kafka writer is not initialized")
	}

	err := s.kafkaWriter.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to write registration message: %w", err)
	}
	log.Printf("Sent registration message: %s\n", string(message.Value))
	return nil
}