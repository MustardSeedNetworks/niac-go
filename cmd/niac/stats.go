package main

import (
	"os"

	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/stats"
)

// exitWithStats exports statistics if configured and then exits with the given code.
func exitWithStats(code int, flags *legacyFlags, statsTracker *stats.Statistics) {
	if flags != nil && (flags.exportStatsJSON != "" || flags.exportStatsCSV != "") {
		exportStatistics(flags, statsTracker)
	}
	os.Exit(code)
}

// exportStatistics exports runtime statistics to JSON and/or CSV files (v1.19.0).
func exportStatistics(flags *legacyFlags, statsTracker *stats.Statistics) {
	if statsTracker == nil {
		return
	}

	// Update final statistics
	statsTracker.Update()

	// Export to JSON if requested
	if flags.exportStatsJSON != "" {
		if err := statsTracker.ExportJSON(flags.exportStatsJSON); err != nil {
			logging.Errorf("Failed to export statistics to JSON: %v", err)
		} else {
			logging.Infof("Statistics exported to JSON: %s", flags.exportStatsJSON)
		}
	}

	// Export to CSV if requested
	if flags.exportStatsCSV != "" {
		if err := statsTracker.ExportCSV(flags.exportStatsCSV); err != nil {
			logging.Errorf("Failed to export statistics to CSV: %v", err)
		} else {
			logging.Infof("Statistics exported to CSV: %s", flags.exportStatsCSV)
		}
	}
}
