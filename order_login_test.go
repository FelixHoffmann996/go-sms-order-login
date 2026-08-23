package main

import (
	"context"
	"errors"
	"testing"
)

type fakeCodes struct {
	verifyErr error
	verified  bool
}

func (f *fakeCodes) RequestCode(context.Context, string, string) error { return nil }
func (f *fakeCodes) VerifyCode(context.Context, string, string, string) error {
	f.verified = true
	return f.verifyErr
}

func TestVerifyAndReadOrder(t *testing.T) {
	tests := []struct {
		name       string
		orderID    string
		phone      string
		verifyErr  error
		wantOrder  bool
		wantVerify bool
	}{
		{name: "verified customer sees order", orderID: "ord_7", phone: "+15550000007", wantOrder: true, wantVerify: true},
		{name: "rejected code hides order", orderID: "ord_7", phone: "+15550000007", verifyErr: errors.New("code rejected"), wantVerify: true},
		{name: "different phone is rejected before verification", orderID: "ord_7", phone: "+15550000008"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes := &fakeCodes{verifyErr: tt.verifyErr}
			service := &orderLogin{codes: codes, orders: map[string]Order{
				"ord_7": {ID: "ord_7", Phone: "+15550000007", Checkout: "paid", Fulfillment: "shipped", Receipt: "rcpt_7", Updates: []string{"shipped"}},
			}}

			order, err := service.verifyAndRead(context.Background(), tt.orderID, tt.phone, "123456", "req_test_7")
			if (err == nil) != tt.wantOrder {
				t.Fatalf("verifyAndRead() error = %v, wantOrder = %v", err, tt.wantOrder)
			}
			if codes.verified != tt.wantVerify {
				t.Fatalf("VerifyCode called = %v, want %v", codes.verified, tt.wantVerify)
			}
			if tt.wantOrder && (order.Checkout != "paid" || order.Fulfillment != "shipped" || order.Receipt != "rcpt_7") {
				t.Fatalf("unexpected order view: %+v", order)
			}
		})
	}
}
