package store

import (
	"context"

	"github.com/rapid-saas/router-api/internal/model"
)

// SQLStore is the durable repository boundary for SQLite/libSQL/sqlc-backed
// implementations. It deliberately does not own migrations.
type SQLStore struct {
	db SQLExecutor
}

func NewSQLStore(db SQLExecutor) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) Ping(ctx context.Context) error {
	if s.db == nil {
		return ErrNotFound
	}
	return s.db.QueryRowContext(ctx, "SELECT 1").Scan(new(int))
}

func (s *SQLStore) ListEndpoints(context.Context, string) ([]model.Endpoint, error) {
	return nil, errSQLStoreNotImplemented()
}

func (s *SQLStore) CreateEndpoint(context.Context, *model.Endpoint) error {
	return errSQLStoreNotImplemented()
}

func (s *SQLStore) GetEndpoint(context.Context, string, string) (model.Endpoint, error) {
	return model.Endpoint{}, errSQLStoreNotImplemented()
}

func (s *SQLStore) UpdateEndpoint(context.Context, *model.Endpoint) error {
	return errSQLStoreNotImplemented()
}

func (s *SQLStore) DeleteEndpoint(context.Context, string, string) error {
	return errSQLStoreNotImplemented()
}

func (s *SQLStore) ListDestinations(context.Context, string) ([]model.Destination, error) {
	return nil, errSQLStoreNotImplemented()
}

func (s *SQLStore) CreateDestination(context.Context, *model.Destination) error {
	return errSQLStoreNotImplemented()
}

func (s *SQLStore) GetDestination(context.Context, string, string) (model.Destination, error) {
	return model.Destination{}, errSQLStoreNotImplemented()
}

func (s *SQLStore) UpdateDestination(context.Context, *model.Destination) error {
	return errSQLStoreNotImplemented()
}

func (s *SQLStore) DeleteDestination(context.Context, string, string) error {
	return errSQLStoreNotImplemented()
}

func (s *SQLStore) ListRules(context.Context, string, string) ([]model.Rule, error) {
	return nil, errSQLStoreNotImplemented()
}

func (s *SQLStore) CreateRule(context.Context, *model.Rule) error {
	return errSQLStoreNotImplemented()
}

func (s *SQLStore) GetRule(context.Context, string, string) (model.Rule, error) {
	return model.Rule{}, errSQLStoreNotImplemented()
}

func (s *SQLStore) UpdateRule(context.Context, *model.Rule) error {
	return errSQLStoreNotImplemented()
}

func (s *SQLStore) DeleteRule(context.Context, string, string) error {
	return errSQLStoreNotImplemented()
}

func (s *SQLStore) WriteDeliveryLog(context.Context, model.DeliveryLog) error {
	return errSQLStoreNotImplemented()
}

func errSQLStoreNotImplemented() error {
	return ErrNotImplemented
}
