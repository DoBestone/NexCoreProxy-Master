package service

import (
	"errors"
	"fmt"

	"nexcoreproxy-master/internal/model"
)

// RelayBindingService 管理 RelayBinding 表 + 调度同步
type RelayBindingService struct {
	syncer *RelaySyncer
}

func NewRelayBindingService(syncer *RelaySyncer) *RelayBindingService {
	if syncer == nil {
		syncer = NewRelaySyncer()
	}
	return &RelayBindingService{syncer: syncer}
}

func (s *RelayBindingService) List() ([]model.RelayBinding, error) {
	var rows []model.RelayBinding
	err := model.GetDB().Order("id desc").Find(&rows).Error
	return rows, err
}

func (s *RelayBindingService) Get(id uint) (*model.RelayBinding, error) {
	var row model.RelayBinding
	if err := model.GetDB().First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Create 新建 binding 后立即跑一次 SyncBinding
func (s *RelayBindingService) Create(b *model.RelayBinding) error {
	if b.RelayNodeID == 0 || b.BackendNodeID == 0 {
		return errors.New("relayNodeId / backendNodeId 必填")
	}
	if b.RelayNodeID == b.BackendNodeID {
		return errors.New("relay 节点和 backend 节点不能相同")
	}
	if b.Mode == "" {
		b.Mode = "transparent"
	}
	if b.PortStrategy == "" {
		b.PortStrategy = "same"
	}
	if err := model.GetDB().Create(b).Error; err != nil {
		return fmt.Errorf("create binding: %w", err)
	}
	return s.syncer.SyncBinding(b)
}

// Update 编辑 binding 后重新同步（端口策略变化会重新分配端口）
func (s *RelayBindingService) Update(b *model.RelayBinding) error {
	if b.ID == 0 {
		return errors.New("id required")
	}
	if err := model.GetDB().Save(b).Error; err != nil {
		return fmt.Errorf("update binding: %w", err)
	}
	return s.syncer.SyncBinding(b)
}

// Delete 删除 binding（同时清理 binding-source relay）
func (s *RelayBindingService) Delete(id uint) error {
	return s.syncer.DropBinding(id)
}

// Resync 手动触发一次同步（管理员排查用）
func (s *RelayBindingService) Resync(id uint) error {
	b, err := s.Get(id)
	if err != nil {
		return err
	}
	return s.syncer.SyncBinding(b)
}
