package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"golang-sms-broadcast/internal/domain"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Repository implements ports.MessageRepository using PostgreSQL.
type Repository struct {
	db *sql.DB
}

// New opens a PostgreSQL connection and returns a Repository.
func New(dsn string) (*Repository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Repository{db: db}, nil
}

// Close closes the underlying database connection pool.
func (r *Repository) Close() error {
	return r.db.Close()
}

// SaveBroadcast inserts a new broadcast row.
func (r *Repository) SaveBroadcast(ctx context.Context, b domain.Broadcast) error {
	const q = `
		INSERT INTO broadcasts (id, name, created_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.ExecContext(ctx, q, b.ID, b.Name, b.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert broadcast: %w", err)
	}
	return nil
}

// SaveMessages inserts a batch of messages inside a single transaction.
func (r *Repository) SaveMessages(ctx context.Context, msgs []domain.Message) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	const q = `
		INSERT INTO messages (id, broadcast_id, to_number, body, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("prepare insert message: %w", err)
	}
	defer stmt.Close() //nolint:errcheck

	for _, m := range msgs {
		if _, err := stmt.ExecContext(ctx, m.ID, m.BroadcastID, m.To, m.Body, m.Status, m.CreatedAt, m.UpdatedAt); err != nil {
			return fmt.Errorf("exec insert message %s: %w", m.ID, err)
		}
	}

	return tx.Commit()
}

// GetPendingMessages returns up to limit messages with status 'pending',
// ordered by created_at ascending (oldest first).
func (r *Repository) GetPendingMessages(ctx context.Context, limit int) ([]domain.Message, error) {
	const q = `
		SELECT id, broadcast_id, to_number, body, status, COALESCE(provider_id,''), created_at, updated_at
		FROM messages
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`
	rows, err := r.db.QueryContext(ctx, q, domain.StatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("query pending: %w", err)
	}
	defer rows.Close()

	var msgs []domain.Message
	for rows.Next() {
		var m domain.Message
		var status string
		if err := rows.Scan(&m.ID, &m.BroadcastID, &m.To, &m.Body, &status, &m.ProviderID, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		m.Status = domain.Status(status)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// UpdateMessageStatus transitions a message to the given status by internal ID.
func (r *Repository) UpdateMessageStatus(ctx context.Context, id uuid.UUID, status domain.Status) error {
	const q = `UPDATE messages SET status = $1, updated_at = $2 WHERE id = $3`
	res, err := r.db.ExecContext(ctx, q, status, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrMessageNotFound
	}
	return nil
}

// UpdateMessageStatusByProviderID transitions a message by the provider's external ID.
func (r *Repository) UpdateMessageStatusByProviderID(ctx context.Context, providerID string, status domain.Status) error {
	const q = `UPDATE messages SET status = $1, updated_at = $2 WHERE provider_id = $3`
	res, err := r.db.ExecContext(ctx, q, status, time.Now().UTC(), providerID)
	if err != nil {
		return fmt.Errorf("update status by provider id: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrMessageNotFound
	}
	return nil
}

// SetProviderID stores the external SMS provider ID on a message.
func (r *Repository) SetProviderID(ctx context.Context, id uuid.UUID, providerID string) error {
	const q = `UPDATE messages SET provider_id = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, q, providerID, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("set provider id: %w", err)
	}
	return nil
}
