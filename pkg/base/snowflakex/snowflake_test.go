package snowflakex

import (
	"os"
	"testing"
)

func TestNewNode_Default(t *testing.T) {
	os.Unsetenv("SNOWFLAKE_NODE")
	node, err := NewNode(0)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	id := node.Generate()
	if id.Int64() <= 0 {
		t.Fatal("generated ID should be positive")
	}
}

func TestNewNode_WithConfig(t *testing.T) {
	os.Unsetenv("SNOWFLAKE_NODE")
	node, err := NewNode(5)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	id := node.Generate()
	if id.Int64() <= 0 {
		t.Fatal("generated ID should be positive")
	}
}

func TestNewNode_WithEnv(t *testing.T) {
	os.Setenv("SNOWFLAKE_NODE", "42")
	defer os.Unsetenv("SNOWFLAKE_NODE")
	node, err := NewNode(0)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	id := node.Generate()
	if id.Int64() <= 0 {
		t.Fatal("generated ID should be positive")
	}
}

func TestNewNode_UniqueIDs(t *testing.T) {
	os.Unsetenv("SNOWFLAKE_NODE")
	node, err := NewNode(1)
	if err != nil {
		t.Fatalf("NewNode failed: %v", err)
	}
	seen := make(map[int64]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := node.Generate().Int64()
		if seen[id] {
			t.Fatalf("duplicate ID: %d", id)
		}
		seen[id] = true
	}
}

func TestResolveNodeID_Bounds(t *testing.T) {
	os.Unsetenv("SNOWFLAKE_NODE")
	// 无论输入多大，结果应在 0~1023 范围内
	tests := []int64{0, 1, 100, 1023, 1024, 99999}
	for _, v := range tests {
		nid := resolveNodeID(v)
		if nid < 0 || nid > maxNodeBits {
			t.Fatalf("resolveNodeID(%d) = %d, out of range [0, %d]", v, nid, maxNodeBits)
		}
	}
}

func TestFnvHash_Deterministic(t *testing.T) {
	h1 := fnvHash("test-hostname")
	h2 := fnvHash("test-hostname")
	if h1 != h2 {
		t.Fatal("fnvHash should be deterministic")
	}
}

func TestFnvHash_Different(t *testing.T) {
	h1 := fnvHash("host-a")
	h2 := fnvHash("host-b")
	if h1 == h2 {
		t.Fatal("fnvHash should differ for different inputs")
	}
}
