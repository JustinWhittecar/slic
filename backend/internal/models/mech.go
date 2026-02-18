package models

type Chassis struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Tonnage  int    `json:"tonnage"`
	TechBase string `json:"tech_base"`
	SarnaURL string `json:"sarna_url,omitempty"`
}

type Variant struct {
	ID         int    `json:"id"`
	ChassisID  int    `json:"chassis_id"`
	ModelCode  string `json:"model_code"`
	Name       string `json:"name"`
	BV         *int   `json:"battle_value,omitempty"`
	IntroYear  *int   `json:"intro_year,omitempty"`
	Era        string `json:"era,omitempty"`
	Role       string `json:"role,omitempty"`
	MulID      *int   `json:"mul_id,omitempty"`
	Config     string `json:"config,omitempty"`
	Source     string `json:"source,omitempty"`
	RulesLevel *int   `json:"rules_level,omitempty"`
}

type MechListItem struct {
	ID                int     `json:"id"`
	ModelCode         string  `json:"model_code"`
	Name              string  `json:"name"`
	Chassis           string  `json:"chassis"`
	AlternateName     string  `json:"alternate_name,omitempty"`
	Tonnage           int     `json:"tonnage"`
	TechBase          string  `json:"tech_base"`
	BV                *int    `json:"battle_value,omitempty"`
	IntroYear         *int    `json:"intro_year,omitempty"`
	Era               string  `json:"era,omitempty"`
	Role              string  `json:"role,omitempty"`
	TMM               int     `json:"tmm"`
	ArmorCoveragePct  float64 `json:"armor_coverage_pct"`
	HeatNeutralDamage float64 `json:"heat_neutral_damage"`
	WalkMP            int     `json:"walk_mp"`
	JumpMP            int     `json:"jump_mp"`
	ArmorTotal        int     `json:"armor_total"`
	MaxDamage              float64 `json:"max_damage"`
	EffHeatNeutralDamage   float64 `json:"effective_heat_neutral_damage"`
	HeatNeutralRange       string  `json:"heat_neutral_range,omitempty"`
	GameDamage             float64 `json:"game_damage"`
	CombatRating           float64 `json:"combat_rating"`
	EngineType             string  `json:"engine_type,omitempty"`
	EngineRating      int     `json:"engine_rating,omitempty"`
	HeatSinkCount     int     `json:"heat_sink_count,omitempty"`
	HeatSinkType      string  `json:"heat_sink_type,omitempty"`
	RunMP             int     `json:"run_mp,omitempty"`
	RulesLevel        int     `json:"rules_level,omitempty"`
	Source            string  `json:"source,omitempty"`
	Config            string  `json:"config,omitempty"`
}

type Equipment struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Damage          float64 `json:"damage,omitempty"`
	Heat            int     `json:"heat,omitempty"`
	MinRange        int     `json:"min_range,omitempty"`
	ShortRange      int     `json:"short_range,omitempty"`
	MediumRange     int     `json:"medium_range,omitempty"`
	LongRange       int     `json:"long_range,omitempty"`
	ExtremeRange    int     `json:"extreme_range,omitempty"`
	Tonnage         float64 `json:"tonnage"`
	Slots           int     `json:"slots"`
	InternalName    string  `json:"internal_name,omitempty"`
	BV              *int    `json:"bv,omitempty"`
	RackSize        int     `json:"rack_size,omitempty"`
	ExpectedDamage  float64 `json:"expected_damage,omitempty"`
	DamagePerTon    float64 `json:"damage_per_ton,omitempty"`
	DamagePerHeat   float64 `json:"damage_per_heat,omitempty"`
	ToHitModifier   int     `json:"to_hit_modifier,omitempty"`
	EffDamageShort  float64 `json:"effective_damage_short,omitempty"`
	EffDamageMedium float64 `json:"effective_damage_medium,omitempty"`
	EffDamageLong   float64 `json:"effective_damage_long,omitempty"`
	EffDPSTon       float64 `json:"effective_dps_ton,omitempty"`
	EffDPSHeat      float64 `json:"effective_dps_heat,omitempty"`
}

type VariantEquipment struct {
	Equipment
	Location string `json:"location"`
	Quantity int    `json:"quantity"`
}

type VariantStats struct {
	WalkMP                   int     `json:"walk_mp"`
	RunMP                    int     `json:"run_mp"`
	JumpMP                   int     `json:"jump_mp"`
	ArmorTotal               int     `json:"armor_total"`
	ISTotal                  int     `json:"internal_structure_total"`
	HeatSinkCount            int     `json:"heat_sink_count"`
	HeatSinkType             string  `json:"heat_sink_type"`
	EngineType               string  `json:"engine_type"`
	EngineRating             int     `json:"engine_rating"`
	CockpitType              string  `json:"cockpit_type,omitempty"`
	GyroType                 string  `json:"gyro_type,omitempty"`
	MyomerType               string  `json:"myomer_type,omitempty"`
	StructureType            string  `json:"structure_type,omitempty"`
	ArmorType                string  `json:"armor_type,omitempty"`
	TMM                      int     `json:"tmm"`
	ArmorCoveragePct         float64 `json:"armor_coverage_pct"`
	HeatNeutralDamage        float64 `json:"heat_neutral_damage"`
	HeatNeutralRange         string  `json:"heat_neutral_range,omitempty"`
	MaxDamage                float64 `json:"max_damage"`
	EffHeatNeutralDamage     float64 `json:"effective_heat_neutral_damage"`
	HasTargetingComputer     bool    `json:"has_targeting_computer"`
	CombatRating             float64 `json:"combat_rating,omitempty"`
	OffenseTurns             float64 `json:"offense_turns,omitempty"`
	DefenseTurns             float64 `json:"defense_turns,omitempty"`
}

type MechDetail struct {
	MechListItem
	SarnaURL    string             `json:"sarna_url,omitempty"`
	IWMUrl      string             `json:"iwm_url,omitempty"`
	CatalystUrl string             `json:"catalyst_url,omitempty"`
	Stats       *VariantStats      `json:"stats,omitempty"`
	Equipment   []VariantEquipment `json:"equipment,omitempty"`
}
