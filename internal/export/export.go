package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileInfo represents a file with its path and size
type FileInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// JSONExportData represents the complete export data in JSON format
type JSONExportData struct {
	GeneratedAt string                           `json:"generated_at"`
	ExportMode  string                           `json:"export_mode"`
	Libraries   map[string]map[string][]FileInfo `json:"libraries"`
	Summary     JSONSummary                      `json:"summary"`
}

// JSONSummary represents summary statistics in JSON format
type JSONSummary struct {
	TotalFiles         int                         `json:"total_files"`
	TotalSize          int64                       `json:"total_size"`
	TotalSizeFormatted string                      `json:"total_size_formatted"`
	LibraryStats       map[string]JSONLibraryStats `json:"library_stats"`
	LabelTotals        map[string]JSONLabelStats   `json:"label_totals"`
}

// JSONLibraryStats represents per-library statistics
type JSONLibraryStats struct {
	TotalFiles         int                       `json:"total_files"`
	TotalSize          int64                     `json:"total_size"`
	TotalSizeFormatted string                    `json:"total_size_formatted"`
	Labels             map[string]JSONLabelStats `json:"labels"`
}

// JSONLabelStats represents per-label statistics
type JSONLabelStats struct {
	Count         int    `json:"count"`
	Size          int64  `json:"size"`
	SizeFormatted string `json:"size_formatted"`
}

// Exporter handles exporting file paths based on labels
type Exporter struct {
	exportLocation string
	exportLabels   []string
	exportMode     string
	currentLibrary string                           // Current library being processed
	accumulated    map[string]map[string][]FileInfo // library -> label -> list of file info
	mutex          sync.Mutex
}

// NewExporter creates a new Exporter instance
func NewExporter(exportLocation string, exportLabels []string, exportMode string) (*Exporter, error) {
	if exportLocation == "" {
		return nil, fmt.Errorf("export location cannot be empty")
	}

	if len(exportLabels) == 0 {
		return nil, fmt.Errorf("export labels cannot be empty")
	}

	if exportMode != "txt" && exportMode != "json" {
		return nil, fmt.Errorf("export mode must be 'txt' or 'json'")
	}

	// Create the export directory if it doesn't exist
	if err := os.MkdirAll(exportLocation, 0755); err != nil {
		return nil, fmt.Errorf("failed to create export directory: %w", err)
	}

	return &Exporter{
		exportLocation: exportLocation,
		exportLabels:   exportLabels,
		exportMode:     exportMode,
		accumulated:    make(map[string]map[string][]FileInfo),
	}, nil
}

// SetCurrentLibrary sets the current library being processed
func (e *Exporter) SetCurrentLibrary(libraryName string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if libraryName == "" {
		return fmt.Errorf("library name cannot be empty")
	}

	// Sanitize library name for use as directory name
	sanitizedName := sanitizeFilename(libraryName)
	e.currentLibrary = sanitizedName

	// Create library-specific subdirectory if it doesn't exist
	libraryPath := filepath.Join(e.exportLocation, sanitizedName)
	if err := os.MkdirAll(libraryPath, 0755); err != nil {
		return fmt.Errorf("failed to create library directory %s: %w", libraryPath, err)
	}

	// Initialize accumulated map for this library if it doesn't exist
	if e.accumulated[sanitizedName] == nil {
		e.accumulated[sanitizedName] = make(map[string][]FileInfo)
	}

	return nil
}

