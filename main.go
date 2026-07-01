package main

import (
	// "fmt"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	AuthController "web_backend/Controller/AuthController"
	BackupController "web_backend/Controller/BackupController"
	Bookmarkcontroller "web_backend/Controller/BookmarkController"
	folderController "web_backend/Controller/FolderControllers"
	"web_backend/Middleware"
	conn "web_backend/Model/Connection"
	"web_backend/Repository/AuthRepositorys"
)

func main() {
	var err error

	// // Tentukan environment
	// env := os.Getenv("APP_ENV")
	// if env == "" {
	// 	env = "development" // default
	// }

	// // Tentukan file env
	// envFile := ".env." + env

	// // Load env file
	// err = godotenv.Load(envFile)
	// if err != nil {
	// 	fmt.Printf("Warning: could not load %s file: %v\n", envFile, err)
	// }

	// fmt.Println("Running in:", env)

	// // Ambil environment variables
	// appPort := os.Getenv("APP_PORT")
	// dbHost := os.Getenv("DB_APP_HOST")
	// dbPort := os.Getenv("DB_PORT")
	// dbUser := os.Getenv("DB_USER")
	// dbPass := os.Getenv("DB_PASS")
	// dbName := os.Getenv("DB_NAME")
	// srcPath := os.Getenv("SRC_DIR")
	// dstPath := os.Getenv("DST_DIR")

	// // Debug (opsional)
	// fmt.Println("App Port:", appPort)
	// fmt.Println("DB Host:", dbHost)

	// ========================
	// Memuat file .env
	// err = godotenv.Load(".env.development")
	err = godotenv.Load()
	if err != nil {
		fmt.Printf("Error loading .env file: %v", err)
	}

	createUserFlag := flag.String("create-user", "", "username for a one-off admin user to create, then exit")
	flag.Parse()

	// Membaca variabel dari lingkungan
	var appPort string = os.Getenv("APP_PORT")
	var dbHost string = os.Getenv("DB_APP_HOST")
	var dbPort string = os.Getenv("DB_PORT")
	var dbUser string = os.Getenv("DB_USER")
	var dbPass string = os.Getenv("DB_PASS")
	var dbName string = os.Getenv("DB_NAME")
	var srcPath string = os.Getenv("SRC_DIR")
	var dstPath string = os.Getenv("DST_DIR")

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

	// One-off admin bootstrap: `./app --create-user <username> <password>`
	// creates the user then exits, without ever exposing a public
	// registration endpoint.
	if *createUserFlag != "" {
		args := flag.Args()
		if len(args) < 1 {
			log.Fatal("usage: app --create-user <username> <password>")
		}
		if err := AuthRepositorys.CreateUser(conn.DB, *createUserFlag, args[0]); err != nil {
			log.Fatal("Failed to create user:", err)
		}
		fmt.Println("User created successfully:", *createUserFlag)
		return
	}

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET environment variable is required")
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

	// Thumbnails served as plain <img src> by the frontend — can't carry a
	// Bearer header, so these stay public unlike the JSON/API routes below.
	r.Static("/new", dstPath)
	r.Static("/sementara", srcPath)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/login", AuthController.Login)
	r.POST("/refresh", AuthController.Refresh)

	protected := r.Group("")
	protected.Use(Middleware.RequireAuth())
	{
		protected.POST("/logout", AuthController.Logout)

		protected.GET("/update", folderController.UpdateAndInsert)
		protected.GET("/folders", folderController.DisplayAllDataFolder)
		protected.GET("/id/:id", folderController.GetDataById)

		// protected.POST("/folders", folderController.MoveRow)
		protected.POST("/folders", folderController.MoveRowAndTrack)
		protected.GET("/folders/progress/:taskID", folderController.FolderProgress)

		protected.GET("/newFolders", folderController.DisplayDataNewfolder)

		protected.GET("/filteredDatas", folderController.GetFilteredData)
		protected.GET("/search", folderController.SearchFolders)

		bookmarks := protected.Group("/bookmarks")
		{
			bookmarks.GET("", Bookmarkcontroller.GetBookmarks)
			bookmarks.GET("/:id", Bookmarkcontroller.GetBookmark)
			bookmarks.POST("", Bookmarkcontroller.ToggleBookmark)
			bookmarks.DELETE("/:id", Bookmarkcontroller.DeleteBookmark)
		}

		protected.GET("/export", BackupController.Export)
		protected.POST("/import", BackupController.Import)
	}

	r.Run(":" + appPort)
}
