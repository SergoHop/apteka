package postgres

import (
	"pharmacy-api/internal/models"
	"pharmacy-api/internal/repositories" // Import the repositories package
	"gorm.io/gorm"
)

// medicineRepository implements the MedicineRepository interface
type medicineRepository struct {  //Renamed to medicineRepository
	db *gorm.DB
}

// NewMedicineRepository creates a new instance of MedicineRepository
func NewMedicineRepository(db *gorm.DB) repositories.MedicineRepository { // Implements the interface
	return &medicineRepository{db: db}
}

// Create creates a new medicine
func (r *medicineRepository) Create(medicine models.Medicine) (models.Medicine, error) {
	result := r.db.Create(&medicine)
	if result.Error != nil {
		return models.Medicine{}, result.Error // Return empty Medicine struct on error
	}
	return medicine, nil
}

// GetByID retrieves a medicine by ID
func (r *medicineRepository) GetByID(id int) (models.Medicine, error) { // Changed id type
	var medicine models.Medicine
	result := r.db.First(&medicine, id)
	if result.Error != nil {
		return models.Medicine{}, result.Error // Return empty Medicine struct on error
	}
	return medicine, nil
}

// GetAll retrieves all medicines
func (r *medicineRepository) GetAll() ([]models.Medicine, error) {
	var medicines []models.Medicine
	result := r.db.Find(&medicines)
	return medicines, result.Error
}

// Update updates an existing medicine
func (r *medicineRepository) Update(id int, medicine models.Medicine) (models.Medicine, error) {
	var existingMedicine models.Medicine
	result := r.db.First(&existingMedicine, id)
	if result.Error != nil {
		return models.Medicine{}, result.Error
	}
	existingMedicine.Name = medicine.Name
	existingMedicine.Price = medicine.Price // Предполагаю, что есть поле Price
	result = r.db.Save(&existingMedicine)
	return existingMedicine, result.Error
}

// Delete deletes a medicine by ID
func (r *medicineRepository) Delete(id int) error { //Changed id type
	result := r.db.Delete(&models.Medicine{}, id)
	return result.Error
}