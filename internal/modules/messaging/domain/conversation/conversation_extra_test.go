package conversation

import (
	"testing"
	"time"
)

// TestNewGroupSuccess 校验群聊构造成功路径与初始状态。
func TestNewGroupSuccess(t *testing.T) {
	t.Parallel()
	g, err := NewGroup(7, 3, "项目组", 100)
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if g.ID() != 7 || g.WorkspaceID() != 3 || g.Name() != "项目组" || g.CreatorID() != 100 {
		t.Fatalf("群聊基础字段错误: %+v", g)
	}
	if g.Type() != TypeGroup || g.Visibility() != VisibilityPrivate {
		t.Fatalf("群聊类型/可见性错误: type=%d vis=%d", g.Type(), g.Visibility())
	}
	if g.Status() != StatusActive || g.MemberCount() != 1 {
		t.Fatalf("群聊初始状态错误: status=%d members=%d", g.Status(), g.MemberCount())
	}
	if g.Metadata() == nil || g.DMKey() != nil {
		t.Fatalf("群聊 metadata/dmKey 错误: %+v", g)
	}
}

// TestNewChannelVisibility 表驱动校验频道可见性分支。
func TestNewChannelVisibility(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		vis     Visibility
		wantErr bool
	}{
		{"public", VisibilityPublic, false},
		{"private", VisibilityPrivate, false},
		{"illegal-zero", Visibility(0), true},
		{"illegal-high", Visibility(9), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ch, err := NewChannel(1, 1, "general", tc.vis, 100)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("可见性 %d 应报错", tc.vis)
				}
				return
			}
			if err != nil {
				t.Fatalf("可见性 %d 意外错误: %v", tc.vis, err)
			}
			if ch.Visibility() != tc.vis || ch.Type() != TypeChannel {
				t.Fatalf("频道状态错误: %+v", ch)
			}
		})
	}
}

// TestNewChannelEmptyName 校验空频道名报错。
func TestNewChannelEmptyName(t *testing.T) {
	t.Parallel()
	if _, err := NewChannel(1, 1, "   ", VisibilityPublic, 100); err == nil {
		t.Fatal("空频道名应报错")
	}
}

// TestArchiveDissolved 校验已解散会话无法归档。
func TestArchiveDissolved(t *testing.T) {
	t.Parallel()
	c := UnmarshalFromDB(1, 1, TypeGroup, VisibilityPrivate, "g", "", 100, nil,
		0, 0, "", nil, 3, StatusDissolved, nil, time.Now(), time.Now())
	if err := c.Archive(); err == nil {
		t.Fatal("已解散会话归档应报错")
	}
	if c.Status() != StatusDissolved {
		t.Fatalf("状态不应改变, got %d", c.Status())
	}
}

// TestIncrMemberCount 表驱动校验成员计数增减与下限钳制。
func TestIncrMemberCount(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		initial int
		delta   int
		want    int
	}{
		{"increase", 1, 2, 3},
		{"decrease", 5, -2, 3},
		{"clamp-to-zero", 1, -5, 0},
		{"exact-zero", 2, -2, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := UnmarshalFromDB(1, 1, TypeGroup, VisibilityPrivate, "g", "", 100, nil,
				0, 0, "", nil, tc.initial, StatusActive, nil, time.Now(), time.Now())
			c.IncrMemberCount(tc.delta)
			if c.MemberCount() != tc.want {
				t.Fatalf("成员数错误: want %d got %d", tc.want, c.MemberCount())
			}
		})
	}
}

// TestUnmarshalFromDB 校验从 DB 重建会话聚合，含 nil metadata 兜底与全字段透传。
func TestUnmarshalFromDB(t *testing.T) {
	t.Parallel()
	now := time.Now()
	at := now.Add(time.Hour)
	key := "1:100#200"
	meta := map[string]any{"k": "v"}

	c := UnmarshalFromDB(11, 22, TypeDM, VisibilityPrivate, "name", "topic", 100, &key,
		9, 88, "digest", &at, 2, StatusActive, meta, now, now)
	if c.ID() != 11 || c.WorkspaceID() != 22 || c.Type() != TypeDM || c.Visibility() != VisibilityPrivate {
		t.Fatalf("重建基础字段错误: %+v", c)
	}
	if c.Name() != "name" || c.Topic() != "topic" || c.CreatorID() != 100 {
		t.Fatalf("重建名称/话题/创建者错误: %+v", c)
	}
	if c.DMKey() == nil || *c.DMKey() != key {
		t.Fatalf("重建 dmKey 错误: %+v", c.DMKey())
	}
	if c.LastSeq() != 9 || c.LastMsgID() != 88 || c.LastMsgDigest() != "digest" {
		t.Fatalf("重建游标错误: %+v", c)
	}
	if c.LastMsgAt() == nil || !c.LastMsgAt().Equal(at) {
		t.Fatal("重建 lastMsgAt 错误")
	}
	if c.MemberCount() != 2 || c.Status() != StatusActive {
		t.Fatalf("重建成员数/状态错误: %+v", c)
	}
	if c.Metadata()["k"] != "v" {
		t.Fatalf("重建 metadata 错误: %+v", c.Metadata())
	}
	if !c.CreatedAt().Equal(now) || !c.UpdatedAt().Equal(now) {
		t.Fatal("重建时间戳错误")
	}

	// nil metadata 应兜底为空 map。
	c2 := UnmarshalFromDB(1, 1, TypeGroup, VisibilityPrivate, "g", "", 100, nil,
		0, 0, "", nil, 1, StatusActive, nil, now, now)
	if c2.Metadata() == nil {
		t.Fatal("nil metadata 应兜底为空 map")
	}
}
