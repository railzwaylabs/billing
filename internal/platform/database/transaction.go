package database

import (
	"context"

	"gorm.io/gorm"
)

type transactionKey struct{}

type Manager struct {
	db *gorm.DB
}

func NewManager(db *gorm.DB) *Manager { return &Manager{db: db} }

func (m *Manager) Within(ctx context.Context, operation func(context.Context) error) error {
	if existing, ok := ctx.Value(transactionKey{}).(*gorm.DB); ok {
		return operation(context.WithValue(ctx, transactionKey{}, existing.WithContext(ctx)))
	}

	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return operation(context.WithValue(ctx, transactionKey{}, tx))
	})
}

func FromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(transactionKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return fallback.WithContext(ctx)
}
