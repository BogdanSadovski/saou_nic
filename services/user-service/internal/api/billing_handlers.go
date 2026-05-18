package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/real-ass/user-service/internal/domain"
	"github.com/real-ass/user-service/internal/service"
)

func (h *Handler) GetBillingPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.subscriptionService.GetAvailablePlans(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to fetch plans")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]any{"plans": plans, "test_mode": true})
}

func (h *Handler) GetMySubscription(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sub, err := h.subscriptionService.GetUserSubscription(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to fetch subscription")
		return
	}
	if sub == nil {
		respondWithJSON(w, http.StatusOK, map[string]any{"subscription": nil, "test_mode": true})
		return
	}

	plan := h.subscriptionService.GetPlanByTier(sub.Tier)
	respondWithJSON(w, http.StatusOK, map[string]any{
		"subscription": domain.SubscriptionResponse{
			ID:           sub.ID,
			UserID:       sub.UserID,
			Tier:         sub.Tier,
			Status:       sub.Status,
			StartDate:    sub.StartDate,
			EndDate:      sub.EndDate,
			RenewalDate:  sub.RenewalDate,
			TrialEndDate: sub.TrialEndDate,
			IsActive:     sub.IsActive,
			Plan:         plan,
		},
		"test_mode": true,
	})
}

func (h *Handler) CreateCheckoutIntent(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req domain.CreateCheckoutIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	intent, err := h.paymentService.CreateCheckoutIntent(r.Context(), userID, req)
	if err != nil {
		fmt.Printf("DEBUG: CreateCheckoutIntent error: %v\n", err)
		if errors.Is(err, service.ErrTrialDoesNotRequire) {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create checkout intent: %v", err))
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]any{
		"intent":    intent,
		"test_mode": true,
		"next_step": "POST /billing/checkout-intents/{id}/confirm",
	})
}

func (h *Handler) ConfirmCheckoutIntent(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	intentID, err := uuid.Parse(mux.Vars(r)["intentID"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid intent ID")
		return
	}

	intent, sub, tx, err := h.paymentService.ConfirmCheckoutIntent(r.Context(), userID, intentID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIntentNotFound):
			respondWithError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrIntentWrongState):
			respondWithError(w, http.StatusConflict, err.Error())
		default:
			respondWithError(w, http.StatusInternalServerError, "failed to confirm checkout intent")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]any{
		"intent":       intent,
		"subscription": sub,
		"transaction":  tx,
		"test_mode":    true,
	})
}

func (h *Handler) TestPaymentWebhook(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req domain.TestWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	intent, err := h.paymentService.ApplyTestWebhook(r.Context(), userID, req.IntentID, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIntentNotFound):
			respondWithError(w, http.StatusNotFound, err.Error())
		default:
			respondWithError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]any{"intent": intent, "test_mode": true})
}

func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 20
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = parsed
	}

	txs, err := h.paymentService.ListTransactions(r.Context(), userID, limit)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to list transactions")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]any{"transactions": txs, "test_mode": true})
}

func (h *Handler) CancelMySubscription(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.subscriptionService.CancelSubscription(r.Context(), userID); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]any{"message": "subscription canceled", "test_mode": true})
}
