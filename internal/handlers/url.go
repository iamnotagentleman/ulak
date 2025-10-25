package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"ulak/internal/models"
	"ulak/internal/service"
)

type Handler struct {
	Service service.Service
}

func NewHandler(service service.Service) *Handler {
	return &Handler{
		Service: service,
	}
}

// SetMessageAutoSend godoc
// @Summary      Set message auto-send configuration
// @Description  Enable or disable automatic message sending
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        request body models.SetMessageAutoSendRequest true "Auto-send configuration"
// @Success      200 {object} models.SetMessageAutoSendResponse
// @Failure      400 {object} models.SetMessageAutoSendResponse
// @Failure      500 {object} models.SetMessageAutoSendResponse
// @Security     ApiKeyAuth
// @Router       /messages/auto-send [post]
func (h *Handler) SetMessageAutoSend(w http.ResponseWriter, req *http.Request) {
	var input models.SetMessageAutoSendRequest

	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	val := h.Service.SetMessageAutoSend(req.Context(), input)
	res, err := json.Marshal(val)

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Determine status code based on response
	statusCode := http.StatusOK
	if val.Result != nil && val.Result.Code != 0 {
		statusCode = val.Result.Code
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(res)
}

// GetSentMessages godoc
// @Summary      Get sent messages
// @Description  Retrieve a list of sent messages
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        limit query int false "Number of messages to return" default(10)
// @Param        offset query int false "Offset for pagination" default(0)
// @Success      200 {object} models.GetMessagesResponse
// @Failure      400 {object} models.GetMessagesResponse
// @Failure      500 {object} models.GetMessagesResponse
// @Security     ApiKeyAuth
// @Router       /messages/sent [get]
func (h *Handler) GetSentMessages(w http.ResponseWriter, req *http.Request) {
	var input models.GetMessagesRequest

	// Parse query parameters
	queryParams := req.URL.Query()

	if limitStr := queryParams.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			input.Limit = limit
		}
	}

	if offsetStr := queryParams.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			input.Offset = offset
		}
	}

	val := h.Service.GetSentMessages(req.Context(), input)
	res, err := json.Marshal(val)

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Determine status code based on response
	statusCode := http.StatusOK
	if val.Result != nil && val.Result.Code != 0 {
		statusCode = val.Result.Code
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(res)
}
