package service

import (
	"errors"
	"fmt"

	"nexcoreproxy-master/internal/model"

	"gorm.io/gorm"
)

// RelaySyncer 把 RelayBinding "整体绑定" 展开为一条条 Relay 记录
//
// 触发场景：
//   - 创建/编辑 RelayBinding → SyncBinding(b)
//   - 删除 RelayBinding → DropBinding(bid)
//   - Backend 节点上的 Inbound 增删改 → OnInboundChanged(backendNodeID)
//     （遍历所有指向该 backend 的 binding，重跑 SyncBinding）
//
// 同步策略：以 binding 为单位增量更新 Relay 表，遵守"管理员手工调整不被覆盖"原则。
type RelaySyncer struct{}

func NewRelaySyncer() *RelaySyncer { return &RelaySyncer{} }

// SyncBinding 把一个 binding 落实到 Relay 表
//
// 算法：
//  1. 查 backend 上所有 enabled inbound
//  2. 查该 binding 已生成的 Relay
//  3. 差集对比：
//     - 有 inbound 但无 relay → 新建（按 PortStrategy 分配端口，wrap 模式自动生成 reality keys）
//     - 有 inbound 也有 relay → 更新（仅当 PortLocked=false 才允许改端口；Source=manual 不动）
//     - 有 relay 但 inbound 没了 → 删除（仅 Source=binding 的；manual 不动避免误杀）
//  4. bump RelayNode etag；wrap 模式额外 bump BackendNode etag（trunk client 要写入）
func (s *RelaySyncer) SyncBinding(binding *model.RelayBinding) error {
	if binding == nil || !binding.Enable || !binding.AutoSync {
		return nil
	}
	db := model.GetDB()

	var inbounds []model.Inbound
	if err := db.Where("node_id = ? AND enable = ?", binding.BackendNodeID, true).
		Order("sort, id").Find(&inbounds).Error; err != nil {
		return fmt.Errorf("load backend inbounds: %w", err)
	}

	var existing []model.Relay
	if err := db.Where("binding_id = ?", binding.ID).Find(&existing).Error; err != nil {
		return fmt.Errorf("load existing relays: %w", err)
	}
	byInbound := make(map[uint]*model.Relay, len(existing))
	for i := range existing {
		byInbound[existing[i].BackendInboundID] = &existing[i]
	}

	usedPorts, err := s.usedPortsOnNode(binding.RelayNodeID)
	if err != nil {
		return err
	}

	// 1+2. 对每条 backend inbound：新建 or 更新
	for _, inb := range inbounds {
		r, ok := byInbound[inb.ID]
		if ok {
			delete(byInbound, inb.ID)
			if r.Source != "binding" {
				continue // manual 不动
			}
			s.refreshRelayFromBinding(r, binding, &inb, usedPorts)
			if err := db.Save(r).Error; err != nil {
				return fmt.Errorf("save relay %d: %w", r.ID, err)
			}
		} else {
			r := s.newRelayFromBinding(binding, &inb, usedPorts)
			if err := db.Create(r).Error; err != nil {
				return fmt.Errorf("create relay for inbound %d: %w", inb.ID, err)
			}
			usedPorts[r.ListenPort] = struct{}{}
		}
	}

	// 3. 残留：backend 已删除的 inbound 对应的 binding-source relay → 删
	for _, leftover := range byInbound {
		if leftover.Source != "binding" {
			continue
		}
		if err := db.Delete(leftover).Error; err != nil {
			return fmt.Errorf("delete stale relay %d: %w", leftover.ID, err)
		}
	}

	// 4. bump etag
	if err := model.BumpEtag(binding.RelayNodeID); err != nil {
		return err
	}
	if binding.Mode == "wrap" {
		_ = model.BumpEtag(binding.BackendNodeID)
	}
	return nil
}

