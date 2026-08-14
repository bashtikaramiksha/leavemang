package notification

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DB() *sql.DB {
	return r.db
}

// Create inserts a new notification into the database.
func (r *Repository) Create(ctx context.Context, n *Notification) (*Notification, error) {
	query := `
		INSERT INTO notifications (user_id, type, title, message, reference_type, reference_id, is_read, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, CURRENT_TIMESTAMP)
	`
	res, err := r.db.ExecContext(ctx, query, n.UserID, n.Type, n.Title, n.Message, n.ReferenceType, n.ReferenceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get notification last insert ID: %w", err)
	}

	return r.GetByID(ctx, id)
}

// ListByUser retrieves all notifications for a given user ordered by created_at DESC.
func (r *Repository) ListByUser(ctx context.Context, userID int64) ([]*Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, reference_type, reference_id, is_read, created_at, read_at
		FROM notifications
		WHERE user_id = ?
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		var n Notification
		var createdAtStr string
		var readAtStr sql.NullString

		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.ReferenceType, &n.ReferenceID, &n.IsRead, &createdAtStr, &readAtStr); err != nil {
			return nil, fmt.Errorf("failed to scan notification row: %w", err)
		}

		if t, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
			n.CreatedAt = t
		} else if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			n.CreatedAt = t
		} else {
			n.CreatedAt = time.Now()
		}

		if readAtStr.Valid && readAtStr.String != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", readAtStr.String); err == nil {
				n.ReadAt = &t
			} else if t, err := time.Parse(time.RFC3339, readAtStr.String); err == nil {
				n.ReadAt = &t
			}
		}

		notifications = append(notifications, &n)
	}

	return notifications, nil
}

// GetByID retrieves a notification by its ID.
func (r *Repository) GetByID(ctx context.Context, id int64) (*Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, reference_type, reference_id, is_read, created_at, read_at
		FROM notifications
		WHERE id = ?
	`
	var n Notification
	var createdAtStr string
	var readAtStr sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.ReferenceType, &n.ReferenceID, &n.IsRead, &createdAtStr, &readAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("failed to get notification by ID: %w", err)
	}

	if t, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
		n.CreatedAt = t
	} else if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		n.CreatedAt = t
	} else {
		n.CreatedAt = time.Now()
	}

	if readAtStr.Valid && readAtStr.String != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", readAtStr.String); err == nil {
			n.ReadAt = &t
		} else if t, err := time.Parse(time.RFC3339, readAtStr.String); err == nil {
			n.ReadAt = &t
		}
	}

	return &n, nil
}

// MarkAsRead sets is_read = 1 and read_at = CURRENT_TIMESTAMP for a notification belonging to userID.
func (r *Repository) MarkAsRead(ctx context.Context, id int64, userID int64) error {
	query := `
		UPDATE notifications
		SET is_read = 1, read_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`
	res, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

// MarkAllAsRead sets is_read = 1 and read_at = CURRENT_TIMESTAMP for all unread notifications of userID.
func (r *Repository) MarkAllAsRead(ctx context.Context, userID int64) error {
	query := `
		UPDATE notifications
		SET is_read = 1, read_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND is_read = 0
	`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}
	return nil
}

// CountUnread returns the number of unread notifications for userID.
func (r *Repository) CountUnread(ctx context.Context, userID int64) (int, error) {
	query := `
		SELECT COUNT(1)
		FROM notifications
		WHERE user_id = ? AND is_read = 0
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}
	return count, nil
}

// ExistsDuplicate checks if a notification of the same type and reference already exists for user_id.
func (r *Repository) ExistsDuplicate(ctx context.Context, userID int64, nType, refType string, refID int64) (bool, error) {
	query := `
		SELECT COUNT(1)
		FROM notifications
		WHERE user_id = ? AND type = ? AND reference_type = ? AND reference_id = ?
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, userID, nType, refType, refID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetManagerUserIDs retrieves user_ids for users with role 'manager' or 'admin'.
func (r *Repository) GetManagerUserIDs(ctx context.Context) ([]int64, error) {
	query := `SELECT id FROM users WHERE role IN ('manager', 'admin') AND status = 'active'`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query manager users: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// GetEmployeeUserID retrieves the user_id corresponding to an employee_id.
func (r *Repository) GetEmployeeUserID(ctx context.Context, employeeID int64) (int64, error) {
	query := `SELECT user_id FROM employees WHERE id = ?`
	var userID int64
	err := r.db.QueryRowContext(ctx, query, employeeID).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("failed to query employee user_id: %w", err)
	}
	return userID, nil
}
