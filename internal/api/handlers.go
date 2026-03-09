package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	auth "github.com/neyuki778/LLM-PDF-OCR/internal/auth"
	"github.com/neyuki778/LLM-PDF-OCR/internal/task"
)

// createTask 处理 POST /api/tasks - 上传 PDF 并创建任务
func (s *Server) createTask(c *gin.Context) {
	// 1. 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "file is required",
		})
		return
	}

	// 2. 验证文件类型（简单检查扩展名）
	if filepath.Ext(file.Filename) != ".pdf" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "only PDF files are allowed now",
		})
		return
	}

	tier, userID, maxPages, statusCode, tierErr := s.resolveTaskTier(c)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": tierErr})
		return
	}

	// 3. 创建上传目录
	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create upload directory",
		})
		return
	}

	// 4. 用 UUID 作为文件名保存，避免冲突
	fileID := uuid.New().String()
	savePath := filepath.Join(uploadDir, fileID+".pdf")
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save file",
		})
		return
	}

	effectiveMaxPages := maxPages
	if s.taskQuota.HardMaxPages > 0 && effectiveMaxPages > s.taskQuota.HardMaxPages {
		effectiveMaxPages = s.taskQuota.HardMaxPages
	}

	// 5. 调用 TaskManager 创建任务
	taskID, err := s.taskManager.CreateTaskWithOptions(savePath, task.CreateTaskOptions{
		MaxPages:    effectiveMaxPages,
		OwnerUserID: userID,
	})
	if err != nil {
		s.cleanupUploadedFile(savePath, "create_task_failed")
		var pageLimitErr *task.PageLimitExceededError
		if errors.As(err, &pageLimitErr) {
			log.Printf(
				"[quota] reject tier=%s user_id=%s total_pages=%d max_pages=%d ip=%s",
				tier,
				userID,
				pageLimitErr.TotalPages,
				pageLimitErr.MaxPages,
				c.ClientIP(),
			)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":       fmt.Sprintf("PDF has %d pages, exceeds max %d pages for %s tier", pageLimitErr.TotalPages, pageLimitErr.MaxPages, tier),
				"tier":        tier,
				"total_pages": pageLimitErr.TotalPages,
				"max_pages":   pageLimitErr.MaxPages,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to create task: %v", err),
		})
		return
	}

	// 6. 提交任务到 WorkerPool 开始处理
	timeOut := 5 * time.Second
	if err := s.taskManager.SubmitTaskToPool(taskID, timeOut); err != nil {
		s.cleanupUploadedFile(savePath, "submit_task_failed")
		log.Printf("[task] submit failed task_id=%s tier=%s user_id=%s err=%v", taskID, tier, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to submit task: %v", err),
		})
		return
	}

	// 7. 返回任务 ID
	c.JSON(http.StatusCreated, gin.H{
		"task_id": taskID,
		"status":  "processing",
		"message": "task created successfully",
	})
}

// getTask 处理 GET /api/tasks/:id - 查询任务状态
func (s *Server) getTask(c *gin.Context) {
	taskID := c.Param("id")
	parentTask := s.taskManager.GetTask(taskID)

	if parentTask == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if !s.authorizeTaskOwner(c, taskID, parentTask.OwnerUserID, "getTask") {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id":         taskID,
		"completed_count": fmt.Sprintf("%d / %d", parentTask.CompletedCount, parentTask.TotalShards),
		"status":          parentTask.Status,
	})
}

// getResult 处理 GET /api/tasks/:id/result - 下载 Markdown 结果
func (s *Server) getResult(c *gin.Context) {
	taskID := c.Param("id")
	parentTask := s.taskManager.GetTask(taskID)

	if parentTask == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if !s.authorizeTaskOwner(c, taskID, parentTask.OwnerUserID, "getResult") {
		return
	}

	if parentTask.Status != task.StatusCompleted {
		c.JSON(http.StatusAccepted, gin.H{
			"task_id": taskID,
			"status":  parentTask.Status,
			"message": "task not completed yet",
		})
		return
	}

	c.File(parentTask.OutputPath)
}

// 暂不支持, task manager还没有实现对应的方法
// deleteTask 处理 DELETE /api/tasks/:id - 删除任务
func (s *Server) deleteTask(c *gin.Context) {
	// 提示：
	// 1. 获取任务 ID
	// 2. 删除任务相关文件（uploads、output 目录）
	// 3. 从 TaskManager 中移除任务

	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "not implemented yet - this is your task!",
	})
}

