package ingestion

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// MTFData holds raw parsed data from a MegaMek .mtf file.
type MTFData struct {
	Chassis   string
	Model     string
	Config    string
	TechBase  string
	Era       string
	Rules     string
	Mass      string
	Engine    string
	Structure string
	HeatSinks string
	Equipment map[string][]string // location -> equipment names
}

// ParseMTF reads a MegaMek .mtf file and returns structured data.
func ParseMTF(path string) (*MTFData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open mtf: %w", err)
	}
	defer f.Close()

	data := &MTFData{
		Equipment: make(map[string][]string),
	}

	scanner := bufio.NewScanner(f)
	var currentSection string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section headers
		switch {
		case strings.HasPrefix(line, "chassis:"):
			data.Chassis = strings.TrimPrefix(line, "chassis:")
			data.Chassis = strings.TrimSpace(data.Chassis)
		case strings.HasPrefix(line, "model:"):
			data.Model = strings.TrimPrefix(line, "model:")
			data.Model = strings.TrimSpace(data.Model)
		case strings.HasPrefix(line, "Config:"):
			data.Config = strings.TrimPrefix(line, "Config:")
			data.Config = strings.TrimSpace(data.Config)
		case strings.HasPrefix(line, "TechBase:"):
			data.TechBase = strings.TrimPrefix(line, "TechBase:")
			data.TechBase = strings.TrimSpace(data.TechBase)
		case strings.HasPrefix(line, "Era:"):
			data.Era = strings.TrimPrefix(line, "Era:")
			data.Era = strings.TrimSpace(data.Era)
		case strings.HasPrefix(line, "Rules Level:"):
			data.Rules = strings.TrimPrefix(line, "Rules Level:")
			data.Rules = strings.TrimSpace(data.Rules)
		case strings.HasPrefix(line, "Mass:"):
			data.Mass = strings.TrimPrefix(line, "Mass:")
			data.Mass = strings.TrimSpace(data.Mass)
		case strings.HasPrefix(line, "Engine:"):
			data.Engine = strings.TrimPrefix(line, "Engine:")
			data.Engine = strings.TrimSpace(data.Engine)
		case strings.HasPrefix(line, "Structure:"):
			data.Structure = strings.TrimPrefix(line, "Structure:")
			data.Structure = strings.TrimSpace(data.Structure)
		case strings.HasPrefix(line, "Heat Sinks:"):
			data.HeatSinks = strings.TrimPrefix(line, "Heat Sinks:")
			data.HeatSinks = strings.TrimSpace(data.HeatSinks)
		case line == "Left Arm:" || line == "Right Arm:" ||
			line == "Left Torso:" || line == "Right Torso:" ||
			line == "Center Torso:" || line == "Head:" ||
			line == "Left Leg:" || line == "Right Leg:":
			currentSection = strings.TrimSuffix(line, ":")
		default:
			if currentSection != "" && !strings.HasPrefix(line, "-Empty-") {
				data.Equipment[currentSection] = append(data.Equipment[currentSection], line)
			}
		}
	}

	return data, scanner.Err()
}
