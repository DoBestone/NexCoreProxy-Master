package model

import "time"

// AcmeAccount 全局 ACME 账户（一个站点通常一个 Let's Encrypt 账户即可）
//
// PrivateKey 是 PEM 编码的 EC 私钥；URI 是 ACME 服务端返回的账户 URL。
// CFAPIToken 是 Cloudflare API Token（DNS-01 验证用）。多账户/多 provider 留 Phase 2。
type AcmeAccount struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Email      string    `json:"email" gorm:"size:255;not null"`
	Provider   string    `json:"provider" gorm:"size:30;default:'letsencrypt'"` // letsencrypt / zerossl
	PrivateKey string    `json:"-" gorm:"type:text"`                            // PEM
	URI        string    `json:"uri" gorm:"size:500"`
	CFAPIToken string    `json:"-" gorm:"size:255"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Certificate 已签发的证书；按 Domain 唯一
//
// CertPEM/KeyPEM 是完整的证书链 + 私钥（agent 拿到后直接落盘给 xray 用）。
type Certificate struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	Domain     string     `json:"domain" gorm:"size:255;uniqueIndex;not null"`
	CertPEM    string     `json:"-" gorm:"type:longtext"`
	KeyPEM     string     `json:"-" gorm:"type:longtext"`
	IssuedAt   time.Time  `json:"issuedAt"`
	ExpiresAt  time.Time  `json:"expiresAt" gorm:"index"`
	Status     string     `json:"status" gorm:"size:20;default:'issued'"` // issued / failed / pending
	LastError  string     `json:"lastError" gorm:"size:1000"`
	LastTriedAt *time.Time `json:"lastTriedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}
