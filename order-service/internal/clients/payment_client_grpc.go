package clients

import (
	"context"
	"time"

	paymentv1 "github.com/ayazb/ap2-generated-contracts/payment/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"order-service/internal/usecase/ports"
)

type PaymentClientGRPC struct {
	client  paymentv1.PaymentServiceClient
	timeout time.Duration
}

func NewPaymentClientGRPC(client paymentv1.PaymentServiceClient, timeout time.Duration) *PaymentClientGRPC {
	return &PaymentClientGRPC{
		client:  client,
		timeout: timeout,
	}
}

func (p *PaymentClientGRPC) Authorize(orderID string, amount int64) (*ports.PaymentResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	resp, err := p.client.ProcessPayment(ctx, &paymentv1.PaymentRequest{
		OrderId:     orderID,
		Amount:      amount,
		RequestedAt: timestamppb.Now(),
	})
	if err != nil {
		return nil, err
	}

	return &ports.PaymentResult{
		Status:        resp.GetStatus(),
		TransactionID: resp.GetTransactionId(),
	}, nil
}