// getStatus 处理 GET /api/status - 获取服务内部状态
func (s *Server) getStatus(c *gin.Context) {
	status := s.taskManager.GetStatus()

	c.JSON(http.StatusOK, gin.H{
		"timestamp": time.Now().Unix(),
		"status":    status,
	})
}

func (s *Server) resolveTaskTier(c *gin.Context) (tier string, userID string, maxPages int, statusCode int, errMsg string) {
	tier = "guest"
	userID = ""
	maxPages = s.taskQuota.GuestMaxPages

	// Auth 未启用时，全部按 guest 处理。
	if s.authService == nil {
		return tier, userID, maxPages, 0, ""
	}

	accessToken, err := c.Cookie(accessTokenCookieName)
	if err != nil || strings.TrimSpace(accessToken) == "" {
		refreshToken, refreshErr := c.Cookie(refreshTokenCookieName)
		if refreshErr == nil && strings.TrimSpace(refreshToken) != "" {
			log.Printf("[auth] access token missing but refresh token exists on createTask ip=%s", c.ClientIP())
			return "", "", 0, http.StatusUnauthorized, "access token missing"
		}
		return tier, userID, maxPages, 0, ""
	}

	user, err := s.authService.Me(c.Request.Context(), accessToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidAccessToken) {
			log.Printf("[auth] invalid access token on createTask ip=%s", c.ClientIP())
			return "", "", 0, http.StatusUnauthorized, "invalid access token"
		}
		log.Printf("[auth] verify login status failed on createTask ip=%s err=%v", c.ClientIP(), err)
		return "", "", 0, http.StatusInternalServerError, "failed to verify login status"
	}

	return "user", user.ID, s.taskQuota.UserMaxPages, 0, ""
}

func (s *Server) authorizeTaskOwner(c *gin.Context, taskID, ownerUserID, scene string) bool {
	owner := strings.TrimSpace(ownerUserID)
	if owner == "" {
		return true
	}

	requesterUserID, statusCode, errMsg := s.resolveRequesterUserID(c, scene)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": errMsg})
		return false
	}
	if requesterUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return false
	}
	if requesterUserID != owner {
		log.Printf("[authz] task owner mismatch scene=%s task_id=%s owner_user_id=%s requester_user_id=%s ip=%s", scene, taskID, owner, requesterUserID, c.ClientIP())
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return false
	}
	return true
}

func (s *Server) resolveRequesterUserID(c *gin.Context, scene string) (userID string, statusCode int, errMsg string) {
	if s.authService == nil {
		return "", 0, ""
	}

	accessToken, err := c.Cookie(accessTokenCookieName)
	if err != nil || strings.TrimSpace(accessToken) == "" {
		refreshToken, refreshErr := c.Cookie(refreshTokenCookieName)
		if refreshErr == nil && strings.TrimSpace(refreshToken) != "" {
			log.Printf("[auth] access token missing but refresh token exists on %s ip=%s", scene, c.ClientIP())
			return "", http.StatusUnauthorized, "access token missing"
		}
		return "", 0, ""
	}

	user, err := s.authService.Me(c.Request.Context(), accessToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidAccessToken) {
			refreshToken, refreshErr := c.Cookie(refreshTokenCookieName)
			if refreshErr == nil && strings.TrimSpace(refreshToken) != "" {
				log.Printf("[auth] invalid access token with refresh token on %s ip=%s", scene, c.ClientIP())
				return "", http.StatusUnauthorized, "invalid access token"
			}
			log.Printf("[auth] invalid access token on %s ip=%s", scene, c.ClientIP())
			return "", 0, ""
		}
		log.Printf("[auth] verify login status failed on %s ip=%s err=%v", scene, c.ClientIP(), err)
		return "", http.StatusInternalServerError, "failed to verify login status"
	}
	return user.ID, 0, ""
}

func (s *Server) cleanupUploadedFile(path, reason string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("[cleanup] remove uploaded file failed reason=%s path=%s err=%v", reason, path, err)
		return
	}
	log.Printf("[cleanup] removed uploaded file reason=%s path=%s", reason, path)
}
