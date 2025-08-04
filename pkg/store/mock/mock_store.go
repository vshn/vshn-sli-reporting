package mock

import (
	"context"
	"errors"
	"time"

	"github.com/vshn/vshn-sli-reporting/pkg/types"
)


type MockDowntimeStore struct {
	DoError         bool
	ReturnValue     *types.DowntimeWindow
	LastCall        string
	LastCallFrom    time.Time
	LastCallTo      time.Time
	LastCallCluster string
}

func (m *MockDowntimeStore) InitializeDB() error {
	return nil
}
func (m *MockDowntimeStore) CloseDB() error {
	return nil
}
func (m *MockDowntimeStore) StoreNewWindow(w *types.DowntimeWindow) (*types.DowntimeWindow, error) {
	m.LastCall = "create"
	if m.DoError {
		return nil, errors.New("some error")
	}
	return w, nil
}
func (m *MockDowntimeStore) ListWindows(from time.Time, to time.Time) ([]*types.DowntimeWindow, error) {
	m.LastCall = "list"
	if m.DoError {
		return nil, errors.New("some error")
	}
	m.LastCallFrom = from
	m.LastCallTo = to
	return []*types.DowntimeWindow{m.ReturnValue}, nil
}
func (m *MockDowntimeStore) ListWindowsMatchingClusterFacts(ctx context.Context, from time.Time, to time.Time, clusterId string) ([]*types.DowntimeWindow, error) {
	m.LastCall = "listcluster"
	if m.DoError {
		return nil, errors.New("some error")
	}
	m.LastCallFrom = from
	m.LastCallTo = to
	m.LastCallCluster = clusterId
	return []*types.DowntimeWindow{m.ReturnValue}, nil
}
func (m *MockDowntimeStore) UpdateWindow(w *types.DowntimeWindow) (*types.DowntimeWindow, error) {
	m.LastCall = "update"
	if m.DoError {
		return nil, errors.New("some error")
	}
	return w, nil
}
func (m *MockDowntimeStore) PatchWindow(w *types.DowntimeWindow) (*types.DowntimeWindow, error) {
	m.LastCall = "patch"
	if m.DoError {
		return nil, errors.New("some error")
	}
	return w, nil
}
