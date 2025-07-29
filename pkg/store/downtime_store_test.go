package store

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vshn/vshn-sli-reporting/pkg/types"
)

type mockLieutenant struct {
	ReturnVal map[string]string
}

func (m *mockLieutenant) GetClusterFacts(ctx context.Context, clusterID string) (map[string]string, error) {
	return m.ReturnVal, nil
}

func setup() *DowntimeStore {
	time.Local = time.UTC
	store, err := NewDowntimeStore(":memory:", &mockLieutenant{})
	if err != nil {
		log.Fatal(err)
	}
	return store
}

func setupAndSeed(facts map[string]string) *DowntimeStore {
	time.Local = time.UTC
	store, err := NewDowntimeStore(":memory:", &mockLieutenant{ReturnVal: facts})
	if err != nil {
		log.Fatal(err)
	}
	time1, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-01-02T00:00:00Z")
	time3, _ := time.Parse(time.RFC3339, "2020-01-03T00:00:00Z")
	time4, _ := time.Parse(time.RFC3339, "2020-01-04T00:00:00Z")
	time5, _ := time.Parse(time.RFC3339, "2020-01-05T00:00:00Z")
	time6, _ := time.Parse(time.RFC3339, "2020-01-06T00:00:00Z")
	store.InitializeDB()
	store.StoreNewWindow(&types.DowntimeWindow{
		StartTime: &time1,
		EndTime:   &time2,
		Title:     "Test1",
		Affects:   []types.AffectedClusterMatcher{}, // matches nothing
	})
	store.StoreNewWindow(&types.DowntimeWindow{
		StartTime: &time2,
		EndTime:   &time3,
		Title:     "Test2",
		Affects:   []types.AffectedClusterMatcher{types.AffectedClusterMatcher{}}, // matches all
	})
	store.StoreNewWindow(&types.DowntimeWindow{
		StartTime: &time3,
		EndTime:   &time4,
		Title:     "Test3",
		Affects: []types.AffectedClusterMatcher{
			map[string]string{"foo": "bar"},
		},
	})
	store.StoreNewWindow(&types.DowntimeWindow{
		StartTime: &time4,
		EndTime:   &time5,
		Title:     "Test4",
		Affects: []types.AffectedClusterMatcher{
			map[string]string{
				"foo": "bar",
				"baz": "quux",
			},
		},
	})
	store.StoreNewWindow(&types.DowntimeWindow{
		StartTime: &time5,
		EndTime:   &time6,
		Title:     "Test5",
		Affects: []types.AffectedClusterMatcher{
			map[string]string{
				"foo": "bar",
			},
			map[string]string{
				"foo": "quack",
			},
		},
	})
	store.StoreNewWindow(&types.DowntimeWindow{
		StartTime: &time1,
		// No end time
		Title:     "Test6",
		Affects: []types.AffectedClusterMatcher{
			map[string]string{
				"foo": "box",
			},
			map[string]string{
				"foo": "quack",
			},
			map[string]string{
				"du": "hans",
			},
		},
	})
	return store
}

func TestInitializeDB(t *testing.T) {
	store := setup()
	err := store.InitializeDB()
	assert.NoError(t, err)

	windows, err := store.ListWindows(time.Time{}, time.Time{})
	assert.NoError(t, err)

	assert.Equal(t, 0, len(windows))
}

func TestListWindowsInTimeFrame(t *testing.T) {
	time1, _ := time.Parse(time.RFC3339, "2020-01-02T12:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-01-04T12:00:00Z")

	store := setupAndSeed(map[string]string{})

	windows, err := store.ListWindows(time1, time2)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(windows))

	names := make([]string, 4)
	for i, w := range windows {
		names[i] = w.Title
	}
	assert.Contains(t, names, "Test2")
	assert.Contains(t, names, "Test3")
	assert.Contains(t, names, "Test4")
	assert.Contains(t, names, "Test6")

}

func TestListWindowsStoreError(t *testing.T) {
	time1, _ := time.Parse(time.RFC3339, "2020-01-02T12:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-01-04T12:00:00Z")

	store := setup()

	_, err := store.ListWindows(time1, time2)
	assert.Error(t, err)

}

func TestListWindowsForCluster(t *testing.T) {
	time1, _ := time.Parse(time.RFC3339, "2019-12-02T12:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-02-04T12:00:00Z")

	store := setupAndSeed(map[string]string{"foo": "bar"})

	windows, err := store.ListWindowsMatchingClusterFacts(context.TODO(), time1, time2, "unused")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(windows))

	names := make([]string, 3)
	for i, w := range windows {
		names[i] = w.Title
	}
	assert.Contains(t, names, "Test2")
	assert.Contains(t, names, "Test3")
	assert.Contains(t, names, "Test5")

}

func TestStoreNewWindow(t *testing.T) {
	store := setup()
	time1, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-01-02T00:00:00Z")
	store.InitializeDB()
	w, err := store.StoreNewWindow(&types.DowntimeWindow{
		StartTime:    &time1,
		EndTime:      &time2,
		Title:        "Test1",
		Description:  "sdf",
		ExternalID:   "a",
		ExternalLink: "ds",
		Affects:      []types.AffectedClusterMatcher{},
	})

	assert.NoError(t, err)

	assert.NotEmpty(t, w.ID)

	w2, err := store.getWindowById(w.ID)
	assert.NoError(t, err)
	wc, err := convertFromDbStruct(&w2)
	assert.NoError(t, err)
	assert.Equal(t, w, wc)
}

