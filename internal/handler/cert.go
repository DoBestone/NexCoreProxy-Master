package handler

import (
	"net/http"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

// ListCerts GET /api/certs
func (h *Handler) ListCerts(c *gin.Context) {
	var rows []model.Certificate
	if err := model.GetDB().Order("expires_at").Find(&rows).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	// 不回 cert/key body（敏感）；前端只需要状态 + 到期
	out := make([]gin.H, len(rows))
	for i, r := range rows {
		out[i] = gin.H{
			"id":         r.ID,
			"domain":     r.Domain,
			"status":     r.Status,
			"issuedAt":   r.IssuedAt,
			"expiresAt":  r.ExpiresAt,
			"lastError":  r.LastError,
			"hasCert":    r.CertPEM != "",
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": out})
}

// IssueCert POST /api/certs/issue  body: {domain: "xxx"}
func (h *Handler) IssueCert(c *gin.Context) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Domain == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "domain required"})
		return
	}
	cert, err := h.cert.Issue(req.Domain)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{
		"domain": cert.Domain, "expiresAt": cert.ExpiresAt, "status": cert.Status,
	}})
}

// DeleteCert DELETE /api/certs/:id
func (h *Handler) DeleteCert(c *gin.Context) {
	id := parseUint(c.Param("id"))
	var cert model.Certificate
	if err := model.GetDB().First(&cert, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "证书不存在"})
		return
	}
	if err := model.GetDB().Delete(&cert).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AcmeSettings GET /api/acme/settings
func (h *Handler) GetAcmeSettings(c *gin.Context) {
	var acct model.AcmeAccount
	if err := model.GetDB().First(&acct).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{"email": "", "configured": false}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{
		"email":          acct.Email,
		"provider":       acct.Provider,
		"configured":     acct.CFAPIToken != "",
		"hasRegistered":  acct.URI != "",
	}})
}

// UpdateAcmeSettings PUT /api/acme/settings  body: {email, cloudflareToken}
func (h *Handler) UpdateAcmeSettings(c *gin.Context) {
	var req struct {
		Email           string `json:"email"`
		CloudflareToken string `json:"cloudflareToken"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	if err := h.cert.SetCloudflareToken(req.Email, req.CloudflareToken); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
