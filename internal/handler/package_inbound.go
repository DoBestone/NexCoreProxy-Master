package handler

import (
	"net/http"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

// GetPackageInbounds GET /api/packages/:id/inbounds
//
// 返回该套餐当前关联的 inbound id 列表 + 已激活订单影响的节点数（前端给个直观提示）
func (h *Handler) GetPackageInbounds(c *gin.Context) {
	pkgID := parseUint(c.Param("id"))
	var rows []model.PackageInbound
	if err := model.GetDB().Where("package_id = ?", pkgID).Find(&rows).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.InboundID)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": ids})
}

// SetPackageInbounds PUT /api/packages/:id/inbounds  body: {inboundIds: [1,2,3]}
//
// 全量替换关联（删旧、写新）。变更后 bump 涉及节点的 etag —— agent 立即拉到。
func (h *Handler) SetPackageInbounds(c *gin.Context) {
	pkgID := parseUint(c.Param("id"))
	var req struct {
		InboundIDs []uint `json:"inboundIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	db := model.GetDB()
	// 收集 before/after inbound 集合 → 算出受影响的节点 id 集合
	affected := map[uint]struct{}{}
	collectNodes := func(inboundIDs []uint) {
		if len(inboundIDs) == 0 {
			return
		}
		var nodeIDs []uint
		_ = db.Model(&model.Inbound{}).Where("id IN ?", inboundIDs).
			Distinct("node_id").Pluck("node_id", &nodeIDs).Error
		for _, id := range nodeIDs {
			affected[id] = struct{}{}
		}
		var relayNodeIDs []uint
		_ = db.Model(&model.Relay{}).Where("backend_inbound_id IN ?", inboundIDs).
			Distinct("relay_node_id").Pluck("relay_node_id", &relayNodeIDs).Error
		for _, id := range relayNodeIDs {
			affected[id] = struct{}{}
		}
	}

	var oldIDs []uint
	_ = db.Model(&model.PackageInbound{}).Where("package_id = ?", pkgID).
		Pluck("inbound_id", &oldIDs).Error
	collectNodes(oldIDs)
	collectNodes(req.InboundIDs)

	// 全量替换
	tx := db.Begin()
	if err := tx.Where("package_id = ?", pkgID).Delete(&model.PackageInbound{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}
	for _, id := range req.InboundIDs {
		if err := tx.Create(&model.PackageInbound{PackageID: pkgID, InboundID: id}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	nodeIDs := make([]uint, 0, len(affected))
	for id := range affected {
		nodeIDs = append(nodeIDs, id)
	}
	_ = model.BumpEtags(nodeIDs)

	c.JSON(http.StatusOK, gin.H{"success": true, "affectedNodes": len(nodeIDs)})
}
