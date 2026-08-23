package main

import (
	"context"
	"errors"
)

type codeGateway interface {
	RequestCode(context.Context, string, string) error
	VerifyCode(context.Context, string, string, string) error
}

type Order struct {
	ID          string   `json:"id"`
	Phone       string   `json:"-"`
	Checkout    string   `json:"checkout"`
	Fulfillment string   `json:"fulfillment"`
	Receipt     string   `json:"receipt"`
	Updates     []string `json:"customer_order_updates"`
}

type orderLogin struct {
	codes  codeGateway
	orders map[string]Order
}

func (s *orderLogin) sendCode(ctx context.Context, orderID, phone, requestID string) error {
	order, ok := s.orders[orderID]
	if !ok || order.Phone != phone {
		return errors.New("order and phone do not match")
	}
	return s.codes.RequestCode(ctx, phone, requestID)
}

func (s *orderLogin) verifyAndRead(ctx context.Context, orderID, phone, code, requestID string) (Order, error) {
	order, ok := s.orders[orderID]
	if !ok || order.Phone != phone {
		return Order{}, errors.New("order and phone do not match")
	}
	if err := s.codes.VerifyCode(ctx, phone, code, requestID); err != nil {
		return Order{}, err
	}
	return order, nil
}
