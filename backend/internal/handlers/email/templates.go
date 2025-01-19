package email

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/email"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	emailtypes "github.com/ZerkerEOD/krakenhashes/backend/pkg/email"
	"github.com/gorilla/mux"
)

// ListTemplates handles GET /api/email/templates
func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	debug.Info("[EmailTemplates] Listing templates")

	// Optional template type filter
	var templateType *emailtypes.TemplateType
	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		tt := emailtypes.TemplateType(typeStr)
		templateType = &tt
		debug.Info("[EmailTemplates] Filtering by type: %s", typeStr)
	}

	templates, err := h.emailService.ListTemplates(ctx, templateType)
	if err != nil {
		debug.Error("[EmailTemplates] Failed to list templates: %v", err)
		http.Error(w, "Failed to list templates", http.StatusInternalServerError)
		return
	}

	debug.Info("[EmailTemplates] Successfully retrieved %d templates", len(templates))
	debug.Debug("[EmailTemplates] Templates: %+v", templates)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(templates); err != nil {
		debug.Error("[EmailTemplates] Failed to encode templates response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetTemplate handles GET /api/email/templates/{id}
func (h *Handler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		debug.Error("[EmailTemplates] Invalid template ID format: %s", vars["id"])
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	debug.Info("[EmailTemplates] Getting template with ID: %d", id)
	template, err := h.emailService.GetTemplate(ctx, id)
	if err == email.ErrTemplateNotFound {
		debug.Info("[EmailTemplates] Template not found: %d", id)
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}
	if err != nil {
		debug.Error("[EmailTemplates] Failed to get template: %v", err)
		http.Error(w, "Failed to get template", http.StatusInternalServerError)
		return
	}

	debug.Debug("[EmailTemplates] Retrieved template: %+v", template)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(template); err != nil {
		debug.Error("[EmailTemplates] Failed to encode template response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// CreateTemplate handles POST /api/email/templates
func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	debug.Info("[EmailTemplates] Creating new template")

	var template emailtypes.Template
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		debug.Error("[EmailTemplates] Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	debug.Debug("[EmailTemplates] Template request data: %+v", template)

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		debug.Error("[EmailTemplates] User ID not found in context")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	debug.Info("[EmailTemplates] Creating template for user: %s", userID)

	if err := h.emailService.CreateTemplate(ctx, &template, userID); err != nil {
		switch err {
		case email.ErrTemplateValidation:
			debug.Error("[EmailTemplates] Template validation failed: %v", err)
			http.Error(w, "Invalid template data", http.StatusBadRequest)
		default:
			debug.Error("[EmailTemplates] Failed to create template: %v", err)
			http.Error(w, "Failed to create template", http.StatusInternalServerError)
		}
		return
	}

	debug.Info("[EmailTemplates] Template created successfully")
	w.WriteHeader(http.StatusCreated)
}

// UpdateTemplate handles PUT /api/email/templates/{id}
func (h *Handler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		debug.Error("[EmailTemplates] Invalid template ID format: %s", vars["id"])
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	debug.Info("[EmailTemplates] Updating template with ID: %d", id)

	var template emailtypes.Template
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		debug.Error("[EmailTemplates] Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	template.ID = id

	debug.Debug("[EmailTemplates] Template update data: %+v", template)

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		debug.Error("[EmailTemplates] User ID not found in context")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	debug.Info("[EmailTemplates] Updating template for user: %s", userID)

	if err := h.emailService.UpdateTemplate(ctx, &template, userID); err != nil {
		switch err {
		case email.ErrTemplateNotFound:
			debug.Info("[EmailTemplates] Template not found: %d", id)
			http.Error(w, "Template not found", http.StatusNotFound)
		case email.ErrTemplateValidation:
			debug.Error("[EmailTemplates] Template validation failed: %v", err)
			http.Error(w, "Invalid template data", http.StatusBadRequest)
		default:
			debug.Error("[EmailTemplates] Failed to update template: %v", err)
			http.Error(w, "Failed to update template", http.StatusInternalServerError)
		}
		return
	}

	debug.Info("[EmailTemplates] Template updated successfully")
	w.WriteHeader(http.StatusOK)
}

// DeleteTemplate handles DELETE /api/email/templates/{id}
func (h *Handler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		debug.Error("[EmailTemplates] Invalid template ID format: %s", vars["id"])
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	debug.Info("[EmailTemplates] Deleting template with ID: %d", id)

	if err := h.emailService.DeleteTemplate(ctx, id); err != nil {
		if err == email.ErrTemplateNotFound {
			debug.Info("[EmailTemplates] Template not found: %d", id)
			http.Error(w, "Template not found", http.StatusNotFound)
			return
		}
		debug.Error("[EmailTemplates] Failed to delete template: %v", err)
		http.Error(w, "Failed to delete template", http.StatusInternalServerError)
		return
	}

	debug.Info("[EmailTemplates] Template deleted successfully")
	w.WriteHeader(http.StatusOK)
}
