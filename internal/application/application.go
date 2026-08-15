package application

import (
	"net/http"

	"example.com/spellingchallenge/internal/challenge"
	"example.com/spellingchallenge/internal/httpapi"
	"example.com/spellingchallenge/internal/review"
)

func NewHandler(snapshotLoaded func()) http.Handler {
	catalog := challenge.NewCatalog()
	results := challenge.NewResultStore()
	challengeService := challenge.NewService(catalog, results)
	reviewRepository := review.NewRepository(review.Record{
		ID:            "daily-001",
		Title:         "今日词库验收",
		Confirmations: map[string]string{},
	})
	reviewService := review.NewService(reviewRepository, snapshotLoaded)
	return httpapi.New(challengeService, reviewService)
}
