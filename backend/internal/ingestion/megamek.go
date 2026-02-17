package ingestion

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// parseArmorValue handles both standard "26" and patchwork "Reactive(Inner Sphere):26" formats
func parseArmorValue(val string) int {
	// Try direct parse first
	if n, err := strconv.Atoi(val); err == nil {
		return n
	}
	// Patchwork format: "ArmorType:value"
	if idx := strings.LastIndex(val, ":"); idx >= 0 {
		if n, err := strconv.Atoi(val[idx+1:]); err == nil {
			return n
		}
	}
	return 0
}

// MTFData holds all parsed data from a MegaMek .mtf file.
type MTFData struct {
	// Header
	Chassis   string
	Model     string
	MulID     int
	Config    string
	TechBase  string
	Era       int
	Source    string
	RulesLevel int

	// Quirks
	Quirks []string

	// Core
	Mass          int
	EngineRating  int
	EngineType    string
	Structure     string
	Myomer        string
	Cockpit       string
	Gyro          string

	// Heat sinks
	HeatSinkCount int
	HeatSinkType  string

	// Movement
	WalkMP int
	JumpMP int

	// Armor
	ArmorType   string
	ArmorValues map[string]int // location -> armor points

	// Weapons summary
	Weapons []WeaponEntry

	// Per-location equipment slots
	LocationEquipment map[string][]string

	// Lore
	Overview     string
	Capabilities string
	Deployment   string
	History      string

	// Manufacturer
	Manufacturer      string
	PrimaryFactory    string
	SystemManufacturer map[string]string // system -> manufacturer line
}

// WeaponEntry is a weapon from the Weapons:N summary block.
type WeaponEntry struct {
	Name     string
	Location string
}

