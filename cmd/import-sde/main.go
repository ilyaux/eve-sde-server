package main

import (
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ilya/eve-sde-server/internal/database"
	"github.com/ilya/eve-sde-server/internal/sde"
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

	log.Info().Msg("🚀 EVE SDE Import Tool")

	// Step 1: Download SDE (if needed)
	var sdeDir string
	if !*skipDownload {
		downloader := sde.NewDownloader(*sdeURL, *dataDir)

		zipPath, checksum, err := downloader.Download()
		if err != nil {
			log.Fatal().Err(err).Msg("download failed")
		}

		log.Info().Str("checksum", checksum[:16]+"...").Msg("download complete")

		// Extract
		extractDir := *dataDir + "/extracted"
		if err := downloader.Extract(zipPath, extractDir); err != nil {
			log.Fatal().Err(err).Msg("extraction failed")
		}

		sdeDir = extractDir
	} else {
		log.Info().Str("dir", *dataDir).Msg("skipping download, using existing SDE")
		sdeDir = *dataDir + "/extracted"
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

	log.Info().
		Int("items", itemCount).
		Int("categories", categoryCount).
		Int("groups", groupCount).
		Msg("✓ Import completed successfully!")

	log.Info().Msg("You can now start the server: go run cmd/server/main.go")
}
