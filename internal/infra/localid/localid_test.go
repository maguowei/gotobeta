package localid

import (
	"context"
	"sync"
	"testing"
)

func TestGeneratorNextIDIncreases(t *testing.T) {
	generator := New()

	first, err := generator.NextID(context.Background())
	if err != nil {
		t.Fatalf("NextID() first error = %v", err)
	}
	second, err := generator.NextID(context.Background())
	if err != nil {
		t.Fatalf("NextID() second error = %v", err)
	}

	if first <= 0 {
		t.Fatalf("first id = %d, want positive", first)
	}
	if second <= first {
		t.Fatalf("second id = %d, want greater than first %d", second, first)
	}
}

// TestGeneratorNextIDUniqueUnderConcurrency 验证同进程高并发下不产生重复 ID。
func TestGeneratorNextIDUniqueUnderConcurrency(t *testing.T) {
	generator := New()

	const workers = 16
	const perWorker = 1000

	var mu sync.Mutex
	seen := make(map[int64]struct{}, workers*perWorker)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			ids := make([]int64, 0, perWorker)
			for range perWorker {
				id, err := generator.NextID(context.Background())
				if err != nil {
					t.Errorf("NextID() error = %v", err)
					return
				}
				ids = append(ids, id)
			}
			mu.Lock()
			defer mu.Unlock()
			for _, id := range ids {
				if _, dup := seen[id]; dup {
					t.Errorf("duplicate id generated: %d", id)
					return
				}
				seen[id] = struct{}{}
			}
		})
	}
	wg.Wait()
}

// TestGeneratorsDifferentNodesDoNotCollide 验证不同节点号的生成器即使同一时刻取号也不冲突，
// 这是多副本部署不产生重复主键的关键保证。
func TestGeneratorsDifferentNodesDoNotCollide(t *testing.T) {
	a := &Generator{nodeID: 1}
	b := &Generator{nodeID: 2}

	const n = 4000
	ids := make(map[int64]struct{}, n*2)
	for range n {
		idA, err := a.NextID(context.Background())
		if err != nil {
			t.Fatalf("node a NextID() error = %v", err)
		}
		idB, err := b.NextID(context.Background())
		if err != nil {
			t.Fatalf("node b NextID() error = %v", err)
		}
		for _, id := range []int64{idA, idB} {
			if _, dup := ids[id]; dup {
				t.Fatalf("cross-node duplicate id: %d", id)
			}
			ids[id] = struct{}{}
		}
	}
}
