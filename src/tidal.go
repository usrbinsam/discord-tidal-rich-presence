package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os/exec"
)

// WindowTitle scans the current process list and returns the window title for the matching process executable name
// An empty string is returned if the process cannot be found
func WindowTitle(imageName string) (string, error) {

	// shell arguments, full command looks like this:
	// TASKLIST.exe /FI "IMAGENAME eq imageName.xe" /FO CSV /V

	shellArgs := []string{
		"/FI", fmt.Sprintf("IMAGENAME eq %s", imageName), // ask tasklist.exe to filter the image name for us
		"/FO", "CSV", // output in CSV format
		"/V", // verbose output - only way to get Window Title
	}

	cmd := exec.Command("TASKLIST.EXE", shellArgs...)
	output, err := cmd.Output()

	if err != nil {
		return "", err
	}

	r := csv.NewReader(bytes.NewReader(output))

	records, err := r.ReadAll()
	if err != nil {
		return "", err
	}

	// TASKLIST returns a CSV list of matches on the above filter. The last column is the Window Title column
	// This takes the first window title that isn't "N/A" and returns that as the title
	// Skips the first row of headers
	for _, row := range records[1:] {
		text := row[len(row)-1]

		if text != "N/A" {
			return text, nil
		}
	}

	return "", nil
}
