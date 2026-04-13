package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

// GetPackages 获取套餐列表（公开，仅启用的）
func (h *Handler) GetPackages(c *gin.Context) {
	var packages []model.Package
	model.GetDB().Where("enable = ?", true).Order("sort asc, price asc").Find(&packages)
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": packages})
}

// GetAllPackages 获取所有套餐（管理员，包含禁用的）
func (h *Handler) GetAllPackages(c *gin.Context) {
	var packages []model.Package
	model.GetDB().Order("sort asc, price asc").Find(&packages)
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": packages})
}

// AddPackage 添加套餐
func (h *Handler) AddPackage(c *gin.Context) {
	var pkg model.Package
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}
	if len(pkg.Name) == 0 || len(pkg.Name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "套餐名称长度需为1-100"})
		return
	}
	if len(pkg.Remark) > 255 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "备注长度不能超过255"})
		return
	}
	if pkg.Price < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "价格不能为负数"})
		return
	}
	if err := model.GetDB().Create(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "添加套餐失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": pkg})
}

// UpdatePackage 更新套餐（支持部分更新）
func (h *Handler) UpdatePackage(c *gin.Context) {
	id := c.Param("id")
	pkgID := parseUint(id)

	// 检查套餐是否存在
	var existing model.Package
	if err := model.GetDB().First(&existing, pkgID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "套餐不存在"})
		return
	}

	// 解析原始 JSON，仅更新请求中明确提供的字段
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	var rawJSON map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &rawJSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	fieldMap := map[string]string{
		"name":     "name",
		"protocol": "protocol",
		"traffic":  "traffic",
		"duration": "duration",
		"price":    "price",
		"nodes":    "nodes",
		"remark":   "remark",
		"sort":     "sort",
		"enable":   "enable",
	}

	updates := map[string]interface{}{}
	for jsonKey, dbCol := range fieldMap {
		if raw, ok := rawJSON[jsonKey]; ok {
			var val interface{}
			if err := json.Unmarshal(raw, &val); err != nil {
				continue
			}
			updates[dbCol] = val
		}
	}

	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	if err := model.GetDB().Model(&model.Package{}).Where("id = ?", pkgID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "更新套餐失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeletePackage 删除套餐
func (h *Handler) DeletePackage(c *gin.Context) {
	id := parseUint(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "无效的ID"})
		return
	}
	if err := model.GetDB().Delete(&model.Package{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "删除套餐失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
