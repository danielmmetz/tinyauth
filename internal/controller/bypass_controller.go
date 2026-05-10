package controller

import (
	"database/sql"
	"errors"
	"maps"
	"net"
	"slices"
	"strconv"
	"strings"

	"github.com/steveiliop56/tinyauth/internal/config"
	"github.com/steveiliop56/tinyauth/internal/repository"
	"github.com/steveiliop56/tinyauth/internal/service"
	"github.com/steveiliop56/tinyauth/internal/utils"
	"github.com/steveiliop56/tinyauth/internal/utils/tlog"

	"github.com/gin-gonic/gin"
)

type BypassController struct {
	router          *gin.RouterGroup
	ipBypassService *service.IPBypassService
}

type CreateBypassRequest struct {
	CIDR      string `json:"cidr"`
	Domain    string `json:"domain" binding:"required"`
	ExpiresAt int64  `json:"expiresAt" binding:"required"`
	Note      string `json:"note"`
	CreatedBy string `json:"createdBy"`
}

type BypassResponse struct {
	ID        int64  `json:"id"`
	CIDR      string `json:"cidr"`
	Domain    string `json:"domain"`
	ExpiresAt int64  `json:"expiresAt"`
	Note      string `json:"note"`
	CreatedBy string `json:"createdBy"`
	CreatedAt int64  `json:"createdAt"`
}

type ListBypassesResponse struct {
	Status   int              `json:"status"`
	Message  string           `json:"message"`
	Bypasses []BypassResponse `json:"bypasses"`
	IsAdmin  bool             `json:"isAdmin"`
	ClientIP string           `json:"clientIP"`
}

type DomainsResponse struct {
	Status  int      `json:"status"`
	Message string   `json:"message"`
	Domains []string `json:"domains"`
}

func NewBypassController(router *gin.RouterGroup, ipBypassService *service.IPBypassService) *BypassController {
	return &BypassController{
		router:          router,
		ipBypassService: ipBypassService,
	}
}

func (c *BypassController) SetupRoutes() {
	if c == nil {
		return
	}
	group := c.router.Group("/bypasses")
	{
		group.GET("", c.listHandler)
		group.DELETE("/:id", c.deleteHandler)
		group.POST("", c.createHandler)
		group.GET("/domains", c.domainsHandler)
	}
}

func (c *BypassController) isAdmin(userContext config.UserContext) bool {
	if !userContext.OAuth {
		return false
	}
	for group := range strings.SplitSeq(userContext.OAuthGroups, ",") {
		if strings.TrimSpace(group) == "tinyauth-admin" {
			return true
		}
	}
	return false
}

// validateNonAdminCIDR enforces that non-admins can only create bypasses for
// their own client IP (either as a bare IP or as a single-host CIDR).
func (c *BypassController) validateNonAdminCIDR(clientIP, requestedCIDR string) bool {
	normalizedCIDR := strings.ReplaceAll(requestedCIDR, "-", "/")

	parsedClientIP := net.ParseIP(clientIP)
	if parsedClientIP == nil {
		return false
	}

	if !strings.Contains(normalizedCIDR, "/") {
		bareIP := net.ParseIP(normalizedCIDR)
		if bareIP == nil {
			return false
		}
		return parsedClientIP.Equal(bareIP)
	}

	_, cidr, err := net.ParseCIDR(normalizedCIDR)
	if err != nil {
		return false
	}

	if !cidr.Contains(parsedClientIP) {
		return false
	}

	ones, bits := cidr.Mask.Size()
	switch bits {
	case 32:
		return ones == 32
	case 128:
		return ones == 128
	}
	return false
}

func (c *BypassController) getAllowedDomainsForUser(userContext config.UserContext, allDomains []string, isAdmin bool) []string {
	allowedForUser := make(map[string]bool)
	for domain := range strings.SplitSeq(userContext.BypassDomainsAllowed, ",") {
		trimmed := strings.TrimSpace(domain)
		if trimmed != "" {
			allowedForUser[trimmed] = true
		}
	}
	if isAdmin {
		for _, domain := range allDomains {
			allowedForUser[domain] = true
		}
	}
	return slices.Sorted(maps.Keys(allowedForUser))
}

func (c *BypassController) isDomainAllowedForUser(userContext config.UserContext, domain string, isAdmin bool) bool {
	if isAdmin {
		return true
	}
	for allowed := range strings.SplitSeq(userContext.BypassDomainsAllowed, ",") {
		if strings.TrimSpace(allowed) == domain {
			return true
		}
	}
	return false
}

