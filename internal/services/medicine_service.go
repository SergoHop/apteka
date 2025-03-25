package services

import (
	"pharmacy-api/internal/models"
	"pharmacy-api/internal/repositories"
)

// MedicineService - интерфейс для сервиса medicine
type MedicineService interface {
	CreateMedicine(medicine models.Medicine) (models.Medicine, error)
	GetMedicineByID(id int) (models.Medicine, error)
	GetAllMedicines() ([]models.Medicine, error)
	UpdateMedicine(id int, medicine models.Medicine) (models.Medicine, error)
	DeleteMedicine(id int) error
}

type medicineService struct {
	medicineRepository repositories.MedicineRepository
}

// NewMedicineService создает новый экземпляр MedicineService
func NewMedicineService(medicineRepository repositories.MedicineRepository) MedicineService {
	return &medicineService{medicineRepository: medicineRepository}
}

// CreateMedicine создает новое лекарство
func (s *medicineService) CreateMedicine(medicine models.Medicine) (models.Medicine, error) {
	// Логика создания лекарства (например, валидация данных)
	return s.medicineRepository.Create(medicine)
}

// GetMedicineByID возвращает лекарство по ID
func (s *medicineService) GetMedicineByID(id int) (models.Medicine, error) {
	// Логика получения лекарства по ID
	return s.medicineRepository.GetByID(id)
}

// GetAllMedicines возвращает все лекарства
func (s *medicineService) GetAllMedicines() ([]models.Medicine, error) {
	// Логика получения всех лекарств
	return s.medicineRepository.GetAll()
}

// UpdateMedicine обновляет информацию о лекарстве
func (s *medicineService) UpdateMedicine(id int, medicine models.Medicine) (models.Medicine, error) {
	// Логика обновления лекарства (например, валидация данных)
	return s.medicineRepository.Update(id, medicine)
}

// DeleteMedicine удаляет лекарство
func (s *medicineService) DeleteMedicine(id int) error {
	// Логика удаления лекарства
	return s.medicineRepository.Delete(id)
}