func TestStoreNewWindowInvalid(t *testing.T) {
	store := setup()
	time1, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-01-02T00:00:00Z")
	store.InitializeDB()
	_, err := store.StoreNewWindow(&types.DowntimeWindow{
		// start time after end time
		StartTime:    &time2,
		EndTime:      &time1,
		Title:        "Test1",
		Description:  "sdf",
		ExternalID:   "a",
		ExternalLink: "ds",
		Affects:      []types.AffectedClusterMatcher{},
	})

	assert.Error(t, err)

	_, err = store.StoreNewWindow(&types.DowntimeWindow{
		// no start time
		EndTime:      &time1,
		Title:        "Test1",
		Description:  "sdf",
		ExternalID:   "a",
		ExternalLink: "ds",
		Affects:      []types.AffectedClusterMatcher{},
	})

	assert.Error(t, err)

}

func TestStoreNewWindowWithSameExtId(t *testing.T) {
	store := setup()
	time1, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-01-02T00:00:00Z")
	store.InitializeDB()
	w, err := store.StoreNewWindow(&types.DowntimeWindow{
		StartTime:  &time1,
		Title:      "Test1",
		ExternalID: "a",
		Affects:    []types.AffectedClusterMatcher{},
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, w.ID)

	w2, err := store.StoreNewWindow(&types.DowntimeWindow{
		StartTime:  &time1,
		EndTime:    &time2,
		Title:      "Test1",
		ExternalID: "a",
		Affects:    []types.AffectedClusterMatcher{},
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, w2.ID)

	assert.Equal(t, w.ID, w2.ID)
	assert.True(t, time2.Equal(*w2.EndTime))

	windows, err := store.ListWindows(time1, time2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(windows))

}

func TestUpdateWindow(t *testing.T) {
	store := setup()
	time1, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-01-02T00:00:00Z")
	store.InitializeDB()
	w, err := store.StoreNewWindow(&types.DowntimeWindow{
		StartTime:  &time1,
		Title:      "Test1",
		ExternalID: "a",
		Affects:    []types.AffectedClusterMatcher{},
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, w.ID)

	w2, err := store.UpdateWindow(&types.DowntimeWindow{
		ID:          w.ID,
		StartTime:   &time1,
		EndTime:     &time2,
		Title:       "TestX",
		Description: "asf",
		ExternalID:  "b",
		Affects:     []types.AffectedClusterMatcher{},
	})
	assert.NoError(t, err)

	w3, err := store.getWindowById(w.ID)
	assert.NoError(t, err)
	wc, err := convertFromDbStruct(&w3)
	assert.NoError(t, err)
	assert.Equal(t, w2, wc)

	windows, err := store.ListWindows(time1, time2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(windows))

}

func TestUpdateFailsWithExtIdConflict(t *testing.T) {
	store := setup()
	time1, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-01-02T00:00:00Z")
	store.InitializeDB()
	_, err := store.StoreNewWindow(&types.DowntimeWindow{
		StartTime:  &time1,
		Title:      "Test1",
		ExternalID: "a",
		Affects:    []types.AffectedClusterMatcher{},
	})
	assert.NoError(t, err)
	w, err := store.StoreNewWindow(&types.DowntimeWindow{
		StartTime:  &time1,
		Title:      "Test2",
		ExternalID: "b",
		Affects:    []types.AffectedClusterMatcher{},
	})

	assert.NoError(t, err)

	_, err = store.UpdateWindow(&types.DowntimeWindow{
		ID:         w.ID,
		StartTime:  &time1,
		EndTime:    &time2,
		ExternalID: "a",
		Affects:    []types.AffectedClusterMatcher{},
	})
	assert.Error(t, err)
}

func TestPatchWindow(t *testing.T) {
	store := setup()
	time1, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	time2, _ := time.Parse(time.RFC3339, "2020-01-02T00:00:00Z")
	store.InitializeDB()
	w, err := store.StoreNewWindow(&types.DowntimeWindow{
		StartTime:   &time1,
		EndTime:     &time2,
		Title:       "Test1",
		Description: "asf",
		Affects:     []types.AffectedClusterMatcher{},
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, w.ID)

	w2, err := store.PatchWindow(&types.DowntimeWindow{
		ID:         w.ID,
		Title:      "TestX",
		ExternalID: "a",
		Affects:     []types.AffectedClusterMatcher{
			map[string]string{"baz": "quux"},
		},
	})
	assert.NoError(t, err)

	assert.Equal(t, "asf", w2.Description)
	assert.Equal(t, "a", w2.ExternalID)
	assert.Equal(t, "TestX", w2.Title)
	assert.Equal(t, 1, len(w2.Affects))
	assert.True(t, time1.Equal(*w2.StartTime))
	assert.True(t, time2.Equal(*w2.EndTime))

	windows, err := store.ListWindows(time1, time2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(windows))

}
