package web

import (
	"context"

	"github.com/niklucky/signal/internal/models"
	"github.com/niklucky/signal/internal/storage"
)

// Store defines the storage operations used by the web UI.
type Store interface {
	GetUserByEmail(ctx context.Context, email string) (*storage.User, error)
	GetUserByID(ctx context.Context, id int64) (*storage.User, error)
	UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error
	UpdateUserProfile(ctx context.Context, id int64, email string) error
	GetHostsByUser(ctx context.Context, userID int64) ([]storage.Host, error)
	GetHostByID(ctx context.Context, id int64) (*storage.Host, error)
	CreateHost(ctx context.Context, userID int64, host models.Host, expectedStatus int, active bool) (int64, error)
	UpdateHost(ctx context.Context, id int64, host models.Host, expectedStatus int, active bool) error
	DeleteHost(ctx context.Context, id int64) error
	DashboardData(ctx context.Context, userID int64) ([]storage.DashboardHost, error)
	HostTimeSeries(ctx context.Context, hostID int64, window string) ([]storage.TimePoint, error)
	HostHourlySeries(ctx context.Context, hostID int64, window string) ([]storage.HostHourlyAvg, error)
}
