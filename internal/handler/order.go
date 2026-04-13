package handler

import (
	"crypto/rand"
	"fmt"
	"log"
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
	if err := model.GetDB().Where("user_id = ?", userID).Order("created_at desc").Limit(20).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "获取订单失败"})
		return
	}
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
	randN, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		randN = big.NewInt(0)
	}
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

	// 余额支付：在同一事务中创建订单 + 扣款 + 分配节点
	if req.PayMethod == "balance" {
		order.Status = "paid"
		order.PaidAt = &now

		err := model.GetDB().Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(order).Error; err != nil {
				return err
			}
			return h.user.PurchasePackageWithTx(tx, userID, req.PackageID)
		})
		if err != nil {
			log.Printf("余额购买套餐失败 [user=%d, pkg=%d]: %v", userID, req.PackageID, err)
			c.JSON(http.StatusOK, gin.H{"success": false, "msg": "购买失败，请稍后重试"})
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

	// 获取订单信息
	var order model.Order
	if err := model.GetDB().First(&order, parseUint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": "订单不存在"})
		return
	}

	// 状态转移校验：禁止从终态回退
	allowed := map[string][]string{
		"pending":   {"paid", "cancelled"},
		"paid":      {"refunded"},
		"cancelled": {},
		"refunded":  {},
	}
	validTransitions, ok := allowed[order.Status]
	if !ok || !contains(validTransitions, req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": fmt.Sprintf("不允许从 %s 转为 %s", order.Status, req.Status)})
		return
	}

	// 从 pending → paid 时激活套餐
	if req.Status == "paid" && order.Status == "pending" {
		now := time.Now()
		err := model.GetDB().Transaction(func(tx *gorm.DB) error {
			order.Status = "paid"
			order.PaidAt = &now
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
			return h.user.ActivatePackageWithTx(tx, order.UserID, order.PackageID)
		})
		if err != nil {
			log.Printf("激活套餐失败 [order=%d, user=%d, pkg=%d]: %v", order.ID, order.UserID, order.PackageID, err)
			c.JSON(http.StatusOK, gin.H{"success": false, "msg": "激活套餐失败"})
			return
		}
	} else {
		updates := map[string]interface{}{"status": req.Status}
		if err := model.GetDB().Model(&order).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "更新失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
