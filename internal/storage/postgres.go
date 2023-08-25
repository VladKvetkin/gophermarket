package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/VladKvetkin/gophermart/internal/entities"
	"github.com/jackc/pgerrcode"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrConflict         = errors.New("conflict")
	ErrNoRows           = errors.New("no rows")
	ErrNotEnoughAccrual = errors.New("not enough accrual")
)

type Storage interface {
	GetUser(context.Context, string, string) (string, error)
	GetUserOrders(context.Context, string) ([]entities.Order, error)
	GetOrCreateOrderIfNotExists(context.Context, string, string) (entities.Order, bool, error)
	GetUserAccrual(context.Context, string) (int, error)
	GetUserWithdrawn(context.Context, string) (int, error)
	GetUserWithdrawals(context.Context, string) ([]entities.Withdrawal, error)

	CreateUser(context.Context, string, string) (string, error)
	CreateOrder(context.Context, string, string) (string, error)
	CreateWithdraw(context.Context, string, string, int) (string, error)

	GetOrdersForAccrualer(context.Context) ([]entities.Order, error)
	UpdateOrder(context.Context, entities.Order, int, string) error

	runMigrations(context.Context) error
}

type PostgresStorage struct {
	db *sqlx.DB
}

func NewPostgresStorage(db *sqlx.DB) (Storage, error) {
	storage := &PostgresStorage{db: db}

	err := storage.runMigrations(context.Background())
	if err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *PostgresStorage) UpdateOrder(ctx context.Context, order entities.Order, accrual int, orderStatus string) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE orders SET status = $1, accrual = $2, updated_at=$3::timestamp WHERE id = $4;`,
		orderStatus, accrual, time.Now().UTC().Format(time.RFC3339), order.ID,
	); err != nil {
		return err
	}

	if accrual != 0 {
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE users SET bonuses = bonuses + $1 WHERE id = $2;`,
			accrual, order.UserID,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *PostgresStorage) GetOrdersForAccrualer(ctx context.Context) ([]entities.Order, error) {
	var orders []entities.Order

	err := s.db.SelectContext(ctx, &orders, "SELECT * FROM orders WHERE status NOT IN ($1,$2);", entities.OrderStatusProcessed, entities.OrderStatusInvalid)
	if err != nil {
		return nil, err
	}

	return orders, nil
}

func (s *PostgresStorage) CreateWithdraw(ctx context.Context, userID string, orderNumber string, withdrawn int) (string, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return "", err
	}

	defer tx.Rollback()

	currentAccrual, err := s.GetUserAccrual(ctx, userID)
	if err != nil {
		return "", err
	}

	if withdrawn > currentAccrual {
		return "", ErrNotEnoughAccrual
	}

	var withdrawID string

	row := tx.QueryRowxContext(
		ctx,
		`INSERT INTO orders_withdraw (number, withdrawn, user_id)
		VALUES ($1, $2, $3) RETURNING id;`,
		orderNumber, withdrawn, userID,
	)

	if err := row.Err(); err != nil {
		return "", err
	}

	if err := row.Scan(&withdrawID); err != nil {
		return "", err
	}

	_, err = tx.ExecContext(ctx, `UPDATE users SET bonuses=bonuses-$1, withdrawn=withdrawn+$1 WHERE id=$2`, withdrawn, userID)
	if err != nil {
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		return "", err
	}

	return withdrawID, nil
}

func (s *PostgresStorage) GetUserWithdrawals(ctx context.Context, userID string) ([]entities.Withdrawal, error) {
	var withdrawals []entities.Withdrawal

	err := s.db.SelectContext(ctx, &withdrawals, "SELECT * FROM orders_withdraw WHERE user_id = $1 ORDER BY created_at ASC;", userID)
	if err != nil {
		return nil, err
	}

	return withdrawals, nil
}

func (s *PostgresStorage) GetUserAccrual(ctx context.Context, userID string) (int, error) {
	var accrual int

	row := s.db.QueryRowxContext(ctx, "SELECT bonuses FROM users WHERE id = $1;", userID)

	if err := row.Err(); err != nil {
		return 0, err
	}

	if err := row.Scan(&accrual); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, err
	}

	return accrual, nil
}

