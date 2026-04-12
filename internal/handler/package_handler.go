package handler

import (
	"net/http"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
)

// GetPackages 获取套餐列表
func (h *Handler) GetPackages(c *gin.Context) {
	var packages []model.Package
	model.GetDB().Where("enable = ?", true).Order("sort asc, price asc").Find(&packages)
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": packages})
}

// AddPackage 添加套餐
func (h *Handler) AddPackage(c *gin.Context) {
	var pkg model.Package
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}
	if err := model.GetDB().Create(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "添加套餐失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": pkg})
}

// UpdatePackage 更新套餐
func (h *Handler) UpdatePackage(c *gin.Context) {
	id := c.Param("id")
	var pkg model.Package
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}
	pkg.ID = parseUint(id)
	if err := model.GetDB().Save(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "更新套餐失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeletePackage 删除套餐
func (h *Handler) DeletePackage(c *gin.Context) {
	id := c.Param("id")
	if err := model.GetDB().Delete(&model.Package{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "删除套餐失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
