package parser

import (
	"strconv"
	"strings"
	"regexp"
)

// Utility functions for common parsing tasks

// IsSection checks if a line is a section boundary (starts with ***)
func IsSection(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "***")
}

// GetSectionName extracts the section name from a section boundary line
func GetSectionName(line string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "***") {
		return ""
	}
	// Remove *** from both ends and trim
	content := strings.TrimSpace(trimmed)
	content = strings.TrimPrefix(content, "***")
	content = strings.TrimSuffix(content, "***")
	return strings.TrimSpace(content)
}

// IsNewSample checks if a line indicates a new sample
func IsNewSample(line string) bool {
	return strings.Contains(line, "*** Sampled system activity")
}

// IsRunningTasks checks if a line indicates the running tasks section
func IsRunningTasks(line string) bool {
	return strings.Contains(line, "*** Running tasks ***")
}

// IsEndOfTasks checks if a line indicates the end of the tasks section
func IsEndOfTasks(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "ALL_TASKS")
}

// IsTasksHeader checks if a line is the tasks section header
func IsTasksHeader(line string) bool {
	return strings.Contains(line, "Name") && strings.Contains(line, "ID")
}

// IsIndented checks if a line is indented (subprocess vs coalition)
func IsIndented(line string) bool {
	return strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
}

// ParseFloat safely parses a float from a string
func ParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// ParseInt safely parses an int from a string
func ParseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// ConvertToMB converts various units to MB
func ConvertToMB(value float64, line string) float64 {
	if strings.Contains(line, "KB") {
		return value / 1024
	} else if strings.Contains(line, "GB") {
		return value * 1024
	} else if strings.Contains(line, "bytes") {
		return value / (1024 * 1024)
	}
	return value
}

// IsPCluster checks if a line indicates a P-cluster (handles P, P0, P1, P2, ... P9+)
func IsPCluster(line string) bool {
	// Use regex to match P-Cluster, P0-Cluster, P1-Cluster, etc.
	pClusterRegex := regexp.MustCompile(`P\d*-Cluster Online:`)
	return pClusterRegex.MatchString(line)
}

// IsECluster checks if a line indicates an E-cluster
func IsECluster(line string) bool {
	return strings.Contains(line, "E-Cluster Online:")
}