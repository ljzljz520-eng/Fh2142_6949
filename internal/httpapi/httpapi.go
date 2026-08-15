package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"example.com/spellingchallenge/internal/challenge"
	"example.com/spellingchallenge/internal/review"
	"example.com/spellingchallenge/web/ui"
)

type API struct {
	challenges *challenge.Service
	reviews    *review.Service
}

func New(challenges *challenge.Service, reviews *review.Service) http.Handler {
	api := &API{challenges: challenges, reviews: reviews}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/words", api.words)
	mux.HandleFunc("POST /api/attempts", api.submitAttempt)
	mux.HandleFunc("GET /api/attempts", api.history)
	mux.HandleFunc("GET /api/stats", api.stats)
	mux.HandleFunc("GET /api/mistakes", api.mistakes)
	mux.HandleFunc("GET /api/reviews/{id}", api.reviewRecord)
	mux.HandleFunc("POST /api/reviews/{id}/confirm", api.confirmReview)
	mux.Handle("/", ui.Handler())
	return mux
}

func (a *API) words(w http.ResponseWriter, r *http.Request) {
	words, err := a.challenges.Words(challenge.Level(r.URL.Query().Get("level")))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, words)
}

func (a *API) submitAttempt(w http.ResponseWriter, r *http.Request) {
	var request struct {
		WordID          string `json:"wordId"`
		Answer          string `json:"answer"`
		DurationSeconds int    `json:"durationSeconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	attempt, err := a.challenges.Submit(request.WordID, request.Answer, request.DurationSeconds)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, challenge.ErrWordNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusCreated, attempt)
}

func (a *API) history(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.challenges.History())
}

func (a *API) stats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.challenges.Stats())
}

func (a *API) mistakes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.challenges.Mistakes())
}

func (a *API) reviewRecord(w http.ResponseWriter, r *http.Request) {
	record, err := a.reviews.Record(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (a *API) confirmReview(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Operator string `json:"operator"`
		Content  string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	record, err := a.reviews.Confirm(r.PathValue("id"), request.Operator, request.Content)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, review.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
