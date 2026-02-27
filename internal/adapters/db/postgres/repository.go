package postgres

import (
	"context"
	"fmt"
	"time"

	"golang-sms-broadcast/internal/domain"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Repository implements ports.MessageRepository using PostgreSQL with GORM.
type Repository struct {
	db *gorm.DB
}

// New opens a PostgreSQL connection using GORM and runs auto-migration.
func New(dsn string) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres with gorm: %w", err)
	}

	// Get underlying SQL DB for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB from gorm: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	// Auto-migrate schemas
	fmt.Println("🔄 Running GORM auto-migration...")
	if err := db.AutoMigrate(&domain.Broadcast{}, &domain.Message{}); err != nil {
		return nil, fmt.Errorf("auto-migrate: %w", err)
	}
	fmt.Println("✅ Auto-migration complete")

	return &Repository{db: db}, nil
}

// Close closes the underlying database connection pool.
func (r *Repository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping checks if the database connection is alive.
func (r *Repository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// SaveBroadcast inserts a new broadcast row.
func (r *Repository) SaveBroadcast(ctx context.Context, b domain.Broadcast) error {
	if err := r.db.WithContext(ctx).Create(&b).Error; err != nil {
		return fmt.Errorf("create broadcast: %w", err)
	}
	return nil
}

// SaveMessages inserts a batch of messages inside a single transaction.
func (r *Repository) SaveMessages(ctx context.Context, msgs []domain.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).CreateInBatches(msgs, 100).Error; err != nil {
		return fmt.Errorf("create messages: %w", err)
	}
	return nil
}

// GetPendingMessages returns up to limit messages with StatusPending.
func (r *Repository) GetPendingMessages(ctx context.Context, limit int) ([]domain.Message, error) {
	var msgs []domain.Message
	err := r.db.WithContext(ctx).
		Where("status = ?", domain.StatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&msgs).Error

	if err != nil {
		return nil, fmt.Errorf("find pending messages: %w", err)
	}
	return msgs, nil
}

// UpdateMessageStatus transitions a message to the given status.
func (r *Repository) UpdateMessageStatus(ctx context.Context, id uuid.UUID, status domain.Status) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("update message status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("message not found: %s", id)
	}

	return nil
}

// UpdateMessageStatusByProviderID transitions a message by the provider's external ID.
func (r *Repository) UpdateMessageStatusByProviderID(ctx context.Context, providerID string, status domain.Status) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Message{}).
		Where("provider_id = ?", providerID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("update message status by provider id: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("message not found for provider_id: %s", providerID)
	}

	return nil
}

// SetProviderID stores the external SMS provider ID on a message after submission.
func (r *Repository) SetProviderID(ctx context.Context, id uuid.UUID, providerID string) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"provider_id": providerID,
			"updated_at":  time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("set provider id: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("message not found: %s", id)
	}

	return nil
}

// GetBroadcast retrieves a broadcast by ID with all its messages.
func (r *Repository) GetBroadcast(ctx context.Context, id uuid.UUID) (*domain.Broadcast, error) {
	var broadcast domain.Broadcast
	err := r.db.WithContext(ctx).
		Preload("Messages").
		Where("id = ?", id).
		First(&broadcast).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("broadcast not found: %s", id)
		}
		return nil, fmt.Errorf("get broadcast: %w", err)
	}

	return &broadcast, nil
}
