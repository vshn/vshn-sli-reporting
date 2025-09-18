package mock

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/vshn/vshn-sli-reporting/pkg/types"
)

type MockDowntimeStore struct {
	DoError         bool
	ReturnValues    []types.DowntimeWindow
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
func (m *MockDowntimeStore) StoreNewWindow(w types.DowntimeWindow) (types.DowntimeWindow, error) {
	m.LastCall = "create"
	if m.DoError {
		return types.DowntimeWindow{}, errors.New("some error")
	}
	return w, nil
}
func (m *MockDowntimeStore) ListWindows(from time.Time, to time.Time) ([]types.DowntimeWindow, error) {
	m.LastCall = "list"
	if m.DoError {
		return nil, errors.New("some error")
	}
	m.LastCallFrom = from
	m.LastCallTo = to
	return slices.Clone(m.ReturnValues), nil
}
func (m *MockDowntimeStore) ListWindowsMatchingClusterFacts(ctx context.Context, from time.Time, to time.Time, clusterId string) ([]types.DowntimeWindow, error) {
	m.LastCall = "listcluster"
	if m.DoError {
		return nil, errors.New("some error")
	}
	m.LastCallFrom = from
	m.LastCallTo = to
	m.LastCallCluster = clusterId
	return slices.Clone(m.ReturnValues), nil
}
func (m *MockDowntimeStore) UpdateWindow(w types.DowntimeWindow) (types.DowntimeWindow, error) {
	m.LastCall = "update"
	if m.DoError {
		return types.DowntimeWindow{}, errors.New("some error")
	}
	return w, nil
}
func (m *MockDowntimeStore) PatchWindow(w types.DowntimeWindow) (types.DowntimeWindow, error) {
	m.LastCall = "patch"
	if m.DoError {
		return types.DowntimeWindow{}, errors.New("some error")
	}
	return w, nil
}
