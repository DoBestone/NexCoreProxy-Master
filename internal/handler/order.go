package handler

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"nexcoreproxy-master/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetMyOrders 获取我的订单
func (h *Handler) GetMyOrders(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	var orders []model.Order
	model.GetDB().Where("user_id = ?", userID).Order("created_at desc").Limit(20).Find(&orders)
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": orders})
}

// GetAllOrders 获取所有订单（管理员）
func (h *Handler) GetAllOrders(c *gin.Context) {
	var orders []model.Order
	if err := model.GetDB().Preload("User").Order("created_at desc").Limit(500).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "获取订单失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": orders})
}

// CreateOrder 创建订单
func (h *Handler) CreateOrder(c *gin.Context) {
	userID := h.getCurrentUserID(c)

	var req struct {
		PackageID uint   `json:"packageId"`
		PayMethod string `json:"payMethod"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	// 获取套餐信息
	var pkg model.Package
	if err := model.GetDB().First(&pkg, req.PackageID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "套餐不存在"})
		return
	}

	// 生成订单号
	randN, _ := rand.Int(rand.Reader, big.NewInt(1000))
	orderNo := fmt.Sprintf("NCP%d%d", time.Now().UnixNano()/1000000, randN.Int64())

	now := time.Now()
	order := &model.Order{
		OrderNo:     orderNo,
		UserID:      userID,
		PackageID:   pkg.ID,
		PackageName: pkg.Name,
		Amount:      pkg.Price,
		PayMethod:   req.PayMethod,
	}

	// 余额支付：先创建订单再扣款，保证原子性
	if req.PayMethod == "balance" {
		order.Status = "paid"
		order.PaidAt = &now

		// 在事务中同时创建订单和扣款
		err := model.GetDB().Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(order).Error; err != nil {
				return err
			}
			return h.user.PurchasePackage(userID, req.PackageID)
		})
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
			return
		}
	} else {
		order.Status = "pending"
		if err := model.GetDB().Create(order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "创建订单失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": order, "msg": "购买成功"})
}

// UpdateOrderStatus 更新订单状态（管理员）
func (h *Handler) UpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status": req.Status,
	}
	if req.Status == "paid" {
		updates["paid_at"] = &now
	}

	if err := model.GetDB().Model(&model.Order{}).Where("id = ?", parseUint(id)).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
