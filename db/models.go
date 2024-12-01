package db

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Spec struct {
	ID                 string         `gorm:"type:text;primaryKey"`
	Title              *string        `gorm:"type:text"`
	Status             *string        `gorm:"type:text"`
	Authors            pq.StringArray `gorm:"type:text[]"`
	SpecType           *string        `gorm:"type:text;column:spec_type"`
	Team               string         `gorm:"type:text;not null"`
	GoogleDocID        string         `gorm:"type:text;not null;column:google_doc_id"`
	GoogleDocName      string         `gorm:"type:text;not null;column:google_doc_name"`
	GoogleDocURL       string         `gorm:"type:text;not null;column:google_doc_url"`
	GoogleDocCreatedAt time.Time      `gorm:"not null;column:google_doc_created_at"`
	GoogleDocUpdatedAt time.Time      `gorm:"not null;column:google_doc_updated_at"`
	OpenComments       []string       `gorm:"type:text[];column:open_comments"`
	TotalComments      []string       `gorm:"type:text[];column:total_comments"`
	SpecHTML           *string        `gorm:"type:text;column:spec_html"`
	CreatedAt          time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt          time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	SyncedAt           time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func Migrate(db *gorm.DB) error {
	// Create the specs table
	if err := db.AutoMigrate(&Spec{}); err != nil {
		return err
	}

	// Create trigger for updating updated_at
	return db.Exec(`
        CREATE OR REPLACE FUNCTION update_specs_updated_at_column() 
        RETURNS TRIGGER AS $$ 
        BEGIN 
            NEW.updated_at = CURRENT_TIMESTAMP;
            RETURN NEW;
        END;
        $$ LANGUAGE plpgsql;

        DROP TRIGGER IF EXISTS update_specs_updated_at ON specs;
        
        CREATE TRIGGER update_specs_updated_at 
        BEFORE UPDATE ON specs 
        FOR EACH ROW 
        EXECUTE FUNCTION update_specs_updated_at_column();
    `).Error
}

func Rollback(db *gorm.DB) error {
	return db.Exec(`
        DROP TRIGGER IF EXISTS update_specs_updated_at ON specs;
        DROP FUNCTION IF EXISTS update_specs_updated_at_column();
        DROP TABLE IF EXISTS specs;
    `).Error
}
