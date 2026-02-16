package models

type Chassis struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Tonnage  int    `json:"tonnage"`
	TechBase string `json:"tech_base"`
	SarnaURL string `json:"sarna_url,omitempty"`
}

type Variant struct {
	ID        int    `json:"id"`
	ChassisID int    `json:"chassis_id"`
	ModelCode string `json:"model_code"`
	Name      string `json:"name"`
	BV        *int   `json:"battle_value,omitempty"`
	IntroYear *int   `json:"intro_year,omitempty"`
	Era       string `json:"era,omitempty"`
	Role      string `json:"role,omitempty"`
}

type MechListItem struct {
	ID        int    `json:"id"`
	ModelCode string `json:"model_code"`
	Name      string `json:"name"`
	Chassis   string `json:"chassis"`
	Tonnage   int    `json:"tonnage"`
	TechBase  string `json:"tech_base"`
	BV        *int   `json:"battle_value,omitempty"`
	IntroYear *int   `json:"intro_year,omitempty"`
	Era       string `json:"era,omitempty"`
	Role      string `json:"role,omitempty"`
}

type Equipment struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Damage      float64 `json:"damage,omitempty"`
	Heat        int     `json:"heat,omitempty"`
	MinRange    int     `json:"min_range,omitempty"`
	ShortRange  int     `json:"short_range,omitempty"`
	MediumRange int     `json:"medium_range,omitempty"`
	LongRange   int     `json:"long_range,omitempty"`
	Tonnage     float64 `json:"tonnage"`
	Slots       int     `json:"slots"`
}

type VariantEquipment struct {
	Equipment
	Location string `json:"location"`
	Quantity int    `json:"quantity"`
}

type VariantStats struct {
	WalkMP         int    `json:"walk_mp"`
	RunMP          int    `json:"run_mp"`
	JumpMP         int    `json:"jump_mp"`
	ArmorTotal     int    `json:"armor_total"`
	ISTotal        int    `json:"internal_structure_total"`
	HeatSinkCount  int    `json:"heat_sink_count"`
	HeatSinkType   string `json:"heat_sink_type"`
	EngineType     string `json:"engine_type"`
	EngineRating   int    `json:"engine_rating"`
}

type MechDetail struct {
	MechListItem
	SarnaURL  string             `json:"sarna_url,omitempty"`
	Stats     *VariantStats      `json:"stats,omitempty"`
	Equipment []VariantEquipment `json:"equipment,omitempty"`
}
