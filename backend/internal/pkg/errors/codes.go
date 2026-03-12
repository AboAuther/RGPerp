package errors

import "net/http"

type AppError struct {
	HTTPStatus int    `json:"-"`
	Code       int    `json:"code"`
	Message    string `json:"message"`
}

func (e *AppError) Error() string {
	return e.Message
}

func New(httpStatus, code int, message string) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
	}
}

// Auth errors (10xxx)
var (
	ErrInvalidAddress   = New(http.StatusBadRequest, 10001, "invalid wallet address")
	ErrNonceExpired     = New(http.StatusUnauthorized, 10002, "nonce expired or already used")
	ErrSignatureInvalid = New(http.StatusUnauthorized, 10003, "signature verification failed")
	ErrUnauthorized     = New(http.StatusUnauthorized, 10004, "unauthorized")
	ErrTokenExpired     = New(http.StatusUnauthorized, 10005, "token expired")
)

// Account errors (20xxx)
var (
	ErrAccountNotFound     = New(http.StatusNotFound, 20001, "account not found")
	ErrInsufficientBalance = New(http.StatusBadRequest, 20002, "insufficient balance")
	ErrAccountFrozen       = New(http.StatusForbidden, 20003, "account is frozen")
)

// Order errors (30xxx)
var (
	ErrSymbolNotFound       = New(http.StatusBadRequest, 30001, "symbol not found or not trading")
	ErrInvalidOrderSize     = New(http.StatusBadRequest, 30002, "invalid order size")
	ErrInvalidLeverage      = New(http.StatusBadRequest, 30003, "invalid leverage")
	ErrInsufficientMargin   = New(http.StatusBadRequest, 30004, "insufficient margin")
	ErrMaxPositionExceeded  = New(http.StatusBadRequest, 30005, "max position notional exceeded")
	ErrOrderRejected        = New(http.StatusBadRequest, 30006, "order rejected by risk check")
	ErrDuplicateClientOrder = New(http.StatusConflict, 30007, "duplicate client order id")
)

// Position errors (40xxx)
var (
	ErrPositionNotFound = New(http.StatusNotFound, 40001, "position not found")
	ErrNoOpenPosition   = New(http.StatusBadRequest, 40002, "no open position for this symbol")
)

// Withdrawal errors (50xxx)
var (
	ErrWithdrawalPending   = New(http.StatusConflict, 50001, "withdrawal already pending")
	ErrMinWithdrawalAmount = New(http.StatusBadRequest, 50002, "amount below minimum")
)

// System errors (90xxx)
var (
	ErrPriceUnavailable = New(http.StatusServiceUnavailable, 90004, "price unavailable")
	ErrInternal         = New(http.StatusInternalServerError, 90001, "internal server error")
	ErrBadRequest       = New(http.StatusBadRequest, 90002, "bad request")
	ErrRateLimit        = New(http.StatusTooManyRequests, 90003, "rate limit exceeded")
)
