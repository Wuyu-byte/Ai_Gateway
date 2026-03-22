package api

import (
	"net/http"
	"strconv"

	"ai-gateway/middleware"
	"ai-gateway/scheduler"
	"ai-gateway/service"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	usageStatsService *service.UsageStatsService
	scheduler         *scheduler.Scheduler
}

func NewStatsHandler(usageStatsService *service.UsageStatsService, scheduler *scheduler.Scheduler) *StatsHandler {
	return &StatsHandler{
		usageStatsService: usageStatsService,
		scheduler:         scheduler,
	}
}

func (h *StatsHandler) DailyUsage(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current user not found"})
		return
	}

	days := 7
	if rawDays := c.DefaultQuery("days", "7"); rawDays != "" {
		if parsed, err := strconv.Atoi(rawDays); err == nil && parsed > 0 {
			days = parsed
		}
	}

	stats, err := h.usageStatsService.DailyUsage(c.Request.Context(), &user.ID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (h *StatsHandler) UserUsage(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current user not found"})
		return
	}

	stats, err := h.usageStatsService.UserUsage(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (h *StatsHandler) ProviderUsage(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current user not found"})
		return
	}

	stats, err := h.usageStatsService.ProviderUsage(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (h *StatsHandler) ProviderStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": h.scheduler.Snapshot()})
}
