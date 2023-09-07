package models

type AuthorizationRequst struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type BalanceWithdrawRequest struct {
	OrderNumber string  `json:"order"`
	Withdrawn   float64 `json:"sum"`
}

type GetOrdersReponse []OrderResponse

type OrderResponse struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

type GetBalanceResponse struct {
	Accrual   float64 `json:"current"`
	Withdranw float64 `json:"withdrawn"`
}

type GetWithdrawalsResponse []WithdrawalResponse

type WithdrawalResponse struct {
	Number    string  `json:"order"`
	Withdrawn float64 `json:"sum"`
	CreatedAt string  `json:"processed_at"`
}

type AccrualAPIGetOrderResponse struct {
	Number  string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}
