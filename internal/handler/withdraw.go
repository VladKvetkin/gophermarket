package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/VladKvetkin/gophermart/internal/models"
	"github.com/VladKvetkin/gophermart/internal/services/converter"
	"github.com/VladKvetkin/gophermart/internal/services/validation"
	"github.com/VladKvetkin/gophermart/internal/storage"
	"go.uber.org/zap"
)

func (h *Handler) GetWithdrawals(res http.ResponseWriter, req *http.Request) {
	userID := h.getUserIDFromReqContext(req)
	if userID == "" {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.storage.GetUserWithdrawals(req.Context(), userID)
	if err != nil {
		zap.L().Info("error get user withdrawals: %w", zap.Error(err))

		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		zap.L().Info("empty user withdrawals: %v", zap.String("UserID", userID))

		res.WriteHeader(http.StatusNoContent)
		return
	}

	responseWithdrawals := make(models.GetWithdrawalsResponse, 0, len(withdrawals))
	for _, withdrawal := range withdrawals {
		responseWithdrawal := models.WithdrawalResponse{
			Number:    withdrawal.Number,
			Withdrawn: converter.FormatAccrual(withdrawal.Withdrawn),
			CreatedAt: withdrawal.CreatedAt.Format(time.RFC3339),
		}

		responseWithdrawals = append(responseWithdrawals, responseWithdrawal)
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	jsonEncoder := json.NewEncoder(res)
	if err := jsonEncoder.Encode(responseWithdrawals); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		zap.L().Info("cannot encode response JSON body: %w", zap.Error(err))
	}
}

func (h *Handler) Withdraw(res http.ResponseWriter, req *http.Request) {
	userID := h.getUserIDFromReqContext(req)
	if userID == "" {
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

	if _, err := h.storage.CreateWithdraw(req.Context(), userID, balanceWithdrawRequest.OrderNumber, converter.ConvertAccrual(balanceWithdrawRequest.Withdrawn)); err != nil {
		if errors.Is(err, storage.ErrNotEnoughAccrual) {
			zap.L().Info("error not enough user accrual for withdrawn: %w", zap.Error(err))

			res.WriteHeader(http.StatusPaymentRequired)
			return
		}

		zap.L().Info("error create withdraw: %w", zap.Error(err))

		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}
