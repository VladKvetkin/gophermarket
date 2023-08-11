package handler

import (
	"encoding/json"
	"net/http"

	"github.com/VladKvetkin/gophermart/internal/middleware"
	"github.com/VladKvetkin/gophermart/internal/models"
	"github.com/VladKvetkin/gophermart/internal/services/converter"
	"github.com/VladKvetkin/gophermart/internal/services/validation"
	"go.uber.org/zap"
)

func (h *Handler) GetBalance(res http.ResponseWriter, req *http.Request) {
	userID, ok := req.Context().Value(middleware.UserIDKey{}).(string)
	if !ok {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	currentAccrual, err := h.storage.GetUserAccrual(req.Context(), userID)
	if err != nil {
		zap.L().Info("error get user accrual: %w", zap.Error(err))

		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	withdrawn, err := h.storage.GetUserWithdrawn(req.Context(), userID)
	if err != nil {
		zap.L().Info("error get user withdrawn: %w", zap.Error(err))

		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := models.GetBalanceResponse{
		Accrual:   converter.FormatAccrual(currentAccrual),
		Withdranw: converter.FormatAccrual(withdrawn),
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	jsonEncoder := json.NewEncoder(res)
	if err := jsonEncoder.Encode(response); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		zap.L().Info("cannot encode response JSON body: %w", zap.Error(err))
	}
}

func (h *Handler) Withdraw(res http.ResponseWriter, req *http.Request) {
	userID, ok := req.Context().Value(middleware.UserIDKey{}).(string)
	if !ok {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	var balanceWithdrawRequest models.BalanceWithdrawRequest
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&balanceWithdrawRequest); err != nil {
		zap.L().Info("cannot decode request to json: %w", zap.Error(err))

		res.WriteHeader(http.StatusBadRequest)
		return
	}

	if balanceWithdrawRequest.Withdrawn == 0 {
		zap.L().Info("balance withdrawn request with sum=0")

		res.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := validation.LuhnValidate(balanceWithdrawRequest.OrderNumber); err != nil {
		zap.L().Info("luhn validation failed: %w", zap.Error(err))

		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	currentAccrual, err := h.storage.GetUserAccrual(req.Context(), userID)
	if err != nil {
		zap.L().Info("error get user accrual: %w", zap.Error(err))

		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if balanceWithdrawRequest.Withdrawn > converter.FormatAccrual(currentAccrual) {
		zap.L().Info("error not enough user accrual for withdrawn: %w", zap.Error(err))

		res.WriteHeader(http.StatusPaymentRequired)
		return
	}

	if _, err := h.storage.CreateWithdraw(req.Context(), userID, balanceWithdrawRequest.OrderNumber, converter.ConvertAccrual(balanceWithdrawRequest.Withdrawn)); err != nil {
		zap.L().Info("error create withdraw: %w", zap.Error(err))

		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}
