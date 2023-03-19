package db

import (
	"errors"

	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db/models"
	"github.com/robherley/snips.sh/internal/logger"
	zl "github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Sqlite struct {
	*gorm.DB
}

func NewSqlite(cfg config.DB) (DB, error) {
	gormdb, err := gorm.Open(sqlite.Open(cfg.FilePath), &gorm.Config{
		Logger: &logger.GormAdapter{ZL: zl.Logger},
	})
	if err != nil {
		return nil, err
	}

	return &Sqlite{gormdb}, nil
}

func (sql *Sqlite) Migrate() error {
	return sql.AutoMigrate(models.All...)
}

func (sql *Sqlite) File(id string) (*models.File, error) {
	var file models.File
	err := sql.Where("id = ?", id).First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &file, nil
}

func (sql *Sqlite) NewFile(file *models.File) error {
	return sql.Create(file).Error
}

func (sql *Sqlite) UpdateFile(file *models.File) error {
	return sql.Save(file).Error
}

func (sql *Sqlite) DeleteFile(id string) error {
	return sql.Delete(&models.File{}, id).Error
}

func (sql *Sqlite) FileForUser(id, userID string) (*models.File, error) {
	var file models.File
	err := sql.Where("id = ? AND user_id = ?", id, userID).First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &file, nil
}

func (sql *Sqlite) FilesForUser(userID string, withContent bool) ([]models.File, error) {
	var files []models.File

	scope := sql.Where("user_id = ?", userID).Order("created_at DESC")

	if !withContent {
		scope = scope.Omit("content")
	}

	err := scope.Find(&files).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return files, nil
}

func (sql *Sqlite) PublicKeyForFingerprint(fingerprint string) (*models.PublicKey, error) {
	var key models.PublicKey
	err := sql.Where("fingerprint = ?", fingerprint).First(&key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &key, nil
}

func (sql *Sqlite) NewUser(publickey *models.PublicKey) (*models.User, error) {
	user := models.User{
		PublicKeys: []models.PublicKey{
			*publickey,
		},
	}

	err := sql.Create(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (sql *Sqlite) User(id string) (*models.User, error) {
	var user models.User
	err := sql.Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}
