package specs

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/canonical/specs-v2.canonical.com/googledrive"
	"gorm.io/gorm"
)

// SyncService handles the synchronization of specification documents from Google Drive
type SyncService struct {
	Logger      *slog.Logger
	DriveClient *googledrive.GoogleDrive
	DB          *gorm.DB
	Config      SyncConfig

	FailedCount  int
	SkippedCount int
}

type SyncConfig struct {
	RootFolderID  string
	MaxGoroutines int
	// ForceSync forces the synchronization of all specs without checking the last updated time
	ForceSync bool
}

type WorkerItem struct {
	File         googledrive.FileResult
	ParentFolder googledrive.FileResult
}

// NewSyncService creates a new specification synchronization service
func NewSyncService(logger *slog.Logger, driveClient *googledrive.GoogleDrive, db *gorm.DB, config SyncConfig) *SyncService {
	return &SyncService{
		Logger:      logger.With("component", "specs_sync"),
		DriveClient: driveClient,
		DB:          db,
		Config:      config,
	}
}

func (s *SyncService) worker(ctx context.Context, id int, items <-chan *WorkerItem) {
	logger := s.Logger.With("worker_id", id)

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopped due to cancellation")
			return
		case item, ok := <-items:
			if !ok {
				return
			}
			logger := logger.With("file_id", item.File.File.Id, "file_name", item.File.File.Name)
			err := s.Parse(ctx, logger, item)
			if err != nil {
				logger.Error("failed to parse file", "error", err.Error())
				s.FailedCount++
			}
		}
	}
}

// SyncSpecs synchronizes the specification documents from Google Drive
func (s *SyncService) SyncSpecs(ctx context.Context) error {
	s.Logger.Info("starting specs synchronization",
		"root_folder_id", s.Config.RootFolderID,
		"max_goroutines", s.Config.MaxGoroutines,
	)
	s.FailedCount = 0
	s.SkippedCount = 0
	startTime := time.Now()

	workerItems := make(chan *WorkerItem, s.Config.MaxGoroutines)
	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < s.Config.MaxGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s.worker(ctx, id, workerItems)
		}(i)
	}

	// Process folders and send files to workers
	folderChan := s.DriveClient.GetSubFoldersChannel(ctx, s.Config.RootFolderID)
	totalCount := int32(0)
	go func() {
		defer close(workerItems)
		for folder := range folderChan {
			if ctx.Err() != nil {
				return
			}

			logger := s.Logger.With("folder_id", folder.File.Id, "folder_name", folder.File.Name)

			if folder.Err != nil {
				logger.Error("failed to get subfolders", "error", folder.Err.Error())
				continue
			}

			subFolderFilesChan := s.DriveClient.GetFilesInFolderChannel(ctx, folder.File.Id)
			logger.Info("processing folder")

			folderCount := 0
			for file := range subFolderFilesChan {
				if ctx.Err() != nil {
					return
				}

				atomic.AddInt32(&totalCount, 1)
				folderCount++

				if file.Err != nil {
					logger.Error("failed to get files", "error", file.Err.Error())
					continue
				}

				workerItem := &WorkerItem{
					File:         file,
					ParentFolder: folder,
				}

				select {
				case workerItems <- workerItem:
				case <-ctx.Done():
					return
				}
			}

			logger.Info("folder processed", "folder_count", folderCount)
		}
	}()

	// Wait for all workers to finish
	wg.Wait()

	deletedSpecs := s.DB.Exec("DELETE FROM specs WHERE synced_at < ?", startTime).RowsAffected
	s.Logger.Info("deleted old specs", "count", deletedSpecs)

	s.Logger.Info("specs synchronization completed",
		"duration", time.Since(startTime).Seconds(),
		"total_count", totalCount,
		"failed_count", s.FailedCount,
		"skipped_count", s.SkippedCount,
	)

	return ctx.Err()
}