// DropBinding 删除 binding 时调用：清掉所有 binding-source relay 并 bump etag
func (s *RelaySyncer) DropBinding(bindingID uint) error {
	db := model.GetDB()
	var b model.RelayBinding
	if err := db.First(&b, bindingID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if err := db.Where("binding_id = ? AND source = ?", bindingID, "binding").
		Delete(&model.Relay{}).Error; err != nil {
		return err
	}
	if err := db.Delete(&b).Error; err != nil {
		return err
	}
	_ = model.BumpEtag(b.RelayNodeID)
	if b.Mode == "wrap" {
		_ = model.BumpEtag(b.BackendNodeID)
	}
	return nil
}

// OnInboundChanged Backend 节点的 inbound 增删改触发；遍历指向该 backend 的所有 binding 重跑同步
func (s *RelaySyncer) OnInboundChanged(backendNodeID uint) error {
	var bindings []model.RelayBinding
	if err := model.GetDB().Where("backend_node_id = ? AND enable = ? AND auto_sync = ?",
		backendNodeID, true, true).Find(&bindings).Error; err != nil {
		return err
	}
	for i := range bindings {
		if err := s.SyncBinding(&bindings[i]); err != nil {
			return err
		}
	}
	return nil
}

// --- 内部 ---

// newRelayFromBinding 用 binding 默认值创建一条新 Relay
func (s *RelaySyncer) newRelayFromBinding(b *model.RelayBinding, inb *model.Inbound,
	usedPorts map[int]struct{}) *model.Relay {
	r := &model.Relay{
		Name:             fmt.Sprintf("%s→inb%d", relayBindingShortName(b), inb.ID),
		RelayNodeID:      b.RelayNodeID,
		BackendInboundID: inb.ID,
		Mode:             b.Mode,
		ViaRelayID:       b.ViaRelayID,
		Source:           "binding",
		BindingID:        b.ID,
		Enable:           true,
	}
	r.ListenPort = pickPort(b, inb, usedPorts)
	if b.Mode == "wrap" {
		// wrap 字段 + trunk 凭据自动生成
		r.WrapProtocol = b.WrapProtocol
		r.WrapSecurity = b.WrapSecurity
		r.WrapStreamJSON = b.WrapStreamTpl
		if b.AutoGenReality {
			priv, pub, short, _ := genRealityKeys()
			r.WrapRealityPriv = priv
			r.WrapRealityPub = pub
			r.WrapRealityShort = short
		}
		r.TrunkUUID = randomUUID()
		r.TrunkPassword = randomToken(32)
	}
	return r
}

// refreshRelayFromBinding 已存在的 binding-source relay 在 sync 时被刷新
func (s *RelaySyncer) refreshRelayFromBinding(r *model.Relay, b *model.RelayBinding,
	inb *model.Inbound, usedPorts map[int]struct{}) {
	if !r.PortLocked {
		// 重新算端口；如果新算出的端口和现有相同就什么都不变
		newPort := pickPort(b, inb, usedPorts)
		r.ListenPort = newPort
	}
	r.Mode = b.Mode
	r.ViaRelayID = b.ViaRelayID
	if b.Mode == "wrap" {
		r.WrapProtocol = b.WrapProtocol
		r.WrapSecurity = b.WrapSecurity
		// 已有 reality 密钥保留，避免每次 sync 把客户端配置改坏
		if r.WrapRealityPriv == "" && b.AutoGenReality {
			priv, pub, short, _ := genRealityKeys()
			r.WrapRealityPriv = priv
			r.WrapRealityPub = pub
			r.WrapRealityShort = short
		}
		if r.TrunkUUID == "" {
			r.TrunkUUID = randomUUID()
		}
		if r.TrunkPassword == "" {
			r.TrunkPassword = randomToken(32)
		}
	}
}

// usedPortsOnNode 给端口分配器用：当前该 relay 节点上所有已占端口
func (s *RelaySyncer) usedPortsOnNode(nodeID uint) (map[int]struct{}, error) {
	out := make(map[int]struct{})
	var ports []int
	if err := model.GetDB().Model(&model.Relay{}).
		Where("relay_node_id = ?", nodeID).Pluck("listen_port", &ports).Error; err != nil {
		return nil, err
	}
	for _, p := range ports {
		if p > 0 {
			out[p] = struct{}{}
		}
	}
	// 同节点上 inbound 也算占用（自身是 backend 兼任 relay 的场景）
	var inbPorts []int
	_ = model.GetDB().Model(&model.Inbound{}).
		Where("node_id = ?", nodeID).Pluck("port", &inbPorts).Error
	for _, p := range inbPorts {
		if p > 0 {
			out[p] = struct{}{}
		}
	}
	return out, nil
}

// pickPort 端口分配策略
//
//   - same:   = inb.Port，冲突时回退 pool
//   - offset: = inb.Port + binding.PortOffset
//   - pool:   从 [PortPoolStart, PortPoolEnd] 里取最小未占用
func pickPort(b *model.RelayBinding, inb *model.Inbound, used map[int]struct{}) int {
	switch b.PortStrategy {
	case "offset":
		p := inb.Port + b.PortOffset
		if _, occupied := used[p]; !occupied {
			return p
		}
		return pickFromPool(b, used)
	case "pool":
		return pickFromPool(b, used)
	default: // "same"
		if _, occupied := used[inb.Port]; !occupied {
			return inb.Port
		}
		return pickFromPool(b, used)
	}
}

func pickFromPool(b *model.RelayBinding, used map[int]struct{}) int {
	start, end := b.PortPoolStart, b.PortPoolEnd
	if start <= 0 {
		start = 30000
	}
	if end <= 0 {
		end = 65000
	}
	for p := start; p <= end; p++ {
		if _, occupied := used[p]; !occupied {
			used[p] = struct{}{}
			return p
		}
	}
	return 0
}

func relayBindingShortName(b *model.RelayBinding) string {
	return fmt.Sprintf("rb%d", b.ID)
}
