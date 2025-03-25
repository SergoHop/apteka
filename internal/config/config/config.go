package config

//Код config.LoadConfig() пытается загрузить конфигурацию из файла .env. Если это не удается (например, файл .env отсутствует),
//и при этом приложение не запущено в production-окружении, то в консоль выводится сообщение об ошибке. 


import (
    "log"
    "os"

    "github.com/joho/godotenv"
)

func LoadConfig() {
    err := godotenv.Load()
    if err != nil && os.Getenv("ENVIRONMENT") != "production" {
        log.Println("Error loading .env file") // Не критическая ошибка, если в production используются переменные окружения
    }
}