// ExportItemWithSizes checks if an item has any of the export labels and accumulates its file info
func (e *Exporter) ExportItemWithSizes(title string, itemLabels []string, fileInfos []FileInfo) error {
	if len(fileInfos) == 0 {
		return nil // Nothing to export
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.currentLibrary == "" {
		return fmt.Errorf("no current library set - call SetCurrentLibrary first")
	}

	// Convert item labels to lowercase for case-insensitive comparison
	itemLabelsMap := make(map[string]bool)
	for _, label := range itemLabels {
		itemLabelsMap[strings.ToLower(strings.TrimSpace(label))] = true
	}

	// Check which export labels this item has
	var matchingLabels []string
	for _, exportLabel := range e.exportLabels {
		if itemLabelsMap[strings.ToLower(strings.TrimSpace(exportLabel))] {
			matchingLabels = append(matchingLabels, exportLabel)
		}
	}

	if len(matchingLabels) == 0 {
		return nil // Item doesn't have any of the export labels
	}

	// Ensure library exists in accumulated map
	if e.accumulated[e.currentLibrary] == nil {
		e.accumulated[e.currentLibrary] = make(map[string][]FileInfo)
	}

	// Accumulate file info for all matching labels
	for _, label := range matchingLabels {
		if e.accumulated[e.currentLibrary][label] == nil {
			e.accumulated[e.currentLibrary][label] = make([]FileInfo, 0)
		}
		e.accumulated[e.currentLibrary][label] = append(e.accumulated[e.currentLibrary][label], fileInfos...)
	}

	return nil
}

// ExportItem checks if an item has any of the export labels and accumulates its file paths (backwards compatibility)
func (e *Exporter) ExportItem(title string, itemLabels []string, filePaths []string) error {
	if len(filePaths) == 0 {
		return nil // Nothing to export
	}

	// Convert paths to FileInfo with zero size for backwards compatibility
	fileInfos := make([]FileInfo, len(filePaths))
	for i, path := range filePaths {
		fileInfos[i] = FileInfo{Path: path, Size: 0}
	}

	return e.ExportItemWithSizes(title, itemLabels, fileInfos)
}

// FlushAll writes all accumulated file paths to their respective files based on export mode
// This method overwrites any existing export files with the new accumulated data
func (e *Exporter) FlushAll() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	switch e.exportMode {
	case "txt":
		return e.flushTxt()
	case "json":
		return e.flushJSON()
	default:
		return fmt.Errorf("unsupported export mode: %s", e.exportMode)
	}
}

// flushTxt writes all accumulated file paths to library-specific txt files
func (e *Exporter) flushTxt() error {
	// Process each library
	for libraryName, libraryData := range e.accumulated {
		libraryPath := filepath.Join(e.exportLocation, libraryName)

		// Ensure library directory exists
		if err := os.MkdirAll(libraryPath, 0755); err != nil {
			return fmt.Errorf("failed to create library directory %s: %w", libraryPath, err)
		}

		// Write files for each export label
		for _, label := range e.exportLabels {
			filename := fmt.Sprintf("%s.txt", sanitizeFilename(label))
			filePath := filepath.Join(libraryPath, filename)

			// Get accumulated file info for this label in this library
			fileInfos := libraryData[label]
			if len(fileInfos) == 0 {
				// Create empty file for labels with no matches
				file, err := os.Create(filePath)
				if err != nil {
					return fmt.Errorf("failed to create export file %s: %w", filePath, err)
				}
				file.Close()
				continue
			}

			// Create/overwrite file and write all paths at once
			file, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create export file %s: %w", filePath, err)
			}

			for _, fileInfo := range fileInfos {
				if _, err := fmt.Fprintf(file, "%s\n", fileInfo.Path); err != nil {
					file.Close()
					return fmt.Errorf("failed to write to export file %s: %w", filePath, err)
				}
			}

			file.Close()
		}
	}

	// Write summary file
	if err := e.writeSummary(); err != nil {
		return fmt.Errorf("failed to write summary file: %w", err)
	}

	// Clear accumulated data after successful write
	e.accumulated = make(map[string]map[string][]FileInfo)

	return nil
}

