package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ilya/eve-sde-server/internal/scheduler"
	"github.com/rs/zerolog/log"
)

type SchedulerHandler struct {
	scheduler *scheduler.Scheduler
}

func NewSchedulerHandler(s *scheduler.Scheduler) *SchedulerHandler {
	return &SchedulerHandler{scheduler: s}
}

// TriggerUpdate manually triggers an SDE update
func (h *SchedulerHandler) TriggerUpdate(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Manual SDE update triggered via API")

	// Trigger update in background
	go func() {
		if err := h.scheduler.TriggerUpdate(); err != nil {
			log.Error().Err(err).Msg("SDE update failed")
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "started",
		"message": "SDE update has been triggered in the background",
	})
}

// GetStatus returns the scheduler status
func (h *SchedulerHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	lastCheck := h.scheduler.GetLastCheck()

	status := map[string]interface{}{
		"last_check": lastCheck,
	}

	if lastCheck.IsZero() {
		status["last_check"] = nil
		status["message"] = "No updates have been performed yet"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
