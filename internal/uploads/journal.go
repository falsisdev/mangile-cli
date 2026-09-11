package uploads

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Journal struct {
	ID          string         `json:"id"`
	CreatedAt   time.Time      `json:"createdAt"`
	SeriesID    string         `json:"seriesId"`
	SeriesType  string         `json:"seriesType"`
	SeriesRev   string         `json:"seriesRev"`
	CreatedDocs []string       `json:"createdDocs"`
	PatchedDocs []JournalPatch `json:"patchedDocs"`
	Assets      []string       `json:"assets"`
}

type JournalPatch struct {
	ID     string         `json:"id"`
	Rev    string         `json:"rev"`
	Before map[string]any `json:"before,omitempty"`
}

func NewJournal(seriesID, seriesType string) *Journal {
	return &Journal{
		ID:          time.Now().Format("20060102-150405"),
		CreatedAt:   time.Now(),
		SeriesID:    seriesID,
		SeriesType:  seriesType,
		CreatedDocs: []string{},
		PatchedDocs: []JournalPatch{},
		Assets:      []string{},
	}
}

func StateDir(uploadsDir string) string {
	return filepath.Join(uploadsDir, ".state")
}

func JournalPath(uploadsDir, id string) string {
	return filepath.Join(StateDir(uploadsDir), "journal-"+id+".json")
}

func (j *Journal) Save(uploadsDir string) error {
	dir := StateDir(uploadsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(JournalPath(uploadsDir, j.ID), data, 0o600)
}

func LoadJournal(uploadsDir, id string) (*Journal, error) {
	data, err := os.ReadFile(JournalPath(uploadsDir, id))
	if err != nil {
		return nil, err
	}
	var j Journal
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}
	return &j, nil
}

func ListJournals(uploadsDir string) ([]*Journal, error) {
	dir := StateDir(uploadsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var journals []*Journal
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "journal-") || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(strings.TrimPrefix(e.Name(), "journal-"), ".json")
		j, err := LoadJournal(uploadsDir, id)
		if err != nil {
			continue
		}
		journals = append(journals, j)
	}
	sort.Slice(journals, func(i, j int) bool { return journals[i].CreatedAt.After(journals[j].CreatedAt) })
	return journals, nil
}

func DeleteJournal(uploadsDir, id string) error {
	return os.Remove(JournalPath(uploadsDir, id))
}
