package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/VladKvetkin/gophermart/internal/middleware"
	"github.com/VladKvetkin/gophermart/internal/models"
	"github.com/VladKvetkin/gophermart/internal/services/jwttoken"
	"github.com/VladKvetkin/gophermart/internal/storage"
	"go.uber.org/zap"
)

func (h *Handler) Login(res http.ResponseWriter, req *http.Request) {
	requestModel, err := h.validateAuthorizationRequest(req)
	if err != nil {
		zap.L().Info("error validate login request: %w", zap.Error(err))

		res.WriteHeader(http.StatusBadRequest)
		return
	}

	userID, err := h.storage.GetUser(req.Context(), requestModel.Login, h.generatePasswordHash(requestModel.Password))
	if err != nil {
		if errors.Is(err, storage.ErrNoRows) {
			zap.L().Info("error login and password hash not found: %w", zap.Error(err))

			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		zap.L().Info("error get user: %w", zap.Error(err))

		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.generateTokenAndSetCookie(res, userID)
}

func (h *Handler) generatePasswordHash(password string) string {
	passwordHash := sha256.Sum256([]byte(password))
	return base64.StdEncoding.EncodeToString(passwordHash[:])
}

func (h *Handler) validateAuthorizationRequest(req *http.Request) (models.AuthorizationRequst, error) {
	var requestModel models.AuthorizationRequst

	jsonDecoder := json.NewDecoder(req.Body)

	if err := jsonDecoder.Decode(&requestModel); err != nil {
		return models.AuthorizationRequst{}, fmt.Errorf("cannot decode request to json: %w", err)
	}

	if requestModel.Login == "" || requestModel.Password == "" {
		return models.AuthorizationRequst{}, fmt.Errorf("empty login or password")
	}

	return requestModel, nil
}

func (h *Handler) generateTokenAndSetCookie(res http.ResponseWriter, userID string) {
	accessToken, err := jwttoken.Generate(userID)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(res, &http.Cookie{
		Name:  middleware.TokenCookieName,
		Value: accessToken,
		Path:  "/",
	})

	res.WriteHeader(http.StatusOK)
}
