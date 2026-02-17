package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JustinWhittecar/slic/internal/db"
	"github.com/JustinWhittecar/slic/internal/ingestion"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dir := flag.String("dir", ".", "Path to mekfiles directory")
	dsn := flag.String("db", "postgres://slic:slic@localhost:5432/slic?sslmode=disable", "Postgres connection string")
	dryRun := flag.Bool("dry-run", false, "Parse only, do not insert into DB")
	verbose := flag.Bool("verbose", false, "Print each parsed mech")
	flag.Parse()

	var files []string
	err := filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mtf") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d .mtf files\n", len(files))

	// Connect to DB unless dry-run
	var store *db.Store
	if !*dryRun {
		ctx := context.Background()
		pool, err := pgxpool.New(ctx, *dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "DB connect error: %v\n", err)
			os.Exit(1)
		}
		defer pool.Close()
		if err := pool.Ping(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "DB ping error: %v\n", err)
			os.Exit(1)
		}
		store = db.NewStore(pool)
		fmt.Println("Connected to database")
	}

	var parsed, failed, inserted int
	chassisSet := map[string]bool{}
	var errors []string

	for i, f := range files {
		data, err := ingestion.ParseMTF(f)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("  %s: %v", filepath.Base(f), err))
			continue
		}
		parsed++

		if *verbose {
			fmt.Printf("  %-40s %3dt  %-20s era:%d\n", data.FullName(), data.Mass, data.TechBase, data.Era)
		}

		if store != nil {
			if err := store.IngestMTF(context.Background(), data); err != nil {
				failed++
				errors = append(errors, fmt.Sprintf("  %s: %v", filepath.Base(f), err))
				continue
			}
			inserted++
			chassisSet[data.Chassis] = true
		}

		if (i+1)%500 == 0 {
			fmt.Printf("  Progress: %d / %d files processed\n", i+1, len(files))
		}
	}

	fmt.Printf("\nResults:\n")
	fmt.Printf("  Parsed:   %d / %d (%.1f%%)\n", parsed, len(files), float64(parsed)/float64(len(files))*100)
	fmt.Printf("  Failed:   %d\n", failed)
	if store != nil {
		fmt.Printf("  Inserted: %d variants across %d chassis\n", inserted, len(chassisSet))
	}

	if len(errors) > 0 {
		fmt.Printf("\nFirst %d errors:\n", min(len(errors), 20))
		for i, e := range errors {
			if i >= 20 {
				break
			}
			fmt.Println(e)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
