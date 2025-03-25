// Файл: internal/handlers/medicine_handler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"pharmacy-api/internal/models"      // Добавьте импорт вашей модели Medicine
	"pharmacy-api/internal/services" // Импорт сервиса
)

// MedicineHandler - структура для обработчиков medicine
type MedicineHandler struct {
	medicineService services.MedicineService // Change to services.MedicineService (not pointer)
	kafkaProducer   *kafka.Producer
	kafkaTopic      string
}

// NewMedicineHandler создает новый экземпляр MedicineHandler
func NewMedicineHandler(medicineService services.MedicineService, kafkaProducer *kafka.Producer, kafkaTopic string) *MedicineHandler {
	return &MedicineHandler{
		medicineService: medicineService,
		kafkaProducer:   kafkaProducer,
		kafkaTopic:      kafkaTopic,
	}
}

// CreateMedicine - создает новое лекарство
func (h *MedicineHandler) CreateMedicine(c *gin.Context) {
	// 1. Получаем данные запроса
	var medicine models.Medicine
	if err := c.ShouldBindJSON(&medicine); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. Создаем лекарство (с помощью medicineService)
	createdMedicine, err := h.medicineService.CreateMedicine(medicine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create medicine"})
		return
	}

	// 3. Отправляем сообщение в Kafka
	h.sendMedicineEventToKafka("medicine.created", int(createdMedicine.ID), c.GetString("username")) // Предположим, что имя пользователя есть в контексте

	// 4. Отправляем ответ
	c.JSON(http.StatusCreated, createdMedicine)
}

// GetMedicineByID - получает лекарство по ID
func (h *MedicineHandler) GetMedicineByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	medicine, err := h.medicineService.GetMedicineByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get medicine"})
		return
	}

	c.JSON(http.StatusOK, medicine)
}

// GetAllMedicines - получает список всех лекарств
func (h *MedicineHandler) GetAllMedicines(c *gin.Context) {
	medicines, err := h.medicineService.GetAllMedicines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get all medicines"})
		return
	}
	c.JSON(http.StatusOK, medicines)
}

// UpdateMedicine - обновляет информацию о лекарстве
func (h *MedicineHandler) UpdateMedicine(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
        return
    }
    var medicine models.Medicine
    if err := c.ShouldBindJSON(&medicine); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    updatedMedicine, err := h.medicineService.UpdateMedicine(id, medicine)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update medicine"})
        return
    }

    // Проверяем, что updatedMedicine не является пустым значением
    if updatedMedicine.ID == 0 { // Или другая подходящая проверка
        log.Println("updatedMedicine has zero value!")
        return // Или обработайте ошибку другим способом
    }

    h.sendMedicineEventToKafka("medicine.updated", int(updatedMedicine.ID), c.GetString("username")) // Предположим, что имя пользователя есть в контексте

    c.JSON(http.StatusOK, updatedMedicine)
}

// DeleteMedicine - удаляет лекарство
func (h *MedicineHandler) DeleteMedicine(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	err = h.medicineService.DeleteMedicine(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete medicine"})
		return
	}
	h.sendMedicineEventToKafka("medicine.deleted", id, c.GetString("username")) // Предположим, что имя пользователя есть в контексте

	c.Status(http.StatusNoContent)
}

//sendMedicineEventToKafka  отправляет сообщение
func (h *MedicineHandler) sendMedicineEventToKafka(event string, medicineID int, username string) {

	message := map[string]interface{}{
		"event":       event,
		"timestamp":   time.Now().Format(time.RFC3339),
		"medicine_id": medicineID,
		"user":        username,
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal medicine event: %s", err)
		return
	}
	err = h.produceMessage(h.kafkaTopic, string(messageJSON))
	if err != nil {
		log.Printf("Failed to send medicine event to Kafka: %s", err)
	}

}

// produceMessage отправляет сообщение в Kafka (вынесено для удобства)
func (h *MedicineHandler) produceMessage(topic string, message string) error {
	err := h.kafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte(message),
	}, nil)

	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}