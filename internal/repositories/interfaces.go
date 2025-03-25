package repositories

import "pharmacy-api/internal/models"

type UserRepository interface {
    Create(user *models.User) error
    GetByUsername(username string) (*models.User, error)
    GetByID(id uint) (*models.User, error)
}

type MedicineRepository interface {
    Create(medicine models.Medicine) (models.Medicine, error)
    GetByID(id int) (models.Medicine, error)
    GetAll() ([]models.Medicine, error)
    Update(id int, medicine models.Medicine) (models.Medicine, error)
    Delete(id int) error
}
