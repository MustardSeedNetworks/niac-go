package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/storage"
)

func historyServer(t *testing.T) *Server {
	t.Helper()
	s, dir := createTestServer(t)
	store, err := storage.Open(filepath.Join(dir, "history.db"), storage.RetainAll)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s.cfg.Storage = store
	for i := range 40 {
		if addErr := store.AddRun(storage.RunRecord{ConfigName: fmt.Sprintf("run-%02d", i+1)}); addErr != nil {
			t.Fatal(addErr)
		}
	}
	return s
}

func historyPage(t *testing.T, s *Server, query string) []storage.RunRecord {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleHistory(w, httptest.NewRequest("GET", "/api/v1/history"+query, nil))
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var records []storage.RunRecord
	if err := json.Unmarshal(w.Body.Bytes(), &records); err != nil {
		t.Fatal(err)
	}
	return records
}

func TestHistoryPagination(t *testing.T) {
	s := historyServer(t)
	first := historyPage(t, s, "")
	if len(first) != 20 || first[0].ID != 40 || first[19].ID != 21 {
		t.Fatalf("first page: %+v", first)
	}
	if err := s.cfg.Storage.AddRun(storage.RunRecord{ConfigName: "new run"}); err != nil {
		t.Fatal(err)
	}
	second := historyPage(t, s, "?before=21")
	if len(second) != 20 || second[0].ID != 20 || second[19].ID != 1 {
		t.Fatalf("second page: %+v", second)
	}
	if last := historyPage(t, s, "?before=1"); len(last) != 0 {
		t.Fatalf("last page: %+v", last)
	}
	if newest := historyPage(t, s, ""); newest[0].ID != 41 {
		t.Fatalf("newest: %+v", newest)
	}
}

func TestHistoryRejectsInvalidCursor(t *testing.T) {
	s := historyServer(t)
	for _, query := range []string{"?before=oops", "?before=-1", "?before=0", "?before=18446744073709551616"} {
		t.Run(query, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.handleHistory(w, httptest.NewRequest("GET", "/api/v1/history"+query, nil))
			if w.Code != 400 {
				t.Fatalf("status = %d, want 400", w.Code)
			}
		})
	}
}
