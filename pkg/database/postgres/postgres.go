package postgres

import (
    "fmt"
    "log"
    "os"
    "time"

    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

func NewPostgresDB() (*gorm.DB, error) {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        return nil, fmt.Errorf("DATABASE_URL not set in .env")
    }

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info), // Показывать SQL запросы
        NowFunc: func() time.Time {
            return time.Now().UTC() // Устанавливаем UTC
        },
        PrepareStmt: true, // Кеширование подготовленных операторов
    })
    if err != nil {
        log.Printf("NewPostgresDB: Failed to connect to database: %v", err)
        return nil, err
    }

    log.Println("Connected to database")
    return db, nil
}

func AutoMigrate(db *gorm.DB, models ...interface{}) error {
    err := db.AutoMigrate(models...)
    if err != nil {
        log.Printf("AutoMigrate: Failed to auto migrate: %v", err)
    }
    return err
}