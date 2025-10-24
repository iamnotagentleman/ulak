package handlers

import (
	"encoding/json"
	"net/http"
	"ulak/internal/models"
	"ulak/internal/service"

	"github.com/ggicci/httpin"
)

type Handler struct {
	Service service.Service
}

func NewHandler(service service.Service) *Handler {
	return &Handler{
		Service: service,
	}
}

func WithHTTPIn[T any](handler http.HandlerFunc, input T) http.Handler {
	return httpin.NewInput(input)(handler)
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
	input := req.Context().Value(httpin.Input).(*models.SetMessageAutoSendRequest)
	val := h.Service.SetMessageAutoSend(req.Context(), *input)
	res, err := json.Marshal(val)

	if err != nil {
		internalError := http.StatusInternalServerError
		http.Error(w, err.Error(), internalError)
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
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
// @Param        isDelivered query bool false "Filter by delivery status" default(false)
// @Success      200 {object} models.GetMessagesResponse
// @Failure      400 {object} models.GetMessagesResponse
// @Failure      500 {object} models.GetMessagesResponse
// @Security     ApiKeyAuth
// @Router       /messages/sent [get]
func (h *Handler) GetSentMessages(w http.ResponseWriter, req *http.Request) {
	input := req.Context().Value(httpin.Input).(*models.GetMessagesRequest)
	val := h.Service.GetSentMessages(req.Context(), *input)
	res, err := json.Marshal(val)

	if err != nil {
		internalError := http.StatusInternalServerError
		http.Error(w, err.Error(), internalError)
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(res)

}
