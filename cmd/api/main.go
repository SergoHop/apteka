package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"pharmacy-api/internal/config/config"
	"pharmacy-api/internal/handlers"
	"pharmacy-api/internal/middleware"
	postgres "pharmacy-api/internal/repositories/postgres" // Alias импорта
	"pharmacy-api/internal/services"
	dbpkg "pharmacy-api/pkg/database/postgres" // Изменен импорт
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	//"github.com/segmentio/kafka-go"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

const (
	kafkaBrokersEnv     = "KAFKA_BROKERS"
	kafkaTopicEnv       = "KAFKA_TOPIC"
	kafkaGroupIDEnv     = "KAFKA_GROUP_ID"
	kafkaLoginTopicEnv  = "KAFKA_LOGIN_TOPIC"
	kafkaRegTopicEnv    = "KAFKA_REGISTRATION_TOPIC"
	dbURL               = "DATABASE_URL"
)

func main() {
	// Загрузка конфигурации из .env
	config.LoadConfig()

	// Подключение к базе данных
	db, err := dbpkg.NewPostgresDB() // Использование функции из пакета
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
		return
	}
	log.Println("Database connection successful")

	// Инициализация репозиториев
	userRepository := postgres.NewUserRepository(db) // Использование postgres.NewUserRepository
	medicineRepo := postgres.NewMedicineRepository(db) // Инициализируем репозиторий для лекарств

	// Инициализация сервисов
	authService := services.NewAuthService(userRepository)
	medicineService := services.NewMedicineService(medicineRepo) // Инициализируем сервис для лекарств

	// Load Kafka Configuration (Consumer)
	kafkaBrokers := strings.Split(os.Getenv(kafkaBrokersEnv), ",")
	if len(kafkaBrokers) == 0 {
		log.Fatalf("Error: %s environment variable not set", kafkaBrokersEnv)
	}

	kafkaTopic := os.Getenv(kafkaTopicEnv)
	if kafkaTopic == "" {
		log.Fatalf("Error: %s environment variable not set", kafkaTopicEnv)
	}

	kafkaGroupID := os.Getenv(kafkaGroupIDEnv)
	if kafkaGroupID == "" {
		kafkaGroupID = "my-group" // Установка значения по умолчанию
		log.Printf("Using default Kafka group ID: %s", kafkaGroupID)
	}

	// Load Kafka Configuration (Producer)
	kafkaLoginTopic := os.Getenv(kafkaLoginTopicEnv)
	if kafkaLoginTopic == "" {
		kafkaLoginTopic = "login-events" // Установка значения по умолчанию
		log.Printf("Using default Kafka login topic: %s", kafkaLoginTopic)
	}

	kafkaRegTopic := os.Getenv(kafkaRegTopicEnv)
	if kafkaRegTopic == "" {
		kafkaRegTopic = "registration-events" // Установка значения по умолчанию
		log.Printf("Using default Kafka registration topic: %s", kafkaRegTopic)
	}

	// Create Kafka Producer
	kafkaProducer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": strings.Join(kafkaBrokers, ","),
	})
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %s", err)
	}
	defer kafkaProducer.Close()

	// Создаем контекст с возможностью отмены
	ctx, cancel := context.WithCancel(context.Background())

	// Обработка сигналов завершения (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel() // Отмена контекста для завершения consumer-а
	}()

	// Consumer
	consumer := createKafkaConsumer(kafkaBrokers, kafkaTopic, kafkaGroupID)
	if consumer != nil {  // Добавлена проверка на nil
		defer func() {
			if err := consumer.Close(); err != nil {
				log.Printf("Error closing consumer: %v", err)
			}
		}()

		// Запускаем consumer в отдельной горутине
		go consumeMessages(ctx, consumer)
	}

	// Инициализация обработчиков
	    // Инициализация обработчиков
		medicineMedicineTopic := os.Getenv("KAFKA_MEDICINE_TOPIC")
		authHandler := handlers.NewAuthHandler(authService, kafkaProducer, kafkaLoginTopic, kafkaRegTopic)
		medicineHandler := handlers.NewMedicineHandler(medicineService, kafkaProducer, medicineMedicineTopic)// Инициализируем обработчик для лекарств
	
	// Настройка Gin роутера
	router := gin.Default()

	// Middleware
	router.Use(gin.Recovery())                         // Включаем recovery middleware
	router.Use(middleware.CORSMiddleware())           // Включаем CORS middleware
	//router.Use(middleware.RequestLogger())

	// Auth routes
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
	}

	// Medicine routes - подключение группы маршрутов для лекарств
    authorized := router.Group("/medicines")
    authorized.Use(middleware.AuthMiddleware(authService))
    {
        authorized.POST("/", medicineHandler.CreateMedicine)
        authorized.GET("/:id", medicineHandler.GetMedicineByID)
        authorized.GET("/", medicineHandler.GetAllMedicines)
        authorized.PUT("/:id", medicineHandler.UpdateMedicine)
        authorized.DELETE("/:id", medicineHandler.DeleteMedicine)
    }

	// Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082" // Значение по умолчанию
	}
	log.Printf("Starting server on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func createKafkaConsumer(kafkaBrokers []string, kafkaTopic string, kafkaGroupID string) *kafka.Consumer {
	config := &kafka.ConfigMap{
		"bootstrap.servers": strings.Join(kafkaBrokers, ","),
		"group.id":          kafkaGroupID,
		"auto.offset.reset": "earliest", // Или "latest", в зависимости от ваших потребностей
	}

	consumer, err := kafka.NewConsumer(config)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %s", err)
		return nil // Или обработайте ошибку другим способом
	}

	err = consumer.SubscribeTopics([]string{kafkaTopic}, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to topic: %s", err)
		return nil // Или обработайте ошибку другим способом
	}
	return consumer
}

func consumeMessages(ctx context.Context, consumer *kafka.Consumer) {
	defer consumer.Close()

	for {
		select {
		case <-ctx.Done():
			log.Println("Consumer stopped")
			return
		default:
			msg, err := consumer.ReadMessage(-1) // Timeout -1 означает бесконечное ожидание
			if err != nil {
				if errors.Is(err, context.Canceled) {
					log.Println("Consumer shutting down due to context cancellation")
					return
				}
				log.Printf("Error reading message: %v", err)
				continue
			}
			fmt.Printf("Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
			consumer.CommitMessage(msg) // Подтверждаем получение сообщения
		}
	}
}