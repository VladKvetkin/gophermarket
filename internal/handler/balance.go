package handler

import (
	"encoding/json"
	"net/http"

	"github.com/VladKvetkin/gophermart/internal/models"
	"github.com/VladKvetkin/gophermart/internal/services/converter"
	"go.uber.org/zap"
)

func (h *Handler) GetBalance(res http.ResponseWriter, req *http.Request) {
	userID := h.getUserIDFromReqContext(req)
	if userID == "" {
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
