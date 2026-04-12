package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
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
		return nil, errors.New("用户不存在或已禁用")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("密码错误")
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
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
		Role:     role,
		Enable:   true,
	}

	return model.GetDB().Create(user).Error
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
		log.Printf("  密码:   %s", password)
		log.Println("  请登录后立即修改密码！")
		log.Println("========================================")
	}

	return s.Create(username, password, "", "admin")
}

// PurchasePackage 用户购买套餐
func (s *UserService) PurchasePackage(userID, packageID uint) error {
	db := model.GetDB()

	return db.Transaction(func(tx *gorm.DB) error {
		// 获取套餐信息
		var pkg model.Package
		if err := tx.First(&pkg, packageID).Error; err != nil {
			return errors.New("套餐不存在")
		}

		// 获取用户信息（加锁防止并发扣款）
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return errors.New("用户不存在")
		}

		// 检查余额
		if user.Balance < pkg.Price {
			return errors.New("余额不足")
		}

		// 扣除余额
		user.Balance -= pkg.Price

		// 设置流量限制
		if pkg.Traffic > 0 {
			user.TrafficLimit = pkg.Traffic
		}

		// 设置到期时间
		if pkg.Duration > 0 {
			expireAt := time.Now().AddDate(0, 0, pkg.Duration)
			user.ExpireAt = &expireAt
		}

		// 保存用户
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		// 分配节点 (简化处理：分配所有启用节点)
		var nodes []model.Node
		tx.Where("enable = ?", true).Find(&nodes)

		for _, node := range nodes {
			userNode := model.UserNode{
				UserID: userID,
				NodeID: node.ID,
				Enable: true,
			}
			if pkg.Duration > 0 {
				expireAt := time.Now().AddDate(0, 0, pkg.Duration)
				userNode.ExpireAt = &expireAt
			}
			if err := tx.Create(&userNode).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
