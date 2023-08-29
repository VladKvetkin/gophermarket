package accrualer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/VladKvetkin/gophermart/internal/entities"
	"github.com/VladKvetkin/gophermart/internal/models"
	"github.com/VladKvetkin/gophermart/internal/services/converter"
	"github.com/VladKvetkin/gophermart/internal/storage"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	getOrderPath = "/api/orders/"
)

type Accrualer struct {
	apiAddress string
	storage    storage.Storage
}

func NewAccrualer(apiAddress string, storage storage.Storage) *Accrualer {
	return &Accrualer{
		apiAddress: apiAddress,
		storage:    storage,
	}
}

func (ac *Accrualer) Start(ctx context.Context) error {
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	if err := ac.selectAndUpdateOrders(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
			if err := ac.selectAndUpdateOrders(ctx); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (ac *Accrualer) selectAndUpdateOrders(ctx context.Context) error {
	var (
		ordersLimit   = 1000
		workersNumber = 10
	)

	eg, ctx := errgroup.WithContext(ctx)
	client := ac.initClient()

	for i := 0; i < workersNumber; i++ {
		i := i
		eg.Go(func() error {
			orders, err := ac.storage.GetOrdersForAccrualer(ctx, ordersLimit*i, ordersLimit)
			if err != nil {
				return err
			}

			if len(orders) == 0 {
				return nil
			}

			retryAfter := 0
			for _, order := range orders {
				response, err, retryAfter := ac.checkOrderAccrual(client, order)
				if err != nil {
					zap.L().Info("error failed to check order accrual %w", zap.Error(err))

					continue
				}

				if retryAfter > 0 {
					break
				}

				err = ac.updateOrder(ctx, order, response)
				if err != nil {
					zap.L().Info("error failed to update order accrual %w", zap.Error(err))

					continue
				}
			}

			if retryAfter > 0 {
				time.Sleep(time.Duration(retryAfter) * time.Second)
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (ac *Accrualer) updateOrder(ctx context.Context, order entities.Order, updateFields models.AccrualAPIGetOrderResponse) error {
	return ac.storage.UpdateOrder(ctx, order, converter.ConvertAccrual(updateFields.Accrual), updateFields.Status)
}

func (ac *Accrualer) initClient() *resty.Client {
	client := resty.New()

	client.
		SetRetryCount(3).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(10 * time.Second)

	return client
}

func (ac *Accrualer) checkOrderAccrual(client *resty.Client, order entities.Order) (models.AccrualAPIGetOrderResponse, error, int) {
	url, err := ac.getOrderPath(order.Number)
	if err != nil {
		return models.AccrualAPIGetOrderResponse{}, err, 0
	}

	request := client.R().SetDoNotParseResponse(true)

	response, err := request.Get(url)
	if err != nil {
		return models.AccrualAPIGetOrderResponse{}, err, 0
	}

	defer response.RawBody().Close()

	if response.StatusCode() == http.StatusNoContent {
		return models.AccrualAPIGetOrderResponse{
			Number: order.Number,
			Status: entities.OrderStatusNew,
		}, nil, 0
	} else if response.StatusCode() == http.StatusTooManyRequests {
		retryAfter, err := strconv.Atoi(response.Header().Get("Retry-After"))
		if err != nil {
			return models.AccrualAPIGetOrderResponse{}, fmt.Errorf("error failed to parse Retry-After value, err: %w", err), 0
		}

		return models.AccrualAPIGetOrderResponse{}, nil, retryAfter
	}

	if response.StatusCode() != http.StatusOK {
		return models.AccrualAPIGetOrderResponse{}, fmt.Errorf("error failed to get order accrual, invalid status: %v", response.Status()), 0
	}

	responseOrderAccrual := models.AccrualAPIGetOrderResponse{}

	jsonDecoder := json.NewDecoder(response.RawBody())

	if err := jsonDecoder.Decode(&responseOrderAccrual); err != nil {
		return models.AccrualAPIGetOrderResponse{}, fmt.Errorf("cannot decode response get order accrual to json: %w", err), 0
	}

	return responseOrderAccrual, nil, 0
}

func (ac *Accrualer) getOrderPath(orderNumber string) (string, error) {
	url, err := url.JoinPath(ac.apiAddress, getOrderPath, orderNumber)
	if err != nil {
		return "", err
	}

	return url, nil
}
