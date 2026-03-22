package api

import (
	"fmt"
	"net/http"

	"ai-gateway/middleware"
	"ai-gateway/provider"
	"ai-gateway/service"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	chatService *service.ChatService
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

func (h *ChatHandler) ChatCompletions(c *gin.Context) {
	var req provider.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "current user not found",
		})
		return
	}

	apiKey, ok := middleware.CurrentAPIKey(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "current api key not found",
		})
		return
	}

	if req.Stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)
		c.Writer.Flush()

		_, err := h.chatService.StreamChat(c.Request.Context(), user, apiKey, &req, func(event provider.StreamEvent) error {
			if _, writeErr := fmt.Fprintf(c.Writer, "data: %s\n\n", event.Data); writeErr != nil {
				return writeErr
			}
			c.Writer.Flush()
			return nil
		})
		if err != nil {
			_, _ = fmt.Fprintf(c.Writer, "event: error\ndata: {\"error\":%q}\n\n", err.Error())
			c.Writer.Flush()
		}
		return
	}

	resp, err := h.chatService.Chat(c.Request.Context(), user, apiKey, &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
