package DuplicateController

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	dto "web_backend/DTO"
	model "web_backend/Model"
	"web_backend/Repository/DuplicateRepositorys"
)

func ListPending(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	items, total, err := DuplicateRepositorys.ListPending(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"page":       page,
		"per_page":   limit,
		"total_data": total,
		"data":       items,
	})
}

func Compare(c *gin.Context) {
	id := c.Param("id")
	resp, err := DuplicateRepositorys.GetCandidateForCompare(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Error", Message: err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "ok", Data: resp})
}

func Resolve(c *gin.Context) {
	var request dto.ResolveDuplicateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: err.Error(), Data: nil})
		return
	}

	id := c.Param("id")
	var err error
	switch request.Action {
	case "new_title":
		if request.NewTitle == "" {
			c.JSON(http.StatusBadRequest, model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "new_title is required", Data: nil})
			return
		}
		err = DuplicateRepositorys.ResolveNewTitle(id, request.NewTitle)
	case "merge":
		if request.MergeMode == "" {
			c.JSON(http.StatusBadRequest, model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "merge_mode is required", Data: nil})
			return
		}
		err = DuplicateRepositorys.ResolveMerge(id, request.MergeMode)
	default:
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "action must be 'new_title' or 'merge'", Data: nil})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "resolved", Data: nil})
}

func Backfill(c *gin.Context) {
	result, err := DuplicateRepositorys.BackfillExistingCollisions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "ok", Data: result})
}
