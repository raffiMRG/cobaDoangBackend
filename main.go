package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	folderController "web_backend/Controller/FolderControllers"
	conn "web_backend/Model/Connection"
)

func main() {

	// Memuat file .env
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Error loading .env file: %v", err)
	}

	// Membaca variabel dari lingkungan
	appPort := os.Getenv("APP_PORT")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")

	// Koneksi ke MySQL
	_, err = conn.ConnectMySQL(dbUser, dbPass, dbHost, dbPort, dbName)
	// conn.ConnectMySQL(username, password, host, dbname)
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	r := gin.Default()

	r.GET("/update", folderController.UpdateAndInsert)
	// r.GET("/folders", folderController.DisplayAllDataFolder)
	r.GET("/folders", folderController.GetDataById)

	r.POST("/folders", folderController.MoveRow)

	r.GET("/filteredDatas", folderController.GetFilteredData)

	r.Run(":" + appPort)
}
