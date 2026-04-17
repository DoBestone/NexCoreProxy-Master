package service

import (
	"errors"
	"fmt"

	"nexcoreproxy-master/internal/model"
)

// InboundService 管理 Inbound 表
//
// 任何会改变 inbound 的操作都要 bump 该节点的 etag，并触发依赖该 inbound 的 RelayBinding 同步。
type InboundService struct {
	syncer *RelaySyncer
}

func NewInboundService() *InboundService {
	return &InboundService{syncer: NewRelaySyncer()}
}

// AttachSyncer 用于循环依赖时延迟注入；当前同进程构造能直接用 NewRelaySyncer，预留位
func (s *InboundService) AttachSyncer(syncer *RelaySyncer) { s.syncer = syncer }

// List 列出某节点的所有 inbound（管理用）
func (s *InboundService) List(nodeID uint) ([]model.Inbound, error) {
	var rows []model.Inbound
	q := model.GetDB().Order("sort, id")
	if nodeID > 0 {
		q = q.Where("node_id = ?", nodeID)
	}
	err := q.Find(&rows).Error
	return rows, err
}

// Get 查单条
func (s *InboundService) Get(id uint) (*model.Inbound, error) {
	var row model.Inbound
	if err := model.GetDB().First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Create 创建 inbound：写库 → bump 节点 etag → 触发 RelayBinding 同步（自动给 relay 节点开端口）
func (s *InboundService) Create(in *model.Inbound) error {
	if in.NodeID == 0 {
		return errors.New("node_id required")
	}
	if in.Tag == "" {
		return errors.New("tag required")
	}
	if in.Port <= 0 {
		return errors.New("port required")
	}
	if err := model.GetDB().Create(in).Error; err != nil {
		return fmt.Errorf("create inbound: %w", err)
	}
	if err := model.BumpEtag(in.NodeID); err != nil {
		return err
	}
	if s.syncer != nil {
		_ = s.syncer.OnInboundChanged(in.NodeID)
	}
	return nil
}

// Update 更新 inbound：写库 → bump 节点 etag → 同步 binding
func (s *InboundService) Update(in *model.Inbound) error {
	if in.ID == 0 {
		return errors.New("id required")
	}
	if err := model.GetDB().Save(in).Error; err != nil {
		return fmt.Errorf("update inbound: %w", err)
	}
	if err := model.BumpEtag(in.NodeID); err != nil {
		return err
	}
	if s.syncer != nil {
		_ = s.syncer.OnInboundChanged(in.NodeID)
	}
	return nil
}

// Delete 删除 inbound：先删库 → bump → 同步 binding（会清掉关联 relay）
func (s *InboundService) Delete(id uint) error {
	var in model.Inbound
	if err := model.GetDB().First(&in, id).Error; err != nil {
		return err
	}
	if err := model.GetDB().Delete(&in).Error; err != nil {
		return err
	}
	if err := model.BumpEtag(in.NodeID); err != nil {
		return err
	}
	if s.syncer != nil {
		_ = s.syncer.OnInboundChanged(in.NodeID)
	}
	return nil
}

// Toggle 切换 enable
func (s *InboundService) Toggle(id uint, enable bool) error {
	var in model.Inbound
	if err := model.GetDB().First(&in, id).Error; err != nil {
		return err
	}
	in.Enable = enable
	return s.Update(&in)
}
