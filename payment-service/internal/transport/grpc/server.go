package grpcadapter

import (
	"context"
	"log"
	"strings"
	"time"

	paymentv1 "github.com/ayazb/ap2-generated-contracts/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"payment-service/internal/usecase"
)

type PaymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	usecase *usecase.PaymentUsecase
}

func NewPaymentServer(paymentUC *usecase.PaymentUsecase) *PaymentServer {
	return &PaymentServer{usecase: paymentUC}
}

func (s *PaymentServer) ProcessPayment(ctx context.Context, req *paymentv1.PaymentRequest) (*paymentv1.PaymentResponse, error) {
	if strings.TrimSpace(req.GetOrderId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	payment, err := s.usecase.ProcessPayment(req.GetOrderId(), req.GetAmount())
	if err != nil {
		switch err {
		case usecase.ErrAmountMustBePositive:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, "failed to process payment")
		}
	}

	return &paymentv1.PaymentResponse{
		OrderId:       payment.OrderID,
		TransactionId: payment.TransactionID,
		Status:        payment.Status,
		ProcessedAt:   timestamppb.New(payment.ProcessedAt),
	}, nil
}

func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)

	log.Printf(
		"grpc request method=%s duration=%s code=%s",
		info.FullMethod,
		time.Since(start),
		status.Code(err),
	)

	return resp, err
}
