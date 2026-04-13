package grpcadapter

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	orderv1 "github.com/ayazb/ap2-generated-contracts/order/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"order-service/internal/usecase"
)

type OrderServer struct {
	orderv1.UnimplementedOrderServiceServer
	usecase      *usecase.OrderUsecase
	pollInterval time.Duration
}

func NewOrderServer(orderUC *usecase.OrderUsecase, pollInterval time.Duration) *OrderServer {
	return &OrderServer{
		usecase:      orderUC,
		pollInterval: pollInterval,
	}
}

func (s *OrderServer) SubscribeToOrderUpdates(
	req *orderv1.OrderRequest,
	stream orderv1.OrderService_SubscribeToOrderUpdatesServer,
) error {
	orderID := strings.TrimSpace(req.GetOrderId())
	if orderID == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	updates, errs, err := s.usecase.WatchOrderStatus(stream.Context(), orderID, s.pollInterval)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return status.Error(codes.NotFound, "order not found")
		}
		return status.Error(codes.Internal, "failed to subscribe to order updates")
	}

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case watchErr, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if errors.Is(watchErr, sql.ErrNoRows) {
				return status.Error(codes.NotFound, "order not found")
			}
			return status.Error(codes.Internal, "failed to read order updates")
		case order, ok := <-updates:
			if !ok {
				return nil
			}
			if err := stream.Send(&orderv1.OrderStatusUpdate{
				OrderId:   order.ID,
				Status:    order.Status,
				UpdatedAt: timestamppb.New(order.UpdatedAt),
			}); err != nil {
				return status.Error(codes.Unavailable, "failed to stream order update")
			}
		}
	}
}