// flushJSON writes all accumulated data as a single JSON file
func (e *Exporter) flushJSON() error {
	jsonData := e.buildJSONExportData()

	// Write JSON file
	jsonPath := filepath.Join(e.exportLocation, "export.json")
	file, err := os.Create(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to create JSON export file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(jsonData); err != nil {
		return fmt.Errorf("failed to write JSON export file: %w", err)
	}

	// Clear accumulated data after successful write
	e.accumulated = make(map[string]map[string][]FileInfo)

	return nil
}

// writeSummary writes a summary.txt file with detailed statistics
func (e *Exporter) writeSummary() error {
	summaryPath := filepath.Join(e.exportLocation, "summary.txt")

	file, err := os.Create(summaryPath)
	if err != nil {
		return fmt.Errorf("failed to create summary file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "Labelarr Export Summary\n")
	fmt.Fprintf(file, "Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// Calculate totals
	totalFiles := 0
	totalSize := int64(0)
	libraryStats := make(map[string]map[string]struct {
		Count int
		Size  int64
	})

	// Collect statistics
	for libraryName, libraryData := range e.accumulated {
		libraryStats[libraryName] = make(map[string]struct {
			Count int
			Size  int64
		})

		for label, fileInfos := range libraryData {
			count := len(fileInfos)
			size := int64(0)
			for _, fi := range fileInfos {
				size += fi.Size
			}

			libraryStats[libraryName][label] = struct {
				Count int
				Size  int64
			}{Count: count, Size: size}

			totalFiles += count
			totalSize += size
		}
	}

	// Write export file list
	fmt.Fprintf(file, "📁 Export Files Generated:\n")
	for libraryName := range libraryStats {
		for _, label := range e.exportLabels {
			if stats, exists := libraryStats[libraryName][label]; exists && stats.Count > 0 {
				fmt.Fprintf(file, "  %s/%s.txt\n", libraryName, sanitizeFilename(label))
			}
		}
	}
	fmt.Fprintf(file, "\n")

	// Write totals
	fmt.Fprintf(file, "📊 Overall Statistics:\n")
	fmt.Fprintf(file, "  Total files: %d\n", totalFiles)
	fmt.Fprintf(file, "  Total size: %s (%d bytes)\n", formatFileSize(totalSize), totalSize)
	fmt.Fprintf(file, "\n")

	// Write per-library breakdown
	fmt.Fprintf(file, "📚 Library Breakdown:\n")
	for libraryName, labelStats := range libraryStats {
		fmt.Fprintf(file, "\n  %s:\n", libraryName)

		libraryTotal := 0
		librarySizeTotal := int64(0)

		for _, label := range e.exportLabels {
			if stats, exists := labelStats[label]; exists {
				if stats.Count > 0 {
					fmt.Fprintf(file, "    %s.txt: %d files, %s (%d bytes)\n",
						sanitizeFilename(label), stats.Count, formatFileSize(stats.Size), stats.Size)
				} else {
					fmt.Fprintf(file, "    %s.txt: 0 files (empty)\n", sanitizeFilename(label))
				}
				libraryTotal += stats.Count
				librarySizeTotal += stats.Size
			} else {
				fmt.Fprintf(file, "    %s.txt: 0 files (empty)\n", sanitizeFilename(label))
			}
		}

		fmt.Fprintf(file, "    Library total: %d files, %s (%d bytes)\n",
			libraryTotal, formatFileSize(librarySizeTotal), librarySizeTotal)
	}

	// Write per-label totals across all libraries
	fmt.Fprintf(file, "\n🏷️ Label Totals (All Libraries):\n")
	labelTotals := make(map[string]struct {
		Count int
		Size  int64
	})

	for _, libraryData := range e.accumulated {
		for label, fileInfos := range libraryData {
			existing := labelTotals[label]
			existing.Count += len(fileInfos)
			for _, fi := range fileInfos {
				existing.Size += fi.Size
			}
			labelTotals[label] = existing
		}
	}

	for _, label := range e.exportLabels {
		if stats, exists := labelTotals[label]; exists && stats.Count > 0 {
			fmt.Fprintf(file, "  %s: %d files, %s (%d bytes)\n",
				label, stats.Count, formatFileSize(stats.Size), stats.Size)
		} else {
			fmt.Fprintf(file, "  %s: 0 files\n", label)
		}
	}

	return nil
}

// formatFileSize converts bytes to human-readable format
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ClearExportFiles removes all existing export files and clears accumulated data
// This method is primarily for manual cleanup or testing purposes
func (e *Exporter) ClearExportFiles() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Remove all library subdirectories and their contents
	for libraryName := range e.accumulated {
		libraryPath := filepath.Join(e.exportLocation, libraryName)
		if err := os.RemoveAll(libraryPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove library directory %s: %w", libraryPath, err)
		}
	}

	// Remove summary file
	summaryPath := filepath.Join(e.exportLocation, "summary.txt")
	if err := os.Remove(summaryPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove summary file: %w", err)
	}

	// Clear accumulated data
	e.accumulated = make(map[string]map[string][]FileInfo)

	return nil
}

// GetExportSummary returns a summary of accumulated file counts (before flushing)
func (e *Exporter) GetExportSummary() (map[string]int, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	summary := make(map[string]int)

	// Aggregate totals across all libraries for each label
	for _, libraryData := range e.accumulated {
		for _, label := range e.exportLabels {
			fileInfos := libraryData[label]
			summary[label] += len(fileInfos)
		}
	}

	return summary, nil
}

// GetLibraryExportSummary returns a detailed summary showing counts per library and label
func (e *Exporter) GetLibraryExportSummary() (map[string]map[string]int, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	summary := make(map[string]map[string]int)

	// Process each library
	for libraryName, libraryData := range e.accumulated {
		summary[libraryName] = make(map[string]int)
		for _, label := range e.exportLabels {
			fileInfos := libraryData[label]
			summary[libraryName][label] = len(fileInfos)
		}
	}

	return summary, nil
}

// GetAccumulatedCount returns the total number of accumulated files across all labels and libraries
func (e *Exporter) GetAccumulatedCount() int {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	total := 0
	for _, libraryData := range e.accumulated {
		for _, fileInfos := range libraryData {
			total += len(fileInfos)
		}
	}
	return total
}

// GetCurrentLibrary returns the name of the currently set library
func (e *Exporter) GetCurrentLibrary() string {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.currentLibrary
}

// sanitizeFilename removes invalid characters from filenames
func sanitizeFilename(filename string) string {
	// Replace common invalid characters with underscores
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return strings.TrimSpace(result)
}

// buildJSONExportData builds a JSONExportData struct from the accumulated data
func (e *Exporter) buildJSONExportData() JSONExportData {
	jsonData := JSONExportData{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		ExportMode:  e.exportMode,
		Libraries:   e.accumulated,
		Summary:     e.buildJSONSummary(),
	}
	return jsonData
}

// buildJSONSummary builds a JSONSummary struct from the accumulated data
func (e *Exporter) buildJSONSummary() JSONSummary {
	totalFiles := 0
	totalSize := int64(0)
	libraryStats := make(map[string]map[string]struct {
		Count int
		Size  int64
	})

	// Collect statistics
	for libraryName, libraryData := range e.accumulated {
		libraryStats[libraryName] = make(map[string]struct {
			Count int
			Size  int64
		})

		for label, fileInfos := range libraryData {
			count := len(fileInfos)
			size := int64(0)
			for _, fi := range fileInfos {
				size += fi.Size
			}

			libraryStats[libraryName][label] = struct {
				Count int
				Size  int64
			}{Count: count, Size: size}

			totalFiles += count
			totalSize += size
		}
	}

	labelTotals := make(map[string]struct {
		Count int
		Size  int64
	})

	for _, libraryData := range e.accumulated {
		for label, fileInfos := range libraryData {
			existing := labelTotals[label]
			existing.Count += len(fileInfos)
			for _, fi := range fileInfos {
				existing.Size += fi.Size
			}
			labelTotals[label] = existing
		}
	}

	// Convert to JSON struct types
	jsonLibraryStats := make(map[string]JSONLibraryStats)
	for libraryName, labelStats := range libraryStats {
		libraryTotal := 0
		librarySizeTotal := int64(0)
		jsonLabels := make(map[string]JSONLabelStats)

		for label, stats := range labelStats {
			jsonLabels[label] = JSONLabelStats{
				Count:         stats.Count,
				Size:          stats.Size,
				SizeFormatted: formatFileSize(stats.Size),
			}
			libraryTotal += stats.Count
			librarySizeTotal += stats.Size
		}

		jsonLibraryStats[libraryName] = JSONLibraryStats{
			TotalFiles:         libraryTotal,
			TotalSize:          librarySizeTotal,
			TotalSizeFormatted: formatFileSize(librarySizeTotal),
			Labels:             jsonLabels,
		}
	}

	jsonLabelTotals := make(map[string]JSONLabelStats)
	for label, stats := range labelTotals {
		jsonLabelTotals[label] = JSONLabelStats{
			Count:         stats.Count,
			Size:          stats.Size,
			SizeFormatted: formatFileSize(stats.Size),
		}
	}

	summary := JSONSummary{
		TotalFiles:         totalFiles,
		TotalSize:          totalSize,
		TotalSizeFormatted: formatFileSize(totalSize),
		LibraryStats:       jsonLibraryStats,
		LabelTotals:        jsonLabelTotals,
	}
	return summary
}
