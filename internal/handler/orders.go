package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/VladKvetkin/gophermart/internal/models"
	"github.com/VladKvetkin/gophermart/internal/services/converter"
	"github.com/VladKvetkin/gophermart/internal/services/validation"
	"go.uber.org/zap"
)

func (h *Handler) SaveOrder(res http.ResponseWriter, req *http.Request) {
	userID := h.getUserIDFromReqContext(req)
	if userID == "" {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	orderNumber, err := io.ReadAll(req.Body)
	if err != nil || len(orderNumber) == 0 {
		zap.L().Info("cannot read order number from request: %w", zap.Error(err))

		res.WriteHeader(http.StatusBadRequest)
		return
	}

	orderNumberString := string(orderNumber)

	if err := validation.LuhnValidate(orderNumberString); err != nil {
		zap.L().Info("luhn validation failed: %w", zap.Error(err))

		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	order, isNewOrder, err := h.storage.GetOrCreateOrderIfNotExists(req.Context(), userID, orderNumberString)
	if err != nil {
		zap.L().Info("error get or create order: %w", zap.Error(err))

		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if isNewOrder {
		res.WriteHeader(http.StatusAccepted)
		return
	}

	if order.UserID == userID {
		res.WriteHeader(http.StatusOK)
		return
	}

	res.WriteHeader(http.StatusConflict)
}

func (h *Handler) GetOrders(res http.ResponseWriter, req *http.Request) {
	userID := h.getUserIDFromReqContext(req)
	if userID == "" {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	orders, err := h.storage.GetUserOrders(req.Context(), userID)
	if err != nil {
		zap.L().Info("error get user orders from database: %w", zap.Error(err))

		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		zap.L().Info("empty user orders: %v", zap.String("UserID", userID))

		res.WriteHeader(http.StatusNoContent)
		return
	}

	responseOrders := make(models.GetOrdersReponse, 0, len(orders))
	for _, order := range orders {
		responseOrder := models.OrderResponse{
			Number:     order.Number,
			Status:     order.Status,
			UploadedAt: order.CreatedAt.Format(time.RFC3339),
		}

		if order.Accrual != 0 {
			responseOrder.Accrual = converter.FormatAccrual(order.Accrual)
		}

		responseOrders = append(responseOrders, responseOrder)
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	jsonEncoder := json.NewEncoder(res)
	if err := jsonEncoder.Encode(responseOrders); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		zap.L().Info("cannot encode response JSON body: %w", zap.Error(err))
	}
}
