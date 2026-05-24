package netutil

import "testing"

func TestParseList(t *testing.T) {
	got := ParseList(" 192.168.0.0/24 , 10.0.0.1/32 , ")
	if len(got) != 2 || got[0] != "192.168.0.0/24" || got[1] != "10.0.0.1/32" {
		t.Fatalf("ParseList: %#v", got)
	}
	if ParseList("") != nil {
		t.Fatal("empty list should be nil")
	}
}

func TestInAnyCIDR(t *testing.T) {
	cidrs := []string{"192.168.0.0/24", "10.0.0.5"}
	if !InAnyCIDR("192.168.0.42", cidrs) {
		t.Fatal("expected match in CIDR")
	}
	if !InAnyCIDR("10.0.0.5", cidrs) {
		t.Fatal("expected match single IP")
	}
	if InAnyCIDR("8.8.8.8", cidrs) {
		t.Fatal("expected no match")
	}
	if InAnyCIDR("not-an-ip", cidrs) {
		t.Fatal("invalid IP should not match")
	}
}
