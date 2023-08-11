package entities

import (
	"time"
)

type OrderWithdraw struct {
	ID        string    `db:"id"`
	Number    string    `db:"number"`
	CreatedAt time.Time `db:"created_at"`
	UserID    string    `db:"user_id"`
	Withdrawn int       `db:"withdrawn"`
}
