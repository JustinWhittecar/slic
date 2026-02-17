package db

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/JustinWhittecar/slic/internal/ingestion"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{Pool: pool}
}

func normalizeTechBase(tb string) string {
	lower := strings.ToLower(tb)
	if strings.Contains(lower, "mixed") {
		return "Mixed"
	}
	if strings.Contains(lower, "clan") {
		return "Clan"
	}
	return "Inner Sphere"
}

func (s *Store) UpsertChassis(ctx context.Context, tx pgx.Tx, name string, tonnage int, techBase string) (int, error) {
	tb := normalizeTechBase(techBase)
	var id int
	// Try insert, on conflict just select
	err := tx.QueryRow(ctx,
		`INSERT INTO chassis (name, tonnage, tech_base)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (name) DO UPDATE SET tonnage = EXCLUDED.tonnage
		 RETURNING id`, name, tonnage, tb).Scan(&id)
	return id, err
}

func (s *Store) InsertVariant(ctx context.Context, tx pgx.Tx, chassisID int, data *ingestion.MTFData) (int, error) {
	era := eraFromYear(data.Era)
	var mulID *int
	if data.MulID > 0 {
		mulID = &data.MulID
	}
	var introYear *int
	if data.Era > 0 {
		introYear = &data.Era
	}
	var rulesLevel *int
	if data.RulesLevel > 0 {
		rulesLevel = &data.RulesLevel
	}

	var id int
	err := tx.QueryRow(ctx,
		`INSERT INTO variants (chassis_id, model_code, name, mul_id, config, source, rules_level, intro_year, era)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id`,
		chassisID, data.Model, data.FullName(), mulID, data.Config, data.Source, rulesLevel, introYear, era,
	).Scan(&id)
	return id, err
}

func (s *Store) InsertVariantStats(ctx context.Context, tx pgx.Tx, variantID int, data *ingestion.MTFData) error {
	runMP := int(math.Ceil(float64(data.WalkMP) * 1.5))
	isTotal := internalStructureTotal(data.Mass)

	_, err := tx.Exec(ctx,
		`INSERT INTO variant_stats
		 (variant_id, walk_mp, run_mp, jump_mp, armor_total, internal_structure_total,
		  heat_sink_count, heat_sink_type, engine_type, engine_rating,
		  cockpit_type, gyro_type, myomer_type, structure_type, armor_type)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		variantID, data.WalkMP, runMP, data.JumpMP, data.TotalArmor(), isTotal,
		data.HeatSinkCount, data.HeatSinkType, data.EngineType, data.EngineRating,
		data.Cockpit, data.Gyro, data.Myomer, data.Structure, data.ArmorType,
	)
	return err
}

func (s *Store) IngestMTF(ctx context.Context, data *ingestion.MTFData) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	chassisID, err := s.UpsertChassis(ctx, tx, data.Chassis, data.Mass, data.TechBase)
	if err != nil {
		return fmt.Errorf("upsert chassis %q: %w", data.Chassis, err)
	}

	variantID, err := s.InsertVariant(ctx, tx, chassisID, data)
	if err != nil {
		return fmt.Errorf("insert variant %q: %w", data.FullName(), err)
	}

	if err := s.InsertVariantStats(ctx, tx, variantID, data); err != nil {
		return fmt.Errorf("insert stats for %q: %w", data.FullName(), err)
	}

	return tx.Commit(ctx)
}

func eraFromYear(year int) string {
	if year <= 0 {
		return ""
	}
	switch {
	case year <= 2570:
		return "Age of War"
	case year <= 2780:
		return "Star League"
	case year <= 2900:
		return "Early Succession Wars"
	case year <= 3049:
		return "Late Succession Wars"
	case year <= 3061:
		return "Clan Invasion"
	case year <= 3067:
		return "Civil War"
	case year <= 3081:
		return "Jihad"
	case year <= 3150:
		return "Dark Age"
	default:
		return "ilClan"
	}
}

// Standard internal structure points by tonnage (BattleTech standard).
var isTable = map[int]int{
	10: 17, 15: 25, 20: 33, 25: 42, 30: 51, 35: 60, 40: 68,
	45: 77, 50: 85, 55: 94, 60: 102, 65: 111, 70: 119, 75: 128,
	80: 136, 85: 145, 90: 153, 95: 162, 100: 171,
}

func internalStructureTotal(tonnage int) int {
	if v, ok := isTable[tonnage]; ok {
		return v
	}
	// Fallback: approximate
	return int(float64(tonnage) * 1.7)
}
