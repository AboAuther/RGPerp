package handler

import (
	"strconv"

	"github.com/AboAuther/RGPerp/backend/internal/pkg/response"
	"github.com/AboAuther/RGPerp/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminService *service.AdminService
}

func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (h *AdminHandler) Overview(c *gin.Context) {
	data, err := h.adminService.GetOverview()
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

func (h *AdminHandler) HedgeTasks(c *gin.Context) {
	items, err := h.adminService.ListHedgeTasks(parseLimit(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h *AdminHandler) RiskSnapshots(c *gin.Context) {
	items, err := h.adminService.ListRiskSnapshots(parseLimit(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h *AdminHandler) Liquidations(c *gin.Context) {
	items, err := h.adminService.ListLiquidations(parseLimit(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h *AdminHandler) Alerts(c *gin.Context) {
	items, err := h.adminService.ListAlerts(parseLimit(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func parseLimit(c *gin.Context) int {
	raw := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}