func (s *PostgresStorage) GetUserWithdrawn(ctx context.Context, userID string) (int, error) {
	var withdrawn int

	row := s.db.QueryRowxContext(ctx, "SELECT withdrawn FROM users WHERE id = $1;", userID)

	if err := row.Err(); err != nil {
		return 0, err
	}

	if err := row.Scan(&withdrawn); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, err
	}

	return withdrawn, nil
}

func (s *PostgresStorage) GetOrCreateOrderIfNotExists(ctx context.Context, userID string, number string) (entities.Order, bool, error) {
	var order entities.Order

	tx, err := s.db.Beginx()
	if err != nil {
		return order, false, err
	}

	defer tx.Rollback()

	row := tx.QueryRowxContext(ctx, "SELECT id, number, status, created_at, updated_at, user_id FROM orders WHERE number = $1;", number)

	if err := row.Err(); err != nil {
		return order, false, err
	}

	err = row.Scan(&order.ID, &order.Number, &order.Status, &order.CreatedAt, &order.UpdatedAt, &order.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			row := tx.QueryRowxContext(
				ctx,
				`INSERT INTO orders (number, status, user_id)
				VALUES ($1, $2, $3) RETURNING id;`,
				number, entities.OrderStatusNew, userID,
			)

			if err := row.Err(); err != nil {
				return order, false, err
			}

			var orderID string
			if err := row.Scan(&orderID); err != nil {
				return order, false, err
			}

			order.ID = orderID
			order.Number = number
			order.Status = entities.OrderStatusNew
			order.UserID = userID
		}

		return order, true, tx.Commit()
	}

	return order, false, tx.Commit()
}

func (s *PostgresStorage) CreateOrder(ctx context.Context, userID string, number string) (string, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return "", err
	}

	defer tx.Rollback()

	var orderID string

	row := tx.QueryRowxContext(
		ctx,
		`INSERT INTO orders (number, status, user_id)
		VALUES ($1, $2, $3) RETURNING id;`,
		number, entities.OrderStatusNew, userID,
	)

	if err := row.Err(); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pgerrcode.IsIntegrityConstraintViolation(string(pqErr.Code)) {
			return "", ErrConflict

		}

		return "", err
	}

	if err := row.Scan(&orderID); err != nil {
		return "", err
	}

	return orderID, tx.Commit()
}

func (s *PostgresStorage) GetUserOrders(ctx context.Context, userID string) ([]entities.Order, error) {
	var orders []entities.Order

	err := s.db.SelectContext(ctx, &orders, "SELECT * FROM orders WHERE user_id = $1;", userID)
	if err != nil {
		return nil, err
	}

	return orders, nil
}

func (s *PostgresStorage) GetUser(ctx context.Context, login string, passwordHash string) (string, error) {
	var userID string

	row := s.db.QueryRowxContext(ctx, "SELECT id FROM users WHERE login = $1 AND password = $2;", login, passwordHash)

	if err := row.Err(); err != nil {
		return "", err
	}

	err := row.Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNoRows
		}

		return "", err
	}

	return userID, nil
}

func (s *PostgresStorage) CreateUser(ctx context.Context, login string, passwordHash string) (string, error) {
	var userID string

	row := s.db.QueryRowxContext(
		ctx,
		`INSERT INTO users (login, password)
		VALUES ($1, $2) RETURNING id;`,
		login, passwordHash,
	)

	if err := row.Err(); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pgerrcode.IsIntegrityConstraintViolation(string(pqErr.Code)) {
			return "", ErrConflict

		}

		return "", err
	}

	if err := row.Scan(&userID); err != nil {
		return "", err
	}

	return userID, nil
}

func (s *PostgresStorage) runMigrations(ctx context.Context) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(
		ctx,
		`
		CREATE TABLE IF NOT EXISTS users(
			id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
			login TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			bonuses INT DEFAULT 0,
			withdrawn INT DEFAULT 0
		);
		`,
	)

	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`
		CREATE TABLE IF NOT EXISTS orders(
			id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
			number VARCHAR NOT NULL UNIQUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			status VARCHAR NOT NULL,
			user_id uuid NOT NULL,
			accrual INT DEFAULT 0,
			CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);
		`,
	)

	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`
		CREATE TABLE IF NOT EXISTS orders_withdraw(
			id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
			number VARCHAR NOT NULL UNIQUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			user_id uuid NOT NULL,
			withdrawn INT DEFAULT 0,
			CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);
		`,
	)

	if err != nil {
		return err
	}

	return tx.Commit()
}
