package repository

import (
	"database/sql"
	"errors"

	"order-service/internal/domain"

	"github.com/lib/pq"
)

var ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")

type orderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(o *domain.Order) error {
	var idem interface{}
	if o.IdempotencyKey != "" {
		idem = o.IdempotencyKey
	} else {
		idem = nil
	}

	_, err := r.db.Exec(
		`INSERT INTO orders (id, customer_id, item_name, amount, status, created_at, updated_at, idempotency_key)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		o.ID, o.CustomerID, o.ItemName, o.Amount, o.Status, o.CreatedAt, o.UpdatedAt, idem,
	)
	if err == nil {
		return nil
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return ErrDuplicateIdempotencyKey
	}

	return err
}

func (r *orderRepository) GetByID(id string) (*domain.Order, error) {
	row := r.db.QueryRow(
		`SELECT id, customer_id, item_name, amount, status, created_at, updated_at, COALESCE(idempotency_key, '')
		 FROM orders WHERE id=$1`,
		id,
	)

	var o domain.Order
	err := row.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &o.CreatedAt, &o.UpdatedAt, &o.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *orderRepository) GetByIdempotencyKey(key string) (*domain.Order, error) {
	row := r.db.QueryRow(
		`SELECT id, customer_id, item_name, amount, status, created_at, updated_at, COALESCE(idempotency_key, '')
		 FROM orders WHERE idempotency_key=$1`,
		key,
	)

	var o domain.Order
	err := row.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &o.CreatedAt, &o.UpdatedAt, &o.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *orderRepository) UpdateStatus(id, status string) error {
	result, err := r.db.Exec("UPDATE orders SET status=$1, updated_at=NOW() WHERE id=$2", status, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
