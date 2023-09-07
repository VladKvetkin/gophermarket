package handler

import (
	"errors"
	"net/http"

	"github.com/VladKvetkin/gophermart/internal/storage"
	"go.uber.org/zap"
)

func (h *Handler) Register(res http.ResponseWriter, req *http.Request) {
	requestModel, err := h.validateAuthorizationRequest(req)
	if err != nil {
		zap.L().Info("error validate register request: %w", zap.Error(err))

		res.WriteHeader(http.StatusBadRequest)
		return
	}

	userID, err := h.storage.CreateUser(req.Context(), requestModel.Login, h.generatePasswordHash(requestModel.Password))
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			zap.L().Info("error login already exists: %w", zap.Error(err))

			res.WriteHeader(http.StatusConflict)
			return
		}

		zap.L().Info("error create user: %w", zap.Error(err))

		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.generateTokenAndSetCookie(res, userID)
}
