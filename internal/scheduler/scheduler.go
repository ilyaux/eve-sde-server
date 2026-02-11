package scheduler

import (
	"database/sql"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	"github.com/ilya/eve-sde-server/internal/sde"
)

// Scheduler handles automatic SDE updates
type Scheduler struct {
	cron      *cron.Cron
	db        *sql.DB
	sdeURL    string
	dataDir   string
	enabled   bool
	lastCheck time.Time
}

// New creates a new scheduler
func New(db *sql.DB, sdeURL, dataDir string, enabled bool) *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		db:      db,
		sdeURL:  sdeURL,
		dataDir: dataDir,
		enabled: enabled,
	}
}

// Start begins the scheduled tasks
func (s *Scheduler) Start() error {
	if !s.enabled {
		log.Info().Msg("scheduler disabled, skipping auto-updates")
		return nil
	}

	// Check for updates daily at 3 AM UTC
	_, err := s.cron.AddFunc("0 3 * * *", func() {
		log.Info().Msg("running scheduled SDE update check")
		if err := s.updateSDE(); err != nil {
			log.Error().Err(err).Msg("scheduled SDE update failed")
		}
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	log.Info().Msg("scheduler started - will check for SDE updates daily at 03:00 UTC")

	return nil
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done() // Wait for running jobs to complete
		log.Info().Msg("scheduler stopped gracefully")
	}
}

// updateSDE downloads and imports the latest SDE
func (s *Scheduler) updateSDE() error {
	start := time.Now()
	log.Info().Msg("starting SDE update...")

	// Download SDE
	downloader := sde.NewDownloader(s.sdeURL, s.dataDir)
	zipPath, checksum, err := downloader.Download()
	if err != nil {
		return err
	}

	log.Info().Str("checksum", checksum[:16]+"...").Msg("SDE downloaded")

	// Check if this version is already imported
	var existingChecksum string
	s.db.QueryRow("SELECT checksum FROM sde_versions WHERE checksum = ? LIMIT 1", checksum).Scan(&existingChecksum)
	if existingChecksum == checksum {
		log.Info().Msg("SDE version already imported, skipping")
		return nil
	}

	// Extract
	extractDir := s.dataDir + "/extracted"
	if err := downloader.Extract(zipPath, extractDir); err != nil {
		return err
	}

	// Import
	parser := sde.NewParser(extractDir)
	importer := sde.NewImporter(s.db)

	if err := importer.ImportAll(parser); err != nil {
		return err
	}

	// Record version
	_, err = s.db.Exec(`
		INSERT INTO sde_versions (version, checksum, downloaded_at)
		VALUES (?, ?, ?)
	`, time.Now().Format("20060102"), checksum, time.Now())

	s.lastCheck = time.Now()

	log.Info().
		Dur("duration", time.Since(start)).
		Str("checksum", checksum[:16]+"...").
		Msg("SDE update completed successfully")

	return err
}

// TriggerUpdate manually triggers an SDE update
func (s *Scheduler) TriggerUpdate() error {
	log.Info().Msg("manual SDE update triggered")
	return s.updateSDE()
}

// GetLastCheck returns the time of the last update check
func (s *Scheduler) GetLastCheck() time.Time {
	return s.lastCheck
}
