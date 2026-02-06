package storage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
)

// PageStorage wraps the page annotation data folder.
type PageStorage struct {
	Folder string
}

// NewPageStorage creates a PageStorage for the given data folder.
func NewPageStorage(folder string) *PageStorage {
	return &PageStorage{Folder: folder}
}

// pageConfigJSON is the structure of the page config.json.
type pageConfigJSON struct {
	PageTypes typeConfig `json:"page_types"`
}

// pageIndexEntry represents a single entry in the page index.json.
type pageIndexEntry struct {
	URL      string `json:"url"`
	PageType string `json:"page_type"`
}

// PageAnnotation represents a single annotated page.
type PageAnnotation struct {
	HTML     string
	URL      string
	Type     string // short page type
	TypeFull string // full page type
}

// GetPageSchema reads the page type schema from config.json.
func (s *PageStorage) GetPageSchema() (*AnnotationSchema, error) {
	data, err := os.ReadFile(filepath.Join(s.Folder, "config.json"))
	if err != nil {
		return nil, err
	}
	var config pageConfigJSON
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return buildSchema(config.PageTypes), nil
}

// GetPageIndex reads the page index file.
func (s *PageStorage) GetPageIndex() (map[string]pageIndexEntry, error) {
	data, err := os.ReadFile(filepath.Join(s.Folder, "index.json"))
	if err != nil {
		return nil, err
	}
	var index map[string]pageIndexEntry
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}
	return index, nil
}

// IterPageAnnotations yields PageAnnotation objects from the storage.
func (s *PageStorage) IterPageAnnotations(opts IterOptions) ([]PageAnnotation, error) {
	schema, err := s.GetPageSchema()
	if err != nil {
		return nil, fmt.Errorf("get page schema: %w", err)
	}
	index, err := s.GetPageIndex()
	if err != nil {
		return nil, fmt.Errorf("get page index: %w", err)
	}

	// Sort by domain + path for deterministic ordering
	type pathInfo struct {
		path string
		info pageIndexEntry
	}
	sorted := make([]pathInfo, 0, len(index))
	for path, info := range index {
		sorted = append(sorted, pathInfo{path, info})
	}
	sort.Slice(sorted, func(i, j int) bool {
		di := GetDomain(sorted[i].info.URL)
		dj := GetDomain(sorted[j].info.URL)
		if di != dj {
			return di < dj
		}
		return sorted[i].path < sorted[j].path
	})

	var annotations []PageAnnotation
	for _, pi := range sorted {
		tp := pi.info.PageType

		if opts.DropNA && tp == schema.NAValue {
			continue
		}
		if opts.DropSkipped && tp == schema.SkipValue {
			continue
		}

		htmlPath := filepath.Join(s.Folder, pi.path)
		htmlData, err := os.ReadFile(htmlPath)
		if err != nil {
			slog.Warn("Cannot read page annotation file", "path", pi.path, "error", err)
			continue
		}

		typeFull := tp
		if full, ok := schema.TypesInv[tp]; ok {
			typeFull = full
		}

		ann := PageAnnotation{
			HTML:     string(htmlData),
			URL:      pi.info.URL,
			Type:     tp,
			TypeFull: typeFull,
		}
		annotations = append(annotations, ann)
	}

	return annotations, nil
}
