package handler

import "github.com/VladKvetkin/gophermart/internal/storage"

type Handler struct {
	storage storage.Storage
}

func NewHandler(storage storage.Storage) *Handler {
	return &Handler{
		storage: storage,
	}
}
