package handlers

import (
	
	"fmt"
	"log"
	"net/http"
	
	"time"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"pharmacy-api/internal/services"
)

// RegisterRequest структура для данных регистрации пользователя
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginRequest структура для данных логина пользователя
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthHandler структура для обработки запросов аутентификации
type AuthHandler struct {
	authService   *services.AuthService
	kafkaProducer *kafka.Producer // Добавляем Kafka Producer
	kafkaTopic    string          // Добавляем топик Kafka для логина
	kafkaRegTopic string          // Добавляем топик Kafka для регистрации
}

// NewAuthHandler создает новый экземпляр AuthHandler
func NewAuthHandler(authService *services.AuthService, kafkaProducer *kafka.Producer, kafkaTopic string, kafkaRegTopic string) *AuthHandler {
	return &AuthHandler{
		authService:   authService,
		kafkaProducer: kafkaProducer,
		kafkaTopic:    kafkaTopic,
		kafkaRegTopic: kafkaRegTopic,
	}
}

// Register обрабатывает запрос регистрации
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.authService.Register(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	// Отправляем сообщение о регистрации в Kafka
	h.sendRegistrationEventToKafka(req.Username, "Registration successful")

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

// Login обрабатывает запрос логина
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		// Отправляем сообщение о неудачной попытке логина в Kafka
		h.sendLoginEventToKafka(req.Username, false, "Invalid credentials")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Отправляем сообщение об успешном логине в Kafka
	h.sendLoginEventToKafka(req.Username, true, "Login successful")

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// Структура для события логина в Kafka
type LoginEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Username    string    `json:"username"`
	Success     bool      `json:"success"`
	Description string    `json:"description"`
}

// Структура для события регистрации в Kafka
type RegistrationEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Username  string    `json:"username"`
	Message   string    `json:"message"`
}

// sendLoginEventToKafka отправляет сообщение о событии логина в Kafka
func (h *AuthHandler) sendLoginEventToKafka(username string, success bool, description string) {
	event := LoginEvent{
		Timestamp:   time.Now(),
		Username:    username,
		Success:     success,
		Description: description,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal login event: %s\n", err)
		return
	}

	err = h.produceMessage(h.kafkaTopic, string(eventJSON))
	if err != nil {
		log.Printf("Failed to send login event to Kafka: %s\n", err)
	}
}

// sendRegistrationEventToKafka отправляет сообщение о событии регистрации в Kafka
func (h *AuthHandler) sendRegistrationEventToKafka(username string, message string) {
	event := RegistrationEvent{
		Timestamp: time.Now(),
		Username:  username,
		Message:   message,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal registration event: %s\n", err)
		return
	}

	err = h.produceMessage(h.kafkaRegTopic, string(eventJSON))
	if err != nil {
		log.Printf("Failed to send registration event to Kafka: %s\n", err)
	}
}

// produceMessage отправляет сообщение в Kafka (вынесено для удобства)
func (h *AuthHandler) produceMessage(topic string, message string) error {
	err := h.kafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte(message),
	}, nil)

	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}