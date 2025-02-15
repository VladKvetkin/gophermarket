package entities

import (
	"time"
)

const (
	OrderStatusNew        = "NEW"
	OrderStatusProcessing = "PROCESSING"
	OrderStatusProcessed  = "PROCESSED"
	OrderStatusInvalid    = "INVALID"
)

type Order struct {
	ID        string    `db:"id"`
	Number    string    `db:"number"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Status    string    `db:"status"`
	UserID    string    `db:"user_id"`
	Accrual   int       `db:"accrual"`
}

type Withdrawal struct {
	ID        string    `db:"id"`
	Number    string    `db:"number"`
	CreatedAt time.Time `db:"created_at"`
	UserID    string    `db:"user_id"`
	Withdrawn int       `db:"withdrawn"`
}
