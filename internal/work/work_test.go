package work

import "testing"

func TestNewOrderPreservesRequest(t *testing.T) {
	o:=NewOrder("  add a feature  ")
	if o.SourceRequest!="  add a feature  " || o.Objective!="add a feature" { t.Fatalf("unexpected order: %+v",o) }
	p:=Packet(o)
	if p.WorkOrderID!=o.ID || len(p.CompletionCriteria)!=2 { t.Fatalf("unexpected packet: %+v",p) }
}