// ParseMTF reads a MegaMek .mtf file and returns structured data.
func ParseMTF(path string) (*MTFData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open mtf: %w", err)
	}
	defer f.Close()

	data := &MTFData{
		ArmorValues:        make(map[string]int),
		LocationEquipment:  make(map[string][]string),
		SystemManufacturer: make(map[string]string),
	}

	scanner := bufio.NewScanner(f)
	// Increase buffer for files with long lore lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var currentLocation string
	var inWeapons bool

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		lower := strings.ToLower(trimmed)

		// Check if we're entering a location block
		if loc := matchLocationHeader(trimmed); loc != "" {
			currentLocation = loc
			inWeapons = false
			continue
		}

		// Check for weapons section
		if strings.HasPrefix(lower, "weapons:") {
			inWeapons = true
			currentLocation = ""
			continue
		}

		// If in a location block, collect equipment
		if currentLocation != "" {
			data.LocationEquipment[currentLocation] = append(data.LocationEquipment[currentLocation], trimmed)
			continue
		}

		// If in weapons section, parse weapon entries
		if inWeapons {
			if parts := strings.SplitN(trimmed, ",", 2); len(parts) == 2 {
				data.Weapons = append(data.Weapons, WeaponEntry{
					Name:     strings.TrimSpace(parts[0]),
					Location: strings.TrimSpace(parts[1]),
				})
			}
			continue
		}

		// Parse key:value fields
		if idx := strings.Index(trimmed, ":"); idx >= 0 {
			key := strings.ToLower(strings.TrimSpace(trimmed[:idx]))
			val := strings.TrimSpace(trimmed[idx+1:])

			switch key {
			case "chassis":
				data.Chassis = val
			case "model":
				data.Model = val
			case "mul id":
				data.MulID, _ = strconv.Atoi(val)
			case "config":
				data.Config = val
			case "techbase":
				data.TechBase = val
			case "era":
				data.Era, _ = strconv.Atoi(val)
			case "source":
				data.Source = val
			case "rules level":
				data.RulesLevel, _ = strconv.Atoi(val)
			case "quirk":
				if val != "" {
					data.Quirks = append(data.Quirks, val)
				}
			case "mass":
				data.Mass, _ = strconv.Atoi(val)
			case "engine":
				data.EngineRating, data.EngineType = parseEngine(val)
			case "structure":
				data.Structure = val
			case "myomer":
				data.Myomer = val
			case "cockpit":
				data.Cockpit = val
			case "gyro":
				data.Gyro = val
			case "ejection":
				// skip ejection type
			case "heat sinks":
				data.HeatSinkCount, data.HeatSinkType = parseHeatSinks(val)
			case "base chassis heat sinks":
				// skip, secondary heat sink info for omnimechs
			case "walk mp":
				data.WalkMP, _ = strconv.Atoi(val)
			case "jump mp":
				data.JumpMP, _ = strconv.Atoi(val)
			case "armor":
				data.ArmorType = val
			case "la armor":
				data.ArmorValues["LA"] = parseArmorValue(val)
			case "ra armor":
				data.ArmorValues["RA"] = parseArmorValue(val)
			case "lt armor":
				data.ArmorValues["LT"] = parseArmorValue(val)
			case "rt armor":
				data.ArmorValues["RT"] = parseArmorValue(val)
			case "ct armor":
				data.ArmorValues["CT"] = parseArmorValue(val)
			case "hd armor":
				data.ArmorValues["HD"] = parseArmorValue(val)
			case "ll armor":
				data.ArmorValues["LL"] = parseArmorValue(val)
			case "rl armor":
				data.ArmorValues["RL"] = parseArmorValue(val)
			case "rtl armor":
				data.ArmorValues["RTL"] = parseArmorValue(val)
			case "rtr armor":
				data.ArmorValues["RTR"] = parseArmorValue(val)
			case "rtc armor":
				data.ArmorValues["RTC"] = parseArmorValue(val)
			// Quad leg locations
			case "fll armor":
				data.ArmorValues["FLL"] = parseArmorValue(val)
			case "frl armor":
				data.ArmorValues["FRL"] = parseArmorValue(val)
			case "rll armor":
				data.ArmorValues["RLL"] = parseArmorValue(val)
			case "rrl armor":
				data.ArmorValues["RRL"] = parseArmorValue(val)
			// Lore
			case "overview":
				data.Overview = val
			case "capabilities":
				data.Capabilities = val
			case "deployment":
				data.Deployment = val
			case "history":
				data.History = val
			// Manufacturer
			case "manufacturer":
				data.Manufacturer = val
			case "primaryfactory":
				data.PrimaryFactory = val
			case "systemmanufacturer":
				if parts := strings.SplitN(val, ":", 2); len(parts) == 2 {
					data.SystemManufacturer[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			case "systemmode":
				// skip
			case "nocrit":
				// skip
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan mtf: %w", err)
	}

	// Validate minimum required fields
	if data.Chassis == "" {
		return nil, fmt.Errorf("missing chassis field")
	}

	return data, nil
}

// matchLocationHeader checks if a line is a location header like "Left Arm:" or "Front Left Leg:"
func matchLocationHeader(line string) string {
	locations := []string{
		"Left Arm:",
		"Right Arm:",
		"Left Torso:",
		"Right Torso:",
		"Center Torso:",
		"Head:",
		"Left Leg:",
		"Right Leg:",
		// Quad mech locations
		"Front Left Leg:",
		"Front Right Leg:",
		"Rear Left Leg:",
		"Rear Right Leg:",
		// LAM locations
		"Center Leg:",
	}
	for _, loc := range locations {
		if line == loc {
			return strings.TrimSuffix(loc, ":")
		}
	}
	return ""
}

// parseEngine parses "300 Fusion Engine(IS)" -> (300, "Fusion Engine(IS)")
func parseEngine(val string) (int, string) {
	parts := strings.SplitN(val, " ", 2)
	if len(parts) < 2 {
		rating, _ := strconv.Atoi(val)
		return rating, ""
	}
	rating, _ := strconv.Atoi(parts[0])
	return rating, parts[1]
}

// parseHeatSinks parses "14 IS Double" -> (14, "IS Double")
func parseHeatSinks(val string) (int, string) {
	parts := strings.SplitN(val, " ", 2)
	if len(parts) < 2 {
		count, _ := strconv.Atoi(val)
		return count, "Single"
	}
	count, _ := strconv.Atoi(parts[0])
	return count, parts[1]
}

// TotalArmor returns the sum of all armor values.
func (d *MTFData) TotalArmor() int {
	total := 0
	for _, v := range d.ArmorValues {
		total += v
	}
	return total
}

// FullName returns "Chassis Model" or just "Chassis" if model is empty.
func (d *MTFData) FullName() string {
	if d.Model == "" {
		return d.Chassis
	}
	return d.Chassis + " " + d.Model
}
