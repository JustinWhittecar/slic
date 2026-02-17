package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
)

type MULResponse struct {
	Units []MULUnit `json:"Units"`
}

type MULUnit struct {
	Id          int     `json:"Id"`
	Name        string  `json:"Name"`
	BattleValue int     `json:"BattleValue"`
	Role        MULRole `json:"Role"`
}

type MULRole struct {
	Name string `json:"Name"`
}

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgres://slic:slic@localhost:5432/slic")
	if err != nil {
		log.Fatalf("DB connect: %v", err)
	}
	defer conn.Close(ctx)

	// Get all distinct chassis names
	rows, err := conn.Query(ctx, "SELECT DISTINCT c.name FROM chassis c ORDER BY c.name")
	if err != nil {
		log.Fatalf("Query chassis: %v", err)
	}
	var chassisNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatalf("Scan: %v", err)
		}
		chassisNames = append(chassisNames, name)
	}
	rows.Close()

	// Get all mul_ids in our DB for fast lookup
	mulRows, err := conn.Query(ctx, "SELECT mul_id FROM variants WHERE mul_id IS NOT NULL")
	if err != nil {
		log.Fatalf("Query mul_ids: %v", err)
	}
	ourMulIDs := make(map[int]bool)
	for mulRows.Next() {
		var id int
		if err := mulRows.Scan(&id); err != nil {
			log.Fatalf("Scan: %v", err)
		}
		ourMulIDs[id] = true
	}
	mulRows.Close()

	fmt.Printf("Found %d chassis, %d variants with mul_id\n", len(chassisNames), len(ourMulIDs))

	client := &http.Client{Timeout: 15 * time.Second}
	matched, updated, errors := 0, 0, 0

	for i, chassis := range chassisNames {
		if (i+1)%50 == 0 {
			fmt.Printf("Progress: %d/%d chassis processed, %d matched, %d updated\n", i+1, len(chassisNames), matched, updated)
		}

		apiURL := "http://masterunitlist.info/Unit/QuickList?Name=" + url.QueryEscape(chassis)
		resp, err := client.Get(apiURL)
		if err != nil {
			log.Printf("WARN: fetch %q: %v", chassis, err)
			errors++
			time.Sleep(200 * time.Millisecond)
			continue
		}

		var mulResp MULResponse
		if err := json.NewDecoder(resp.Body).Decode(&mulResp); err != nil {
			resp.Body.Close()
			log.Printf("WARN: decode %q: %v", chassis, err)
			errors++
			time.Sleep(200 * time.Millisecond)
			continue
		}
		resp.Body.Close()

		for _, unit := range mulResp.Units {
			if !ourMulIDs[unit.Id] {
				continue
			}
			matched++
			tag, err := conn.Exec(ctx,
				"UPDATE variants SET battle_value = $1, role = $2 WHERE mul_id = $3",
				unit.BattleValue, unit.Role.Name, unit.Id)
			if err != nil {
				log.Printf("WARN: update mul_id=%d: %v", unit.Id, err)
				continue
			}
			if tag.RowsAffected() > 0 {
				updated++
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	fmt.Printf("\nDone! %d chassis processed, %d matched, %d updated, %d errors\n",
		len(chassisNames), matched, updated, errors)
}
