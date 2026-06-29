package conversation

import "time"

// MemberType 表示成员主体类型。
type MemberType int8

const (
	// MemberUser 普通用户。
	MemberUser MemberType = 1
	// MemberBot 机器人（AI 一等公民）。
	MemberBot MemberType = 2
)

// Role 表示会话成员角色。
type Role int8

const (
	// RoleOwner 群主/频道所有者。
	RoleOwner Role = 1
	// RoleAdmin 管理员。
	RoleAdmin Role = 2
	// RoleMember 普通成员。
	RoleMember Role = 3
)

// MemberStatus 表示成员状态。
type MemberStatus int8

const (
	// MemberActive 正常。
	MemberActive MemberStatus = 1
	// MemberLeft 已退出。
	MemberLeft MemberStatus = 2
)

// Member 是会话成员实体（聚合内）。
type Member struct {
	id             int64
	conversationID int64
	memberType     MemberType
	memberID       int64
	role           Role
	readSeq        int64
	lastReadAt     *time.Time
	isMuted        bool
	isPinned       bool
	status         MemberStatus
	joinedAt       time.Time
	createdAt      time.Time
	updatedAt      time.Time
}

func (m *Member) ID() int64              { return m.id }
func (m *Member) ConversationID() int64  { return m.conversationID }
func (m *Member) MemberType() MemberType { return m.memberType }
func (m *Member) MemberID() int64        { return m.memberID }
func (m *Member) Role() Role             { return m.role }
func (m *Member) ReadSeq() int64         { return m.readSeq }
func (m *Member) LastReadAt() *time.Time { return m.lastReadAt }
func (m *Member) IsMuted() bool          { return m.isMuted }
func (m *Member) IsPinned() bool         { return m.isPinned }
func (m *Member) Status() MemberStatus   { return m.status }
func (m *Member) JoinedAt() time.Time    { return m.joinedAt }
func (m *Member) CreatedAt() time.Time   { return m.createdAt }
func (m *Member) UpdatedAt() time.Time   { return m.updatedAt }

// NewMember 创建会话成员。
func NewMember(id, conversationID int64, memberType MemberType, memberID int64, role Role) *Member {
	now := time.Now()
	return &Member{
		id: id, conversationID: conversationID, memberType: memberType, memberID: memberID,
		role: role, readSeq: 0, status: MemberActive, joinedAt: now, createdAt: now, updatedAt: now,
	}
}

// UnmarshalMemberFromDB 从数据库重建成员实体。
func UnmarshalMemberFromDB(id, conversationID int64, memberType MemberType, memberID int64, role Role, readSeq int64, lastReadAt *time.Time, isMuted, isPinned bool, status MemberStatus, joinedAt, createdAt, updatedAt time.Time) *Member {
	return &Member{
		id: id, conversationID: conversationID, memberType: memberType, memberID: memberID,
		role: role, readSeq: readSeq, lastReadAt: lastReadAt, isMuted: isMuted, isPinned: isPinned,
		status: status, joinedAt: joinedAt, createdAt: createdAt, updatedAt: updatedAt,
	}
}

// Unread 根据会话当前 last_seq 计算未读数（不为负）。
func (m *Member) Unread(lastSeq int64) int64 {
	if lastSeq > m.readSeq {
		return lastSeq - m.readSeq
	}
	return 0
}

// MarkRead 单调推进已读水位；新水位不大于旧值时不变更并返回 false。
func (m *Member) MarkRead(seq int64, at time.Time) bool {
	if seq <= m.readSeq {
		return false
	}
	m.readSeq = seq
	m.lastReadAt = &at
	m.updatedAt = at
	return true
}
