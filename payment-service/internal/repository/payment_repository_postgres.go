package repository

import (
	"database/sql"
	"payment-service/internal/domain"
)

type paymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(p *domain.Payment) error {
	_, err := r.db.Exec(
		"INSERT INTO payments (id, order_id, transaction_id, amount, status, processed_at) VALUES ($1,$2,$3,$4,$5,$6)",
		p.ID, p.OrderID, p.TransactionID, p.Amount, p.Status, p.ProcessedAt,
	)
	return err
}

func (r *paymentRepository) GetByOrderID(orderID string) (*domain.Payment, error) {
	row := r.db.QueryRow(
		"SELECT id, order_id, transaction_id, amount, status, processed_at FROM payments WHERE order_id=$1",
		orderID,
	)

	var p domain.Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status, &p.ProcessedAt)
	if err != nil {
		return nil, err
	}

	return &p, nil
}