func (c *BypassController) listHandler(ctx *gin.Context) {
	userContext, err := utils.GetContext(ctx)
	if err != nil || !userContext.IsLoggedIn {
		ctx.JSON(401, gin.H{"status": 401, "message": "Unauthorized"})
		return
	}

	isAdmin := c.isAdmin(userContext)

	bypasses, err := c.ipBypassService.List(ctx, userContext.Username, isAdmin)
	if err != nil {
		tlog.App.Error().Err(err).Msg("Failed to list bypasses")
		ctx.JSON(500, gin.H{"status": 500, "message": "Failed to list bypasses"})
		return
	}

	bypassesResponse := make([]BypassResponse, len(bypasses))
	for i, bypass := range bypasses {
		bypassesResponse[i] = BypassResponse{
			ID:        bypass.ID,
			CIDR:      bypass.Cidr,
			Domain:    bypass.Domain,
			ExpiresAt: bypass.ExpiresAt,
			Note:      bypass.Note,
			CreatedBy: bypass.CreatedBy,
			CreatedAt: bypass.CreatedAt,
		}
	}

	ctx.JSON(200, ListBypassesResponse{
		Status:   200,
		Message:  "Success",
		Bypasses: bypassesResponse,
		IsAdmin:  isAdmin,
		ClientIP: ctx.ClientIP(),
	})
}

func (c *BypassController) createHandler(ctx *gin.Context) {
	userContext, err := utils.GetContext(ctx)
	if err != nil || !userContext.IsLoggedIn {
		ctx.JSON(401, gin.H{"status": 401, "message": "Unauthorized"})
		return
	}

	var req CreateBypassRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"status": 400, "message": "Invalid request"})
		return
	}

	isAdmin := c.isAdmin(userContext)
	targetCIDR := req.CIDR
	if targetCIDR == "" {
		targetCIDR = ctx.ClientIP()
	}

	if !isAdmin {
		clientIP := ctx.ClientIP()
		if !c.validateNonAdminCIDR(clientIP, targetCIDR) {
			ctx.JSON(400, gin.H{"status": 400, "message": "Non-admin users can only create bypasses for their own IP address"})
			return
		}
		if !c.isDomainAllowedForUser(userContext, req.Domain, isAdmin) {
			ctx.JSON(400, gin.H{"status": 400, "message": "You are not allowed to create bypasses for this domain"})
			return
		}
	}

	createdBy := userContext.Username
	if isAdmin && req.CreatedBy != "" {
		createdBy = req.CreatedBy
	}

	bypass, err := c.ipBypassService.Create(ctx, repository.CreateIPBypassParams{
		Cidr:      targetCIDR,
		Domain:    req.Domain,
		ExpiresAt: req.ExpiresAt,
		Note:      req.Note,
		CreatedBy: createdBy,
	})
	if err != nil {
		tlog.App.Error().Err(err).Msg("Failed to create bypass")
		ctx.JSON(400, gin.H{"status": 400, "message": "Failed to create bypass"})
		return
	}

	ctx.JSON(200, gin.H{
		"status":  200,
		"message": "Bypass created successfully",
		"bypass": BypassResponse{
			ID:        bypass.ID,
			CIDR:      bypass.Cidr,
			Domain:    bypass.Domain,
			ExpiresAt: bypass.ExpiresAt,
			Note:      bypass.Note,
			CreatedBy: bypass.CreatedBy,
			CreatedAt: bypass.CreatedAt,
		},
	})
}

func (c *BypassController) deleteHandler(ctx *gin.Context) {
	userContext, err := utils.GetContext(ctx)
	if err != nil || !userContext.IsLoggedIn {
		ctx.JSON(401, gin.H{"status": 401, "message": "Unauthorized"})
		return
	}

	id := ctx.Param("id")
	bypassID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		ctx.JSON(400, gin.H{"status": 400, "message": "Invalid bypass ID"})
		return
	}

	isAdmin := c.isAdmin(userContext)
	err = c.ipBypassService.Delete(ctx, bypassID, userContext.Username, isAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(404, gin.H{"status": 404, "message": "Bypass not found"})
			return
		}
		tlog.App.Error().Err(err).Msg("Failed to delete bypass")
		ctx.JSON(400, gin.H{"status": 400, "message": "Failed to delete bypass"})
		return
	}

	ctx.JSON(200, gin.H{
		"status":  200,
		"message": "Bypass deleted successfully",
	})
}

func (c *BypassController) domainsHandler(ctx *gin.Context) {
	userContext, err := utils.GetContext(ctx)
	if err != nil || !userContext.IsLoggedIn {
		ctx.JSON(401, gin.H{"status": 401, "message": "Unauthorized"})
		return
	}

	domains, err := c.ipBypassService.GetDomains(ctx)
	if err != nil {
		tlog.App.Error().Err(err).Msg("Failed to get domains")
		ctx.JSON(500, gin.H{"status": 500, "message": "Failed to get domains"})
		return
	}

	if domains == nil {
		domains = []string{"*"}
	}

	isAdmin := c.isAdmin(userContext)
	allowedDomains := c.getAllowedDomainsForUser(userContext, domains, isAdmin)

	ctx.JSON(200, DomainsResponse{
		Status:  200,
		Message: "Success",
		Domains: allowedDomains,
	})
}
