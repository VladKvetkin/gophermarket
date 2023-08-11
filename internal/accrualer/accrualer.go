package accrualer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/VladKvetkin/gophermart/internal/entities"
	"github.com/VladKvetkin/gophermart/internal/models"
	"github.com/VladKvetkin/gophermart/internal/services/converter"
	"github.com/VladKvetkin/gophermart/internal/storage"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
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
	ticker := time.NewTicker(3 * time.Second)
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
	orders, err := ac.storage.GetOrdersForAccrualer(ctx)
	if err != nil {
		return err
	}

	if len(orders) == 0 {
		return nil
	}

	client := ac.initClient()

	for _, order := range orders {
		response, err := ac.checkOrderAccrual(client, order)
		if err != nil {
			zap.L().Info("error failed to check order accrual %w", zap.Error(err))

			continue
		}

		err = ac.updateOrder(ctx, order, response)
		if err != nil {
			zap.L().Info("error failed to update order accrual %w", zap.Error(err))

			continue
		}
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

func (ac *Accrualer) checkOrderAccrual(client *resty.Client, order entities.Order) (models.AccrualAPIGetOrderResponse, error) {
	url, err := ac.getOrderPath(order.Number)
	if err != nil {
		return models.AccrualAPIGetOrderResponse{}, err
	}

	request := client.R().SetDoNotParseResponse(true)

	response, err := request.Get(url)
	if err != nil {
		return models.AccrualAPIGetOrderResponse{}, err
	}

	defer response.RawBody().Close()

	if response.StatusCode() != http.StatusOK {
		return models.AccrualAPIGetOrderResponse{}, fmt.Errorf("error failed to get order accrual, invalid status: %v", response.Status())
	}

	responseOrderAccrual := models.AccrualAPIGetOrderResponse{}

	jsonDecoder := json.NewDecoder(response.RawBody())

	if err := jsonDecoder.Decode(&responseOrderAccrual); err != nil {
		return models.AccrualAPIGetOrderResponse{}, fmt.Errorf("cannot decode response get order accrual to json: %w", err)
	}

	return responseOrderAccrual, nil
}

func (ac *Accrualer) getOrderPath(orderNumber string) (string, error) {
	url, err := url.JoinPath(ac.apiAddress, getOrderPath, orderNumber)
	if err != nil {
		return "", err
	}

	return url, nil
}
