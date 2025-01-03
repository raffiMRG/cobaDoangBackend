package FolderControllers

import (
	"net/http"
	"path/filepath"

	// "strconv"
	"strings"

	"github.com/gin-gonic/gin"

	dto "web_backend/DTO"
	model "web_backend/Model"
	connection "web_backend/Model/Connection"
	messageStatus "web_backend/Model/MessageStatus"

	// newFolder "web_backend/Model/NewFolder"
	// tbFolder "web_backend/Model/TbFolder"
	"web_backend/Repository/FolderRepositorys"
)

func UpdateAndInsert(c *gin.Context) {
	root := "../folder0/folder1/" // Change to your desired root directory
	folders, err := FolderRepositorys.ScanFolders(root)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Process folders and insert into the database
	var messages []messageStatus.Message
	for _, folder := range folders {
		unixStylePath := strings.ReplaceAll(folder, "\\", "/")
		finalFolderName := filepath.Base(unixStylePath)

		db := connection.DB

		// Insert into database and collect status
		err := FolderRepositorys.InsertFolder(db, finalFolderName)
		if err != nil {
			messages = append(messages, messageStatus.Message{
				FolderName: finalFolderName,
				Status:     "error",
				Error:      err.Error(),
			})
		} else {
			messages = append(messages, messageStatus.Message{
				FolderName: finalFolderName,
				Status:     "success",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
	})
}

func DisplayAllDataFolder(c *gin.Context) {
	// var response model.BaseResponseModel

	response := FolderRepositorys.GetAllData("folders")
	if response.CodeResponse != 200 {
		c.JSON(response.CodeResponse, response)
		return
	}

	// use the parse data
	// fmt.Printf("Recived data : %+v\n", )
	c.JSON(http.StatusOK, response)
}

func GetDataById(c *gin.Context) {
	var response model.BaseResponseModel

	strId := c.Query("id")

	response = FolderRepositorys.GetDataFromId("folders", strId)
	if response.CodeResponse != 200 {
		c.JSON(response.CodeResponse, response)
		return
	}
	c.JSON(http.StatusOK, response)
}

func MoveRow(c *gin.Context) {
	// var request tbFolder.Folder
	var request dto.InputDataReq
	var response model.BaseResponseModel

	if err := c.ShouldBindJSON(&request); err != nil {
		response = model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       err.Error(),
			Data:          nil,
		}
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// for _, ID := range request.IDS{
	// func MoveRows(sourceTable string, targetTable string) model.BaseResponseModel {
	response = FolderRepositorys.MoveRows(request.IDS, "folders", "new_folder")
	// response = FolderRepositorys.MoveRows(int(ID), "folders", "new_folder")
	if response.CodeResponse != 200 {
		c.JSON(response.CodeResponse, response)
		return
	}
	// }

	c.JSON(http.StatusOK, response)
}

func GetFilteredData(c *gin.Context) {
	// var response model.BaseResponseModel

	// strId := c.Query("id")

	response := FolderRepositorys.FilteredData("folders", "new_folder")
	// response := FolderRepositorys.GetDataFromId("folders", strId)
	if response.CodeResponse != 200 {
		c.JSON(response.CodeResponse, response)
		return
	}
	c.JSON(http.StatusOK, response)
}
