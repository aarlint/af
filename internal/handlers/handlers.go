package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/aarlint/af/internal/db"
	"github.com/aarlint/af/internal/models"
)

type Handler struct {
	DB *sql.DB
}

func (h *Handler) Actions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := db.ActionFilter{
		President: q.Get("president"),
		Category:  q.Get("category"),
		Country:   q.Get("country"),
		Impact:    q.Get("impact"),
		Search:    q.Get("search"),
		Sort:      q.Get("sort"),
	}

	actions, total, err := db.GetActions(h.DB, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := models.ActionsResponse{
		Actions:  actions,
		Total:    total,
		Filtered: len(actions),
	}
	if resp.Actions == nil {
		resp.Actions = []models.Action{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Scores(w http.ResponseWriter, r *http.Request) {
	president := r.URL.Query().Get("president")

	scores, err := db.GetScores(h.DB, president)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores)
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
