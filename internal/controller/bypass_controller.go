package controller

import (
	"maps"
	"net"
	"slices"
	"strconv"
	"strings"
	"tinyauth/internal/config"
	"tinyauth/internal/model"
	"tinyauth/internal/service"
	"tinyauth/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
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

func NewBypassController(
	router *gin.RouterGroup,
	ipBypassService *service.IPBypassService,
) *BypassController {
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
	groups := strings.SplitSeq(userContext.OAuthGroups, ",")
	for group := range groups {
		if strings.TrimSpace(group) == "tinyauth-admin" {
			return true
		}
	}
	return false
}

// validateNonAdminCIDR checks that the requested CIDR exactly matches the client's IP.
// For non-admins, they can only create bypasses for their own IP address.
func (c *BypassController) validateNonAdminCIDR(clientIP, requestedCIDR string) bool {
	// Normalize the requested CIDR (replace - with / for range notation)
	normalizedCIDR := strings.ReplaceAll(requestedCIDR, "-", "/")

	// Parse the client IP
	parsedClientIP := net.ParseIP(clientIP)
	if parsedClientIP == nil {
		return false
	}

	// Case 1: Bare IP address (no CIDR notation)
	if !strings.Contains(normalizedCIDR, "/") {
		bareIP := net.ParseIP(normalizedCIDR)
		if bareIP == nil {
			return false
		}
		return parsedClientIP.Equal(bareIP)
	}

	// Case 2: CIDR notation - must be /32 for IPv4 or /128 for IPv6 and match the IP
	_, cidr, err := net.ParseCIDR(normalizedCIDR)
	if err != nil {
		return false
	}

	// Verify the IP is in the CIDR range
	if !cidr.Contains(parsedClientIP) {
		return false
	}

	// Verify it's a /32 (IPv4) or /128 (IPv6) - single host
	ones, bits := cidr.Mask.Size()
	switch bits {
	case 32:
		return ones == 32 // IPv4 /32
	case 128:
		return ones == 128 // IPv6 /128
	}

	return false
}

// getAllowedDomainsForUser returns the list of domains a user is allowed to create bypasses for.
// Admins can access all domains. Non-admins are restricted to their BypassDomainsAllowed list.
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

// isDomainAllowedForUser checks if a user is allowed to create a bypass for a specific domain.
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
		log.Error().Err(err).Msg("Failed to list bypasses")
		ctx.JSON(500, gin.H{"status": 500, "message": "Failed to list bypasses"})
		return
	}

	if bypasses == nil {
		bypasses = []model.IPBypass{}
	}

	bypassesResponse := make([]BypassResponse, len(bypasses))
	for i, bypass := range bypasses {
		bypassesResponse[i] = BypassResponse{
			ID:        bypass.ID,
			CIDR:      bypass.CIDR,
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

	// Non-admin users can only create bypasses for their own IP
	if !isAdmin {
		clientIP := ctx.ClientIP()
		if !c.validateNonAdminCIDR(clientIP, targetCIDR) {
			ctx.JSON(400, gin.H{"status": 400, "message": "Non-admin users can only create bypasses for their own IP address"})
			return
		}

		// Non-admin users can only select from their allowed domains
		if !c.isDomainAllowedForUser(userContext, req.Domain, isAdmin) {
			ctx.JSON(400, gin.H{"status": 400, "message": "You are not allowed to create bypasses for this domain"})
			return
		}
	}

	// Determine CreatedBy: use provided value if admin, otherwise use authenticated user
	createdBy := userContext.Username
	if isAdmin && req.CreatedBy != "" {
		createdBy = req.CreatedBy
	}

	bypass := &model.IPBypass{
		CIDR:      targetCIDR,
		Domain:    req.Domain,
		ExpiresAt: req.ExpiresAt,
		Note:      req.Note,
		CreatedBy: createdBy,
	}

	err = c.ipBypassService.Create(ctx, bypass)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create bypass")
		ctx.JSON(400, gin.H{"status": 400, "message": "Failed to create bypass"})
		return
	}

	ctx.JSON(200, gin.H{
		"status":  200,
		"message": "Bypass created successfully",
		"bypass": BypassResponse{
			ID:        bypass.ID,
			CIDR:      bypass.CIDR,
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
		log.Error().Err(err).Msg("Failed to delete bypass")
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
		log.Error().Err(err).Msg("Failed to get domains")
		ctx.JSON(500, gin.H{"status": 500, "message": "Failed to get domains"})
		return
	}

	if domains == nil {
		domains = []string{"*"}
	}

	// Filter domains for non-admin users
	isAdmin := c.isAdmin(userContext)
	allowedDomains := c.getAllowedDomainsForUser(userContext, domains, isAdmin)

	ctx.JSON(200, DomainsResponse{
		Status:  200,
		Message: "Success",
		Domains: allowedDomains,
	})
}
