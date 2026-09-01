package AuthController

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "web_backend/DTO"
	model "web_backend/Model"
	connection "web_backend/Model/Connection"
	"web_backend/Repository/AuthRepositorys"
)

func Login(c *gin.Context) {
	var request dto.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	db := connection.DB
	user, err := AuthRepositorys.FindUserByUsername(db, request.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.BaseResponseModel{
			CodeResponse:  401,
			HeaderMessage: "Unauthorized",
			Message:       "invalid username or password",
			Data:          nil,
		})
		return
	}

	if err := AuthRepositorys.VerifyPassword(user.Password, request.Password); err != nil {
		c.JSON(http.StatusUnauthorized, model.BaseResponseModel{
			CodeResponse:  401,
			HeaderMessage: "Unauthorized",
			Message:       "invalid username or password",
			Data:          nil,
		})
		return
	}

	accessToken, err := AuthRepositorys.GenerateAccessToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponseModel{
			CodeResponse:  500,
			HeaderMessage: "Error",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	refreshToken, err := AuthRepositorys.GenerateAndStoreRefreshToken(db, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponseModel{
			CodeResponse:  500,
			HeaderMessage: "Error",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponseModel{
		CodeResponse:  200,
		HeaderMessage: "Success",
		Message:       "login successful",
		Data: dto.LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	})
}

func Refresh(c *gin.Context) {
	var request dto.RefreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	db := connection.DB
	userID, err := AuthRepositorys.ValidateRefreshToken(db, request.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.BaseResponseModel{
			CodeResponse:  401,
			HeaderMessage: "Unauthorized",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	accessToken, err := AuthRepositorys.GenerateAccessToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponseModel{
			CodeResponse:  500,
			HeaderMessage: "Error",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponseModel{
		CodeResponse:  200,
		HeaderMessage: "Success",
		Message:       "token refreshed",
		Data: gin.H{
			"access_token": accessToken,
		},
	})
}

func Logout(c *gin.Context) {
	var request dto.RefreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	if err := AuthRepositorys.RevokeRefreshToken(connection.DB, request.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponseModel{
			CodeResponse:  500,
			HeaderMessage: "Error",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponseModel{
		CodeResponse:  200,
		HeaderMessage: "Success",
		Message:       "logged out",
		Data:          nil,
	})
}
