package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rapid-saas/router-api/internal/model"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrConflict       = errors.New("conflict")
	ErrNotImplemented = errors.New("not implemented")
)

// SQLExecutor is intentionally compatible with database/sql, modernc SQLite,
// and libSQL database handles used by sqlc-generated query packages.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type HealthRepository interface {
	Ping(context.Context) error
}

type AdminRepository interface {
	HealthRepository
	ListEndpoints(context.Context, string) ([]model.Endpoint, error)
	CreateEndpoint(context.Context, *model.Endpoint) error
	GetEndpoint(context.Context, string, string) (model.Endpoint, error)
	UpdateEndpoint(context.Context, *model.Endpoint) error
	DeleteEndpoint(context.Context, string, string) error
	ListDestinations(context.Context, string) ([]model.Destination, error)
	CreateDestination(context.Context, *model.Destination) error
	GetDestination(context.Context, string, string) (model.Destination, error)
	UpdateDestination(context.Context, *model.Destination) error
	DeleteDestination(context.Context, string, string) error
	ListRules(context.Context, string, string) ([]model.Rule, error)
	CreateRule(context.Context, *model.Rule) error
	GetRule(context.Context, string, string) (model.Rule, error)
	UpdateRule(context.Context, *model.Rule) error
	DeleteRule(context.Context, string, string) error
}

type ProcessingRepository interface {
	HealthRepository
	GetEndpoint(context.Context, string, string) (model.Endpoint, error)
	GetDestination(context.Context, string, string) (model.Destination, error)
	ListRules(context.Context, string, string) ([]model.Rule, error)
	WriteDeliveryLog(context.Context, model.DeliveryLog) error
}
