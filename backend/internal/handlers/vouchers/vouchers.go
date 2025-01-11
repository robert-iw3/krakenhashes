package vouchers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ZerkerEOD/hashdom/backend/internal/services"
	"github.com/ZerkerEOD/hashdom/backend/pkg/debug"
	"github.com/gorilla/mux"
)

// GenerateVoucherRequest represents the request to generate a voucher
type GenerateVoucherRequest struct {
	UserID       string `json:"userId"`
	ExpiresIn    int64  `json:"expiresIn"` // Duration in seconds
	IsContinuous bool   `json:"isContinuous"`
}

type VoucherHandler struct {
	service *services.ClaimVoucherService
}

func NewVoucherHandler(service *services.ClaimVoucherService) *VoucherHandler {
	return &VoucherHandler{service: service}
}

// GenerateVoucher handles voucher generation
func (h *VoucherHandler) GenerateVoucher(w http.ResponseWriter, r *http.Request) {
	debug.Info("Creating temporary voucher")

	// Get user ID from context
	userID := r.Context().Value("user_id").(string)
	if userID == "" {
		debug.Error("user ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req GenerateVoucherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		debug.Error("failed to decode request body: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Create voucher
	voucher, err := h.service.CreateTempVoucher(r.Context(), userID, time.Duration(req.ExpiresIn)*time.Second, req.IsContinuous)
	if err != nil {
		debug.Error("failed to create voucher: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	debug.Info("Successfully created voucher: %s", voucher.Code)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(voucher)
}

// ListVouchers handles listing all active vouchers
func (h *VoucherHandler) ListVouchers(w http.ResponseWriter, r *http.Request) {
	debug.Info("Listing active vouchers")

	vouchers, err := h.service.ListVouchers(r.Context())
	if err != nil {
		debug.Error("failed to list vouchers: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	debug.Info("Found %d active vouchers", len(vouchers))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vouchers)
}

// DeactivateVoucher handles voucher deactivation
func (h *VoucherHandler) DeactivateVoucher(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	if code == "" {
		debug.Error("missing voucher code")
		http.Error(w, "Missing voucher code", http.StatusBadRequest)
		return
	}

	debug.Info("Deactivating voucher: %s", code)

	if err := h.service.DisableVoucher(r.Context(), code); err != nil {
		debug.Error("failed to disable voucher: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	debug.Info("Successfully deactivated voucher: %s", code)
	w.WriteHeader(http.StatusOK)
}
