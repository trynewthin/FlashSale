package eventx

import "testing"

func TestBuildSeckillOrderNoStable(t *testing.T) {
	got1 := BuildSeckillOrderNo(1001, 2002, 3003, " idem-key ")
	got2 := BuildSeckillOrderNo(1001, 2002, 3003, "idem-key")
	if got1 != got2 {
		t.Fatalf("expected stable order no, got %q and %q", got1, got2)
	}
	if len(got1) != 32 {
		t.Fatalf("unexpected order no len=%d", len(got1))
	}
}
