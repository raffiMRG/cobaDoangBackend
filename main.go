package main

import (
	// "fmt"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	folderController "web_backend/Controller/FolderControllers"
	conn "web_backend/Model/Connection"
)

func main() {
	var err error

	// Memuat file .env
	err = godotenv.Load()
	if err != nil {
		fmt.Printf("Error loading .env file: %v", err)
	}

	// Membaca variabel dari lingkungan
	var appPort string = os.Getenv("APP_PORT")
	var dbHost string = os.Getenv("DB_APP_HOST")
	var dbPort string = os.Getenv("DB_PORT")
	var dbUser string = os.Getenv("DB_USER")
	var dbPass string = os.Getenv("DB_PASS")
	var dbName string = os.Getenv("DB_NAME")

	// fmt.Println("app port:", appPort)
	// fmt.Println("db host:", dbHost)
	// fmt.Println("db port:", dbPort)
	// fmt.Println("db user:", dbUser)
	// fmt.Println("db pass:", dbPass)
	// fmt.Println("db name:", dbName)

	// Koneksi ke MySQL
	_, err = conn.ConnectMySQL(dbUser, dbPass, dbHost, dbPort, dbName)
	// conn.ConnectMySQL(username, password, host, dbname)
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	r := gin.Default()

	// Atur middleware CORS
	r.Use(cors.New(cors.Config{
		// AllowOrigins:     []string{"http://localhost:3000"}, // sesuaikan origin frontend kamu
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Static("/new", "../new")
	r.Static("/sementara", "../sementara")

	r.GET("/update", folderController.UpdateAndInsert)
	r.GET("/folders", folderController.DisplayAllDataFolder)
	r.GET("/id/:id", folderController.GetDataById)

	r.POST("/folders", folderController.MoveRow)

	r.GET("/newFolders", folderController.DisplayDataNewfolder)

	r.GET("/filteredDatas", folderController.GetFilteredData)

	r.Run(":" + appPort)
}
