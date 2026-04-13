package usecase

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"order-service/internal/domain"
	"order-service/internal/repository"
	"order-service/internal/usecase/ports"

	"github.com/google/uuid"
)

var (
	ErrAmountMustBePositive  = errors.New("amount must be > 0")
	ErrPaymentServiceDown    = errors.New("payment service unavailable")
	ErrCannotCancelPaidOrder = errors.New("cannot cancel paid order")
	ErrStatusRequired        = errors.New("status is required")
)

type OrderUsecase struct {
	repo   repository.OrderRepository
	payAPI ports.PaymentClient
}

func NewOrderUsecase(r repository.OrderRepository, payAPI ports.PaymentClient) *OrderUsecase {
	return &OrderUsecase{
		repo:   r,
		payAPI: payAPI,
	}
}

func (u *OrderUsecase) CreateOrder(customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, error) {
	if amount <= 0 {
		return nil, ErrAmountMustBePositive
	}

	if idempotencyKey != "" {
		existing, err := u.repo.GetByIdempotencyKey(idempotencyKey)
		if err == nil {
			return existing, nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	now := time.Now()
	order := &domain.Order{
		ID:             uuid.New().String(),
		CustomerID:     customerID,
		ItemName:       itemName,
		Amount:         amount,
		Status:         "Pending",
		CreatedAt:      now,
		UpdatedAt:      now,
		IdempotencyKey: idempotencyKey,
	}

	if err := u.repo.Create(order); err != nil {
		if errors.Is(err, repository.ErrDuplicateIdempotencyKey) && idempotencyKey != "" {
			existing, findErr := u.repo.GetByIdempotencyKey(idempotencyKey)
			if findErr == nil {
				return existing, nil
			}
			return nil, findErr
		}
		return nil, err
	}

	paymentResp, err := u.payAPI.Authorize(order.ID, order.Amount)
	if err != nil {
		_ = u.repo.UpdateStatus(order.ID, "Failed")
		order.Status = "Failed"
		return nil, ErrPaymentServiceDown
	}

	switch paymentResp.Status {
	case "Authorized":
		order.Status = "Paid"
	case "Declined":
		order.Status = "Failed"
	default:
		order.Status = "Failed"
	}

	if err := u.repo.UpdateStatus(order.ID, order.Status); err != nil {
		return nil, err
	}

	return u.repo.GetByID(order.ID)
}

func (u *OrderUsecase) GetOrder(id string) (*domain.Order, error) {
	return u.repo.GetByID(id)
}

func (u *OrderUsecase) CancelOrder(id string) error {
	order, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}

	if order.Status == "Paid" {
		return ErrCannotCancelPaidOrder
	}

	_, err = u.UpdateOrderStatus(id, "Cancelled")
	return err
}

func (u *OrderUsecase) UpdateOrderStatus(id, status string) (*domain.Order, error) {
	if strings.TrimSpace(status) == "" {
		return nil, ErrStatusRequired
	}

	if err := u.repo.UpdateStatus(id, status); err != nil {
		return nil, err
	}

	return u.repo.GetByID(id)
}

func (u *OrderUsecase) WatchOrderStatus(ctx context.Context, id string, interval time.Duration) (<-chan *domain.Order, <-chan error, error) {
	current, err := u.repo.GetByID(id)
	if err != nil {
		return nil, nil, err
	}

	updates := make(chan *domain.Order, 1)
	errs := make(chan error, 1)

	go func() {
		defer close(updates)
		defer close(errs)

		lastStatus := current.Status
		lastUpdatedAt := current.UpdatedAt

		if !sendOrderUpdate(ctx, updates, current) {
			return
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				latest, watchErr := u.repo.GetByID(id)
				if watchErr != nil {
					select {
					case errs <- watchErr:
					case <-ctx.Done():
					}
					return
				}

				if latest.Status == lastStatus && latest.UpdatedAt.Equal(lastUpdatedAt) {
					continue
				}

				lastStatus = latest.Status
				lastUpdatedAt = latest.UpdatedAt

				if !sendOrderUpdate(ctx, updates, latest) {
					return
				}
			}
		}
	}()

	return updates, errs, nil
}

func sendOrderUpdate(ctx context.Context, updates chan<- *domain.Order, order *domain.Order) bool {
	select {
	case updates <- order:
		return true
	case <-ctx.Done():
		return false
	}
}
