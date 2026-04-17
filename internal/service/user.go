package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"nexcoreproxy-master/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// JWT密钥：优先从环境变量读取，未设置则自动生成随机密钥（每次重启会导致旧token失效）
var jwtSecret = func() []byte {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		if len(secret) < 32 {
			log.Fatal("JWT_SECRET 长度不能少于 32 字符")
		}
		return []byte(secret)
	}
	// 未配置时自动生成随机密钥
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatal("生成 JWT 密钥失败:", err)
	}
	secret := hex.EncodeToString(b)
	log.Println("[安全警告] JWT_SECRET 未设置，已自动生成临时密钥（重启后所有登录会话将失效）")
	log.Println("[建议] 设置环境变量: export JWT_SECRET=\"" + secret + "\"")
	return []byte(secret)
}()

// Claims JWT声明
type Claims struct {
	UserID   uint   `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// UserService 用户服务
type UserService struct{}

// NewUserService 创建用户服务
func NewUserService() *UserService {
	return &UserService{}
}

// Authenticate 验证用户登录
func (s *UserService) Authenticate(username, password string) (*model.User, error) {
	var user model.User
	if err := model.GetDB().Where("username = ? AND enable = ?", username, true).First(&user).Error; err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	return &user, nil
}

// GenerateToken 生成JWT Token
func (s *UserService) GenerateToken(user *model.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "nexcore-proxy",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析JWT Token
func (s *UserService) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 防止算法混淆攻击：只允许 HMAC 签名
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// Create 创建用户
func (s *UserService) Create(username, password, email, role string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &model.User{
		Username:       username,
		Password:       string(hashedPassword),
		Email:          email,
		Role:           role,
		Enable:         true,
		SubscribeToken: NewSubscribeToken(),
	}

	db := model.GetDB()
	if err := db.Create(user).Error; err != nil {
		return err
	}

	// 生成邀请码（与注册流程一致：NC + 用户ID）
	inviteCode := fmt.Sprintf("NC%06d", user.ID)
	return db.Model(user).Update("invite_code", inviteCode).Error
}

// NewSubscribeToken 生成 32 字符（128 bit）的随机订阅令牌，URL-safe
func NewSubscribeToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// rand.Read 失败极罕见，fallback 到时间戳 hash 避免返回空串
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// BackfillSubscribeTokens 为历史用户补齐 SubscribeToken（幂等，仅填充空值）
func (s *UserService) BackfillSubscribeTokens() error {
	var users []model.User
	if err := model.GetDB().Where("subscribe_token = '' OR subscribe_token IS NULL").Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		if err := model.GetDB().Model(&model.User{}).Where("id = ?", u.ID).
			Update("subscribe_token", NewSubscribeToken()).Error; err != nil {
			log.Printf("[SubscribeToken] 用户 %d 补齐失败: %v", u.ID, err)
		}
	}
	return nil
}

// ResetSubscribeToken 为指定用户生成新的订阅令牌（旧 URL 立即失效）
func (s *UserService) ResetSubscribeToken(userID uint) (string, error) {
	tok := NewSubscribeToken()
	if err := model.GetDB().Model(&model.User{}).Where("id = ?", userID).
		Update("subscribe_token", tok).Error; err != nil {
		return "", err
	}
	return tok, nil
}

// HashPassword 加密密码
func (s *UserService) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// GetAll 获取所有用户
func (s *UserService) GetAll() ([]model.User, error) {
	var users []model.User
	err := model.GetDB().Find(&users).Error
	return users, err
}

// GetByID 根据ID获取用户
func (s *UserService) GetByID(id uint) (*model.User, error) {
	var user model.User
	err := model.GetDB().First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户
func (s *UserService) Update(user *model.User) error {
	return model.GetDB().Save(user).Error
}

// Delete 删除用户
func (s *UserService) Delete(id uint) error {
	return model.GetDB().Delete(&model.User{}, id).Error
}

// InitAdmin 初始化管理员账户
func (s *UserService) InitAdmin() error {
	var count int64
	model.GetDB().Model(&model.User{}).Count(&count)
	if count > 0 {
		return nil
	}

	// 从环境变量读取管理员账号密码
	username := os.Getenv("NCP_ADMIN_USER")
	password := os.Getenv("NCP_ADMIN_PASS")

	if username == "" {
		username = "admin"
	}
	if password == "" {
		// 自动生成安全随机密码
		b := make([]byte, 12)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		password = hex.EncodeToString(b)
		log.Println("========================================")
		log.Println("  管理员账户已自动创建")
		log.Printf("  用户名: %s", username)
		log.Println("  密码已自动生成，请查看启动日志")
		log.Println("  请登录后立即修改密码！")
		log.Println("========================================")
		// 密码仅在首次创建时输出到 stderr，避免日志持久化泄露
		fmt.Fprintf(os.Stderr, "[NexCore] 初始管理员密码: %s\n", password)
	}

	return s.Create(username, password, "", "admin")
}

// PurchasePackage 用户余额购买套餐（自建事务版本）
func (s *UserService) PurchasePackage(userID, packageID uint) error {
	return s.PurchasePackageWithTx(model.GetDB(), userID, packageID)
}

// PurchasePackageWithTx 用户余额购买套餐（扣余额 + 激活）
func (s *UserService) PurchasePackageWithTx(db *gorm.DB, userID, packageID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var pkg model.Package
		if err := tx.First(&pkg, packageID).Error; err != nil {
			return errors.New("套餐不存在")
		}

		// 获取用户信息（加锁防止并发扣款）
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("用户不存在")
			}
			return fmt.Errorf("查询用户失败: %v", err)
		}

		if user.Balance < pkg.Price {
			return errors.New("余额不足")
		}

		user.Balance -= pkg.Price
		return s.activatePackage(tx, &user, &pkg)
	})
}

// ActivatePackageWithTx 激活套餐（不扣余额，用于管理员确认外部支付订单）
func (s *UserService) ActivatePackageWithTx(db *gorm.DB, userID, packageID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var pkg model.Package
		if err := tx.First(&pkg, packageID).Error; err != nil {
			return errors.New("套餐不存在")
		}

		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("用户不存在")
			}
			return fmt.Errorf("查询用户失败: %v", err)
		}

		return s.activatePackage(tx, &user, &pkg)
	})
}

// activatePackage 自研 agent 架构版本
//
//   1. 确保用户有 UUID / TrojanPassword / SS2022Password（首次激活生成）
//   2. 设置流量限额（优先 TransferGB，否则 Traffic）+ 重置 traffic_used + 续期 expire_at + 重新启用
//   3. 套餐的 device/speed limit 同步到用户上
//   4. 不再写 user_nodes（新架构通过 orders.paid + package_inbounds 反查授权）
//   5. bump 该套餐覆盖到的所有节点 etag → agent 立即拉到新用户
func (s *UserService) activatePackage(tx *gorm.DB, user *model.User, pkg *model.Package) error {
	// 1. 凭据
	if user.UUID == "" {
		user.UUID = randomUUID()
	}
	if user.TrojanPassword == "" {
		user.TrojanPassword = randomToken(32)
	}
	if user.SS2022Password == "" {
		// SS-2022 AES-128-GCM 需要 16 字节 base64 编码的 PSK，xray 和客户端都按 base64 解
		user.SS2022Password = randomSS2022PSK(16)
	}
	if user.SubscribeToken == "" {
		user.SubscribeToken = randomSubscribeToken()
	}

	// 2. 流量与有效期
	limit := pkg.Traffic
	if pkg.TransferGB > 0 {
		limit = int64(pkg.TransferGB) * 1024 * 1024 * 1024
	}
	if limit > 0 {
		user.TrafficLimit = limit
	}
	user.TrafficUsed = 0 // 续费/换套餐重置
	if pkg.Duration > 0 {
		expireAt := time.Now().AddDate(0, 0, pkg.Duration)
		user.ExpireAt = &expireAt
	}
	if pkg.SpeedLimit > 0 {
		user.SpeedLimit = pkg.SpeedLimit
	}
	if pkg.DeviceLimit > 0 {
		user.DeviceLimit = pkg.DeviceLimit
	}
	user.Enable = true // 之前因超额停用的恢复

	if err := tx.Save(user).Error; err != nil {
		return err
	}

	// 3. bump 受影响节点 etag（agent 下次拉立即看到该用户）
	var nodeIDs []uint
	_ = tx.Raw(`
		SELECT DISTINCT i.node_id
		FROM package_inbounds pi
		JOIN inbounds i ON i.id = pi.inbound_id
		WHERE pi.package_id = ?
	`, pkg.ID).Scan(&nodeIDs).Error
	// 同时 bump 关联的 relay 节点
	var relayNodeIDs []uint
	_ = tx.Raw(`
		SELECT DISTINCT r.relay_node_id
		FROM relays r
		JOIN package_inbounds pi ON pi.inbound_id = r.backend_inbound_id
		WHERE pi.package_id = ?
	`, pkg.ID).Scan(&relayNodeIDs).Error
	_ = model.BumpEtags(append(nodeIDs, relayNodeIDs...))

	return nil
}
