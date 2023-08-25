package handler

import (
	"net/http"

	"github.com/VladKvetkin/gophermart/internal/middleware"
	"github.com/VladKvetkin/gophermart/internal/storage"
)

type Handler struct {
	storage storage.Storage
}

func NewHandler(storage storage.Storage) *Handler {
	return &Handler{
		storage: storage,
	}
}

func (h *Handler) getUserIDFromReqContext(req *http.Request) string {
	userID, ok := req.Context().Value(middleware.UserIDKey{}).(string)
	if !ok {
		return ""
	}

	return userID
}
