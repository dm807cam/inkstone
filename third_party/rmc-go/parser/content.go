package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ContentPage represents a page entry in the .content file
type ContentPage struct {
	ID      string `json:"id"`
	Idx     struct {
		Timestamp string `json:"timestamp"`
		Value     string `json:"value"`
	} `json:"idx"`
	Modified string `json:"modifed"` // Note: typo in reMarkable format
}

// ContentPages represents the cPages section of a .content file
type ContentPages struct {
	LastOpened struct {
		Timestamp string `json:"timestamp"`
		Value     string `json:"value"`
	} `json:"lastOpened"`
	Original struct {
		Timestamp string `json:"timestamp"`
		Value     int    `json:"value"`
	} `json:"original"`
	Pages []ContentPage `json:"pages"`
}

// ContentFile represents a reMarkable .content file
type ContentFile struct {
	CPages    ContentPages `json:"cPages"`
	PageCount int          `json:"pageCount"`
	FileType  string       `json:"fileType"`
}

// ReadContentFile reads and parses a reMarkable .content file
func ReadContentFile(path string) (*ContentFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read content file: %w", err)
	}

	var content ContentFile
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("failed to parse content file: %w", err)
	}

	return &content, nil
}

// GetPageIDs returns the page IDs in the correct order from the content file
func (c *ContentFile) GetPageIDs() []string {
	ids := make([]string, 0, len(c.CPages.Pages))
	for _, page := range c.CPages.Pages {
		ids = append(ids, page.ID)
	}
	return ids
}

// OrderFilesByContent orders .rm files according to a .content file
// Returns the ordered files and a boolean indicating if the content file was used
func OrderFilesByContent(files []string, contentPath string) ([]string, bool) {
	// Try to read the content file
	content, err := ReadContentFile(contentPath)
	if err != nil {
		return files, false
	}

	// Get page IDs in order
	pageIDs := content.GetPageIDs()
	if len(pageIDs) == 0 {
		return files, false
	}

	// Create a map of page ID to file path
	fileMap := make(map[string]string)
	for _, file := range files {
		// Extract the base name without extension
		baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		fileMap[baseName] = file
	}

	// Build ordered list based on content file
	orderedFiles := make([]string, 0, len(pageIDs))
	matchedCount := 0

	for _, pageID := range pageIDs {
		if file, ok := fileMap[pageID]; ok {
			orderedFiles = append(orderedFiles, file)
			matchedCount++
		}
	}

	// If we didn't match any files, return original list
	if matchedCount == 0 {
		return files, false
	}

	// If we matched some but not all files, add unmatched files at the end
	// sorted by modification time
	if matchedCount < len(files) {
		unmatchedFiles := make([]string, 0)
		matchedSet := make(map[string]bool)
		for _, f := range orderedFiles {
			matchedSet[f] = true
		}

		for _, file := range files {
			if !matchedSet[file] {
				unmatchedFiles = append(unmatchedFiles, file)
			}
		}

		// Sort unmatched by modification time
		sort.Slice(unmatchedFiles, func(i, j int) bool {
			infoI, _ := os.Stat(unmatchedFiles[i])
			infoJ, _ := os.Stat(unmatchedFiles[j])
			return infoI.ModTime().Before(infoJ.ModTime())
		})

		orderedFiles = append(orderedFiles, unmatchedFiles...)
	}

	return orderedFiles, true
}
