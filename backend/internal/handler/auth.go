package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AboAuther/RGPerp/backend/internal/pkg/response"
	"github.com/AboAuther/RGPerp/backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type challengeRequest struct {
	WalletAddress string `json:"wallet_address" binding:"required"`
	Domain        string `json:"domain"`
	ChainID       uint64 `json:"chain_id"`
}

type loginRequest struct {
	WalletAddress string `json:"wallet_address" binding:"required"`
	Nonce         string `json:"nonce" binding:"required"`
	Signature     string `json:"signature" binding:"required"`
}

func (h *AuthHandler) Challenge(c *gin.Context) {
	var req challengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMsg(c, http.StatusBadRequest, 90002, "invalid request payload")
		return
	}

	result, err := h.authService.CreateChallenge(service.ChallengeInput{
		WalletAddress: req.WalletAddress,
		Domain:        req.Domain,
		ChainID:       req.ChainID,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, result)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMsg(c, http.StatusBadRequest, 90002, "invalid request payload")
		return
	}

	result, err := h.authService.Login(service.LoginInput{
		WalletAddress: req.WalletAddress,
		Nonce:         req.Nonce,
		Signature:     req.Signature,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, result)
}
