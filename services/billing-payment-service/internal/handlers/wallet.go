package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/models"
	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/service"
)

// CreateWallet handles wallet creation requests
func CreateWallet(billingService *service.BillingService, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.WalletCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Failed to decode wallet creation request", zap.Error(err))
			writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		wallet, err := billingService.CreateWallet(r.Context(), &req)
		if err != nil {
			logger.Error("Failed to create wallet", zap.Error(err))
			if billingErr, ok := err.(*models.BillingError); ok {
				writeErrorResponse(w, getHTTPStatusFromBillingError(billingErr), billingErr.Message, err)
			} else {
				writeErrorResponse(w, http.StatusInternalServerError, "Failed to create wallet", err)
			}
			return
		}

		writeJSONResponse(w, http.StatusCreated, wallet)
	}
}

// GetWalletBalance handles wallet balance requests
func GetWalletBalance(billingService *service.BillingService, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletIDStr := chi.URLParam(r, "walletID")
		walletID, err := uuid.Parse(walletIDStr)
		if err != nil {
			logger.Error("Invalid wallet ID", zap.String("wallet_id", walletIDStr), zap.Error(err))
			writeErrorResponse(w, http.StatusBadRequest, "Invalid wallet ID", err)
			return
		}

		balance, err := billingService.GetWalletBalance(r.Context(), walletID)
		if err != nil {
			logger.Error("Failed to get wallet balance", zap.String("wallet_id", walletIDStr), zap.Error(err))
			if billingErr, ok := err.(*models.BillingError); ok {
				writeErrorResponse(w, getHTTPStatusFromBillingError(billingErr), billingErr.Message, err)
			} else {
				writeErrorResponse(w, http.StatusInternalServerError, "Failed to get wallet balance", err)
			}
			return
		}

		writeJSONResponse(w, http.StatusOK, balance)
	}
}

// DepositTokens handles token deposit requests
func DepositTokens(billingService *service.BillingService, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletIDStr := chi.URLParam(r, "walletID")
		walletID, err := uuid.Parse(walletIDStr)
		if err != nil {
			logger.Error("Invalid wallet ID", zap.String("wallet_id", walletIDStr), zap.Error(err))
			writeErrorResponse(w, http.StatusBadRequest, "Invalid wallet ID", err)
			return
		}

		var req models.DepositRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Failed to decode deposit request", zap.Error(err))
			writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		req.WalletID = walletID

		transaction, err := billingService.ProcessDeposit(r.Context(), &req)
		if err != nil {
			logger.Error("Failed to process deposit", zap.Error(err))
			if billingErr, ok := err.(*models.BillingError); ok {
				writeErrorResponse(w, getHTTPStatusFromBillingError(billingErr), billingErr.Message, err)
			} else {
				writeErrorResponse(w, http.StatusInternalServerError, "Failed to process deposit", err)
			}
			return
		}

		logger.Info("Deposit processed successfully",
			zap.String("wallet_id", walletID.String()),
			zap.String("amount", req.Amount.String()),
			zap.String("transaction_id", transaction.ID.String()),
		)

		writeJSONResponse(w, http.StatusOK, transaction)
	}
}

// WithdrawTokens handles token withdrawal requests
func WithdrawTokens(billingService *service.BillingService, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletIDStr := chi.URLParam(r, "walletID")
		walletID, err := uuid.Parse(walletIDStr)
		if err != nil {
			logger.Error("Invalid wallet ID", zap.String("wallet_id", walletIDStr), zap.Error(err))
			writeErrorResponse(w, http.StatusBadRequest, "Invalid wallet ID", err)
			return
		}

		var req models.WithdrawalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Failed to decode withdrawal request", zap.Error(err))
			writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		req.WalletID = walletID

		transaction, err := billingService.ProcessWithdrawal(r.Context(), &req)
		if err != nil {
			logger.Error("Failed to process withdrawal", zap.Error(err))
			if billingErr, ok := err.(*models.BillingError); ok {
				writeErrorResponse(w, getHTTPStatusFromBillingError(billingErr), billingErr.Message, err)
			} else {
				writeErrorResponse(w, http.StatusInternalServerError, "Failed to process withdrawal", err)
			}
			return
		}

		logger.Info("Withdrawal processed successfully",
			zap.String("wallet_id", walletID.String()),
			zap.String("amount", req.Amount.String()),
			zap.String("to_address", req.ToAddress),
			zap.String("transaction_id", transaction.ID.String()),
		)

		writeJSONResponse(w, http.StatusOK, transaction)
	}
}

// GetTransactionHistory handles transaction history requests
func GetTransactionHistory(billingService *service.BillingService, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletIDStr := chi.URLParam(r, "walletID")
		walletID, err := uuid.Parse(walletIDStr)
		if err != nil {
			logger.Error("Invalid wallet ID", zap.String("wallet_id", walletIDStr), zap.Error(err))
			writeErrorResponse(w, http.StatusBadRequest, "Invalid wallet ID", err)
			return
		}

		// Parse query parameters
		req := &models.TransactionHistoryRequest{
			WalletID: &walletID,
			Limit:    50, // Default limit
			Offset:   0,  // Default offset
		}

		// Parse limit and offset from query parameters
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
				req.Limit = limit
			}
		}

		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
				req.Offset = offset
			}
		}

		history, err := billingService.GetTransactionHistory(r.Context(), req)
		if err != nil {
			logger.Error("Failed to get transaction history", zap.Error(err))
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to get transaction history", err)
			return
		}

		writeJSONResponse(w, http.StatusOK, history)
	}
}

// Helper functions

// writeJSONResponse writes a JSON response
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// If we can't encode the response, log the error
		// but don't try to write another response as headers are already sent
		zap.L().Error("Failed to encode JSON response", zap.Error(err))
	}
}

// writeErrorResponse writes an error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"error":  message,
		"status": statusCode,
	}

	if err != nil {
		errorResponse["details"] = err.Error()
	}

	if encodeErr := json.NewEncoder(w).Encode(errorResponse); encodeErr != nil {
		zap.L().Error("Failed to encode error response", zap.Error(encodeErr))
	}
}

// getHTTPStatusFromBillingError maps billing errors to HTTP status codes
func getHTTPStatusFromBillingError(err *models.BillingError) int {
	switch err.Code {
	case models.ErrCodeWalletNotFound, models.ErrCodeTransactionNotFound, models.ErrCodeSessionNotFound:
		return http.StatusNotFound
	case models.ErrCodeWalletExists, models.ErrCodeSessionActive:
		return http.StatusConflict
	case models.ErrCodeInsufficientFunds, models.ErrCodeInvalidAmount, models.ErrCodeValidationFailed:
		return http.StatusBadRequest
	case models.ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case models.ErrCodeForbidden:
		return http.StatusForbidden
	case models.ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case models.ErrCodeSolanaConnection, models.ErrCodeSolanaTransaction:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}
