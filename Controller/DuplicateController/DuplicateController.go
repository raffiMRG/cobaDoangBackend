package DuplicateController

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	dto "web_backend/DTO"
	model "web_backend/Model"
	"web_backend/Repository/DuplicateRepositorys"
)

// @Summary List kandidat duplikat yang belum direview
// @Tags duplicates
// @Produce json
// @Param page query int false "Halaman" default(1)
// @Param limit query int false "Item per halaman" default(20)
// @Success 200 {object} object{page=int,per_page=int,total_data=int,data=[]dto.DuplicateCandidateItem}
// @Security BearerAuth
// @Router /duplicates [get]
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

// @Summary Bandingkan halaman existing vs incoming
// @Tags duplicates
// @Produce json
// @Param id path int true "duplicate_candidates.id"
// @Success 200 {object} model.BaseResponseModel{Data=dto.DuplicateCompareResponse}
// @Failure 400 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /duplicates/{id}/compare [get]
func Compare(c *gin.Context) {
	id := c.Param("id")
	resp, err := DuplicateRepositorys.GetCandidateForCompare(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Error", Message: err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "ok", Data: resp})
}

// @Summary Selesaikan kandidat duplikat
// @Description action: new_title (butuh new_title), merge (butuh merge_mode replace|append), keep_existing.
// @Tags duplicates
// @Accept json
// @Produce json
// @Param id path int true "duplicate_candidates.id"
// @Param body body dto.ResolveDuplicateRequest true "Aksi"
// @Success 200 {object} model.BaseResponseModel
// @Failure 400 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /duplicates/{id}/resolve [post]
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
	case "keep_existing":
		err = DuplicateRepositorys.ResolveKeepExisting(id)
	default:
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "action must be 'new_title', 'merge', or 'keep_existing'", Data: nil})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "resolved", Data: nil})
}

// @Summary Konversi tabrakan nama lama jadi kandidat duplikat
// @Tags duplicates
// @Produce json
// @Success 200 {object} model.BaseResponseModel{Data=dto.BackfillResult}
// @Security BearerAuth
// @Router /duplicates/backfill [post]
func Backfill(c *gin.Context) {
	result, err := DuplicateRepositorys.BackfillExistingCollisions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "ok", Data: result})
}
