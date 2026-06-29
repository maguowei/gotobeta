package acl_test

import (
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/acl"
)

func TestDecidePriority(t *testing.T) {
	deny, _ := acl.NewEntry(1, 1, acl.SubjectUser, 10, "conversation", "100", "message.send", acl.EffectDeny, "冻结", 9, nil)
	allow, _ := acl.NewEntry(2, 1, acl.SubjectUser, 10, "conversation", "100", "message.send", acl.EffectAllow, "特批", 9, nil)

	if acl.Decide(true, deny) {
		t.Fatal("显式拒绝应覆盖 RBAC 允许")
	}
	if !acl.Decide(false, allow) {
		t.Fatal("显式允许应放行即使 RBAC 拒绝")
	}
	if acl.Decide(false, nil) {
		t.Fatal("无 ACL 时应回落 RBAC（拒绝）")
	}
	if !acl.Decide(true, nil) {
		t.Fatal("无 ACL 时应回落 RBAC（允许）")
	}
}

func TestNewEntryRequiresReason(t *testing.T) {
	if _, err := acl.NewEntry(1, 1, acl.SubjectUser, 10, "conversation", "100", "message.send", acl.EffectAllow, "", 9, nil); err == nil {
		t.Fatal("空原因应被拒绝")
	}
}

func TestIsActive(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	expired, _ := acl.NewEntry(1, 1, acl.SubjectUser, 10, "c", "1", "a", acl.EffectAllow, "r", 9, &past)
	valid, _ := acl.NewEntry(2, 1, acl.SubjectUser, 10, "c", "1", "a", acl.EffectAllow, "r", 9, &future)
	if expired.IsActive(time.Now()) {
		t.Fatal("已过期条目应失效")
	}
	if !valid.IsActive(time.Now()) {
		t.Fatal("未过期条目应有效")
	}
}
