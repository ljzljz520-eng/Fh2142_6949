package application_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"example.com/spellingchallenge/internal/application"
	"example.com/spellingchallenge/internal/challenge"
	"example.com/spellingchallenge/internal/review"
)

func TestChallengeJourneyPersistsResultsMistakesAndStats(t *testing.T) {
	handler := application.NewHandler(nil)

	var simpleWords []challenge.Word
	getJSON(t, handler, "/api/words?level=simple", &simpleWords)
	if len(simpleWords) != 3 || simpleWords[0].ID != "simple-apple" {
		t.Fatalf("simple words = %#v", simpleWords)
	}
	var mediumWords []challenge.Word
	getJSON(t, handler, "/api/words?level=medium", &mediumWords)
	if len(mediumWords) != 3 || mediumWords[2].ID != "medium-journey" {
		t.Fatalf("medium words = %#v", mediumWords)
	}

	var correct challenge.Attempt
	postJSON(t, handler, "/api/attempts", map[string]any{
		"wordId": "simple-apple", "answer": "APPLE", "durationSeconds": 7,
	}, http.StatusCreated, &correct)
	if !correct.Correct || correct.Score != 10 || correct.DurationSeconds != 7 {
		t.Fatalf("correct attempt = %#v", correct)
	}

	var incorrect challenge.Attempt
	postJSON(t, handler, "/api/attempts", map[string]any{
		"wordId": "medium-journey", "answer": "jorney", "durationSeconds": 13,
	}, http.StatusCreated, &incorrect)
	if incorrect.Correct || incorrect.Score != 0 || incorrect.Expected != "journey" {
		t.Fatalf("incorrect attempt = %#v", incorrect)
	}

	var stats challenge.Stats
	getJSON(t, handler, "/api/stats", &stats)
	wantStats := challenge.Stats{
		Attempts: 2, Correct: 1, Incorrect: 1, TotalSeconds: 20,
		TotalScore: 10, AverageSeconds: 10, AccuracyPercent: 50,
	}
	if stats != wantStats {
		t.Fatalf("stats = %#v, want %#v", stats, wantStats)
	}

	var mistakes []challenge.Mistake
	getJSON(t, handler, "/api/mistakes", &mistakes)
	if len(mistakes) != 1 || mistakes[0].Expected != "journey" || mistakes[0].LastAnswer != "jorney" || mistakes[0].Count != 1 {
		t.Fatalf("mistakes = %#v", mistakes)
	}

	var history []challenge.Attempt
	getJSON(t, handler, "/api/attempts", &history)
	if len(history) != 2 || history[0].ID != "attempt-001" || history[1].ID != "attempt-002" {
		t.Fatalf("history = %#v", history)
	}
}

func TestInterfaceAndSequentialReviewJourney(t *testing.T) {
	handler := application.NewHandler(nil)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "单词拼写挑战") {
		t.Fatalf("interface response = %d %q", response.Code, response.Body.String())
	}

	var first review.Record
	postJSON(t, handler, "/api/reviews/daily-001/confirm", map[string]string{
		"operator": "操作员甲", "content": "简单词库已确认",
	}, http.StatusOK, &first)
	var second review.Record
	postJSON(t, handler, "/api/reviews/daily-001/confirm", map[string]string{
		"operator": "操作员乙", "content": "中等词库已确认",
	}, http.StatusOK, &second)
	if len(second.Confirmations) != 2 || second.Confirmations["操作员甲"] != "简单词库已确认" || second.Confirmations["操作员乙"] != "中等词库已确认" {
		t.Fatalf("review = %#v", second.Confirmations)
	}
}

func TestConcurrentConfirmationsRemainInSummary(t *testing.T) {
	reached := make(chan struct{}, 2)
	release := make(chan struct{})
	handler := application.NewHandler(func() {
		reached <- struct{}{}
		<-release
	})

	requests := []map[string]string{
		{"operator": "操作员甲", "content": "简单词库已确认"},
		{"operator": "操作员乙", "content": "中等词库已确认"},
	}
	responses := make([]*httptest.ResponseRecorder, len(requests))
	var wait sync.WaitGroup
	for index, payload := range requests {
		wait.Add(1)
		go func(index int, payload map[string]string) {
			defer wait.Done()
			body, err := json.Marshal(payload)
			if err != nil {
				t.Errorf("marshal request: %v", err)
				return
			}
			request := httptest.NewRequest(http.MethodPost, "/api/reviews/daily-001/confirm", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			responses[index] = response
		}(index, payload)
	}
	<-reached
	<-reached
	close(release)
	wait.Wait()

	for index, response := range responses {
		if response == nil || response.Code != http.StatusOK {
			t.Fatalf("response %d = %#v", index, response)
		}
	}

	var summary review.Record
	getJSON(t, handler, "/api/reviews/daily-001", &summary)
	want := map[string]string{
		"操作员甲": "简单词库已确认",
		"操作员乙": "中等词库已确认",
	}
	if len(summary.Confirmations) != len(want) {
		t.Fatalf("confirmations = %#v, want %#v", summary.Confirmations, want)
	}
	for operator, content := range want {
		if summary.Confirmations[operator] != content {
			t.Fatalf("confirmations = %#v, want %#v", summary.Confirmations, want)
		}
	}
}

func getJSON(t *testing.T, handler http.Handler, path string, destination any) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s = %d %q", path, response.Code, response.Body.String())
	}
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode GET %s: %v", path, err)
	}
}

func postJSON(t *testing.T, handler http.Handler, path string, payload any, status int, destination any) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal POST %s: %v", path, err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("POST %s = %d %q", path, response.Code, response.Body.String())
	}
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode POST %s: %v", path, err)
	}
}
