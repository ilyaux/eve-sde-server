package main

import (
	"flag"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ilyaux/eve-sde-server/internal/database"
	"github.com/ilyaux/eve-sde-server/internal/sde"
)

func main() {
	// Setup logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Parse flags
	skipDownload := flag.Bool("skip-download", false, "Skip downloading SDE (use existing files)")
	sdeURL := flag.String("url", sde.DefaultSDEURL, "SDE download URL")
	dataDir := flag.String("data-dir", "data/sde", "SDE data directory")
	dbPath := flag.String("db", "data/sde.db", "Database path")
	flag.Parse()

	log.Info().Msg("EVE SDE Import Tool")

	// Step 1: Download SDE (if needed)
	var sdeDir string
	var checksum string
	if !*skipDownload {
		downloader := sde.NewDownloader(*sdeURL, *dataDir)

		zipPath, downloadedChecksum, err := downloader.Download()
		if err != nil {
			log.Fatal().Err(err).Msg("download failed")
		}
		checksum = downloadedChecksum

		log.Info().Str("checksum", checksum[:16]+"...").Msg("download complete")

		// Extract
		extractDir := *dataDir + "/extracted"
		if err := downloader.Extract(zipPath, extractDir); err != nil {
			log.Fatal().Err(err).Msg("extraction failed")
		}

		sdeDir = extractDir
	} else {
		log.Info().Str("dir", *dataDir).Msg("skipping download, using existing SDE")
		sdeDir = *dataDir
	}

	// Step 2: Open database
	db, err := database.New(*dbPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// Step 3: Parse and import
	parser := sde.NewParser(sdeDir)
	importer := sde.NewImporter(db)

	if err := importer.ImportAll(parser); err != nil {
		log.Fatal().Err(err).Msg("import failed")
	}

	// Step 4: Verify
	var itemCount, categoryCount, groupCount int
	db.QueryRow("SELECT COUNT(*) FROM items").Scan(&itemCount)
	db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&categoryCount)
	db.QueryRow("SELECT COUNT(*) FROM groups").Scan(&groupCount)

	if checksum != "" {
		if _, err := db.Exec(`
			INSERT INTO sde_versions (
				version,
				checksum,
				downloaded_at,
				imported_at,
				items_count
			)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(checksum) DO UPDATE SET
				imported_at = excluded.imported_at,
				items_count = excluded.items_count,
				error = NULL
		`, time.Now().Format("20060102"), checksum, time.Now(), time.Now(), itemCount); err != nil {
			log.Warn().Err(err).Msg("failed to record SDE version")
		}
	}

	log.Info().
		Int("items", itemCount).
		Int("categories", categoryCount).
		Int("groups", groupCount).
		Msg("Import completed successfully")

	log.Info().Msg("You can now start the server: go run cmd/server/main.go")
}
