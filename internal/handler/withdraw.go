package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/VladKvetkin/gophermart/internal/middleware"
	"github.com/VladKvetkin/gophermart/internal/models"
	"github.com/VladKvetkin/gophermart/internal/services/converter"
	"go.uber.org/zap"
)

func (h *Handler) GetWithdrawals(res http.ResponseWriter, req *http.Request) {
	userID, ok := req.Context().Value(middleware.UserIDKey{}).(string)
	if !ok {
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
