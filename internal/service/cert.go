// Package service · cert.go — ACME 证书子系统
//
// 设计要点：
//   - Master 是唯一证书签发者；agent 不接触 ACME，只接收 cert+key
//   - DNS-01 走 Cloudflare API（避免节点开 80 端口）
//   - 账户私钥 + 已签发证书全在 Master DB；丢了 = 重签
//   - 续期：每天扫一次，到期 < 30 天的自动续，失败重试 + 告警
//
// 依赖：github.com/go-acme/lego/v4
package service

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"nexcoreproxy-master/internal/model"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/registration"
)

// CertService 持有 ACME 客户端缓存
type CertService struct{}

func NewCertService() *CertService { return &CertService{} }

// Issue 给指定域名签发证书；幂等（已有未过期证书直接返回）
//
// 同步执行（Cloudflare DNS-01 通常 30-60s）；调用方建议 goroutine。
func (s *CertService) Issue(domain string) (*model.Certificate, error) {
	if domain == "" {
		return nil, errors.New("domain required")
	}
	db := model.GetDB()

	// 已有未过期证书直接返回
	var existing model.Certificate
	if err := db.Where("domain = ? AND status = ?", domain, "issued").First(&existing).Error; err == nil {
		if existing.ExpiresAt.After(time.Now().Add(30 * 24 * time.Hour)) {
			return &existing, nil
		}
	}

	acct, err := s.ensureAccount()
	if err != nil {
		return nil, err
	}
	if acct.CFAPIToken == "" {
		return nil, errors.New("Cloudflare API Token 未配置（系统设置 → ACME）")
	}

	user, err := buildLegoUser(acct)
	if err != nil {
		return nil, fmt.Errorf("build lego user: %w", err)
	}

	cfg := lego.NewConfig(user)
	cfg.CADirURL = lego.LEDirectoryProduction
	cfg.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("new lego client: %w", err)
	}

	// Cloudflare DNS-01
	cfProviderConfig := cloudflare.NewDefaultConfig()
	cfProviderConfig.AuthToken = acct.CFAPIToken
	cfProvider, err := cloudflare.NewDNSProviderConfig(cfProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("cloudflare provider: %w", err)
	}
	if err := client.Challenge.SetDNS01Provider(cfProvider); err != nil {
		return nil, fmt.Errorf("set dns01 provider: %w", err)
	}

	// 首次注册（如果账户还没注册过）
	if user.Registration == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return nil, fmt.Errorf("register: %w", err)
		}
		user.Registration = reg
		acct.URI = reg.URI
		_ = db.Save(acct).Error
	}

	req := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}
	res, err := client.Certificate.Obtain(req)
	if err != nil {
		// 记录失败状态
		now := time.Now()
		_ = db.Save(&model.Certificate{
			Domain: domain, Status: "failed",
			LastError: err.Error(), LastTriedAt: &now,
		}).Error
		return nil, fmt.Errorf("obtain: %w", err)
	}

	// 解析过期时间
	expiresAt := parseCertExpiry(res.Certificate)

	out := &model.Certificate{
		Domain:    domain,
		CertPEM:   string(res.Certificate),
		KeyPEM:    string(res.PrivateKey),
		IssuedAt:  time.Now(),
		ExpiresAt: expiresAt,
		Status:    "issued",
	}
	if existing.ID > 0 {
		out.ID = existing.ID
	}
	if err := db.Save(out).Error; err != nil {
		return nil, fmt.Errorf("save cert: %w", err)
	}

	// 证书更新会影响所有引用该域名的 inbound → bump 受影响节点 etag
	bumpNodesForCertDomain(domain)
	log.Printf("[acme] issued cert for %s, expires %s", domain, expiresAt.Format(time.RFC3339))
	return out, nil
}

// Get 拿单条证书（agent 渲染时用）
func (s *CertService) Get(domain string) (*model.Certificate, error) {
	var c model.Certificate
	if err := model.GetDB().Where("domain = ? AND status = ?", domain, "issued").
		First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// RenewExpiring 扫描所有 < 30 天到期的证书并续签
func (s *CertService) RenewExpiring() {
	cutoff := time.Now().Add(30 * 24 * time.Hour)
	var rows []model.Certificate
	if err := model.GetDB().Where("status = ? AND expires_at < ?", "issued", cutoff).
		Find(&rows).Error; err != nil {
		log.Printf("[acme] renew query failed: %v", err)
		return
	}
	for _, c := range rows {
		if _, err := s.Issue(c.Domain); err != nil {
			log.Printf("[acme] renew %s failed: %v", c.Domain, err)
		}
	}
}

// ensureAccount 加载或创建 ACME 账户
//
// 系统只维护一个全局账户（ID=1），简化模型；私钥首次创建后写库。
func (s *CertService) ensureAccount() (*model.AcmeAccount, error) {
	var acct model.AcmeAccount
	if err := model.GetDB().First(&acct).Error; err == nil {
		return &acct, nil
	}
	// 从环境变量取初始邮箱（首次部署时 main.go 注入）
	email := os.Getenv("ACME_EMAIL")
	if email == "" {
		email = "admin@example.com"
	}
	priv, err := generateECKey()
	if err != nil {
		return nil, err
	}
	acct = model.AcmeAccount{
		Email:      email,
		Provider:   "letsencrypt",
		PrivateKey: priv,
	}
	if err := model.GetDB().Create(&acct).Error; err != nil {
		return nil, err
	}
	return &acct, nil
}

// SetCloudflareToken 系统设置接口：管理员从 UI 配置 CF Token
func (s *CertService) SetCloudflareToken(email, token string) error {
	acct, err := s.ensureAccount()
	if err != nil {
		return err
	}
	if email != "" {
		acct.Email = email
	}
	acct.CFAPIToken = token
	return model.GetDB().Save(acct).Error
}

// --- 内部 ---

type legoUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *legoUser) GetEmail() string                        { return u.Email }
func (u *legoUser) GetRegistration() *registration.Resource { return u.Registration }
func (u *legoUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

func buildLegoUser(acct *model.AcmeAccount) (*legoUser, error) {
	key, err := parseECKey(acct.PrivateKey)
	if err != nil {
		return nil, err
	}
	u := &legoUser{Email: acct.Email, key: key}
	if acct.URI != "" {
		u.Registration = &registration.Resource{URI: acct.URI}
	}
	return u, nil
}

func generateECKey() (string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", err
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", err
	}
	out := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	return string(out), nil
}

func parseECKey(pemStr string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("invalid pem")
	}
	return x509.ParseECPrivateKey(block.Bytes)
}

func parseCertExpiry(pemBytes []byte) time.Time {
	for {
		block, rest := pem.Decode(pemBytes)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err == nil {
				return cert.NotAfter
			}
		}
		pemBytes = rest
	}
	return time.Now().Add(80 * 24 * time.Hour) // 兜底
}

// bumpNodesForCertDomain 找出所有引用该域名的 Inbound → bump 对应 Node etag
func bumpNodesForCertDomain(domain string) {
	var nodeIDs []uint
	_ = model.GetDB().Model(&model.Inbound{}).
		Where("cert_domain = ?", domain).
		Distinct("node_id").Pluck("node_id", &nodeIDs).Error
	_ = model.BumpEtags(nodeIDs)
}
