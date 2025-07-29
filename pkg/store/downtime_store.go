package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/vshn/vshn-sli-reporting/pkg/lieutenant"
	"github.com/vshn/vshn-sli-reporting/pkg/types"

	_ "github.com/mattn/go-sqlite3"
)

type dbDowntimeWindow struct {
	ID           string `db:"id"`
	StartTime    int64  `db:"start_time"`
	EndTime      int64  `db:"end_time"`
	Title        string `db:"title"`
	Description  string `db:"description"`
	ExternalID   string `db:"external_id"`
	ExternalLink string `db:"external_link"`
	Affects      string `db:"affects"`
}

type DowntimeStore struct {
	db *sqlx.DB
	lieutenant *lieutenant.Client
}

func NewDowntimeStore(dbpath string, lieutenant *lieutenant.Client) (*DowntimeStore, error) {
	db, err := sqlx.Open("sqlite3", dbpath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}
	return &DowntimeStore{db: db, lieutenant: lieutenant}, nil
}

func (s *DowntimeStore) InitializeDB() error {
	createSQL := `CREATE TABLE IF NOT EXISTS downtime (
	  "id" TEXT PRIMARY KEY,
	  "start_time" INTEGER NOT NULL,
	  "end_time" INTEGER,
	  "title" TEXT,
	  "description" TEXT,
	  "external_id" TEXT,
	  "external_link" TEXT,
	  "affects" TEXT
	)`

	statement, err := s.db.Prepare(createSQL)
	if err != nil {
		return fmt.Errorf("failed to initialize db: %w", err)
	}
	_, err = statement.Exec()
	if err != nil {
		return fmt.Errorf("failed to initialize db: %w", err)
	}
	return nil
}

func (s *DowntimeStore) CloseDB() error {
	return s.db.Close()
}

func (s *DowntimeStore) StoreNewWindow(w *types.DowntimeWindow) (*types.DowntimeWindow, error) {
	q := `INSERT INTO downtime (id, start_time, end_time, title, description, external_id, external_link, affects) VALUES (:id, :start_time, :end_time, :title, :description, :external_id, :external_link, :affects)`
	st, err := convertToDbStruct(w)

	if err != nil {
		return nil, fmt.Errorf("unable to convert downtime window for store: %w", err)
	}

	err = s.validate(&st)
	if err != nil {
		return nil, fmt.Errorf("invalid downtime window: %w", err)
	}

	existing_id, err := s.idFromExternalID(st.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("error while validating external ID: %w", err)
	}

	if len(existing_id) > 0 && existing_id != st.ID {
		st.ID = existing_id
		return s.updateWindow(&st)
	}

	_, err = s.db.NamedExec(q, st)
	if err != nil {
		return nil, fmt.Errorf("unable to store downtime window: %w", err)
	}

	rv, err := convertFromDbStruct(&st)
	if err != nil {
		return nil, fmt.Errorf("unable to convert store result: %w", err)
	}

	return rv, nil
}

func (s *DowntimeStore) ListWindows(from time.Time, to time.Time) ([]*types.DowntimeWindow, error) {
	fromUnix := from.Unix()
	toUnix := to.Unix()

	results := []dbDowntimeWindow{}

	err := s.db.Select(&results, "SELECT * FROM downtime WHERE (end_time > ? OR end_time <= 0) AND start_time < ?", fromUnix, toUnix)
	if err != nil {
		return []*types.DowntimeWindow{}, fmt.Errorf("error while querying downtime windows: %w", err)
	}

	converted := make([]*types.DowntimeWindow, len(results))
	for i, r := range results {
		c, err := convertFromDbStruct(&r)
		if err != nil {
			return []*types.DowntimeWindow{}, fmt.Errorf("error while converting downtime window list: %w", err)
		}
		converted[i] = c
	}
	return converted, nil
}

func (s *DowntimeStore) ListWindowsMatchingClusterFacts(ctx context.Context, from time.Time, to time.Time, clusterId string) ([]*types.DowntimeWindow, error) {
	facts, err := s.lieutenant.GetClusterFacts(ctx, clusterId)
	if err != nil {
		return nil, fmt.Errorf("could not list downtime windows matching cluster facts: %w", err)
	}
	windows, err := s.ListWindows(from, to)
	if err != nil {
		return nil, err
	}

	matchedWindows := make([]*types.DowntimeWindow, 0)

	for _, w := range windows {
		if windowMatchesClusterFacts(w, facts) {
			matchedWindows = append(matchedWindows, w)
		}
	}

	return matchedWindows, nil
}

func windowMatchesClusterFacts (w *types.DowntimeWindow, facts map[string]string) bool {
	for _, a := range w.Affects {
		matches := true
		for k, v := range a {
			fact, ok := facts[k]
			matches = matches && ok && v == fact
		}
		if matches {
			return true
		}
	}

	return false
}

func (s *DowntimeStore) UpdateWindow(w *types.DowntimeWindow) (*types.DowntimeWindow, error) {
	st, err := convertToDbStruct(w)
	if err != nil {
		return nil, fmt.Errorf("unable to convert downtime window for store: %w", err)
	}

	err = s.validate(&st)
	if err != nil {
		return nil, fmt.Errorf("invalid downtime window: %w", err)
	}

	existing_id, err := s.idFromExternalID(st.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("error while validating external ID: %w", err)
	}

	if len(existing_id) > 0 && existing_id != st.ID {
		return nil, errors.New("could not update downtime window: external ID conflicts with existing record")
	}

	return s.updateWindow(&st)
}

func (s *DowntimeStore) PatchWindow(w *types.DowntimeWindow) (*types.DowntimeWindow, error) {
	existing, err := s.getWindowById(w.ID)
	if err != nil {
		return nil, fmt.Errorf("unable to find existing record for patch: %w", err)
	}
	st, err := updateDbStruct(existing, w)
	if err != nil {
		return nil, fmt.Errorf("unable to convert downtime window for patch: %w", err)
	}

	err = s.validate(&st)
	if err != nil {
		return nil, fmt.Errorf("invalid downtime window: %w", err)
	}

	existing_id, err := s.idFromExternalID(st.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("error while validating external ID: %w", err)
	}

	if len(existing_id) > 0 && existing_id != st.ID {
		return nil, errors.New("could not patch downtime window: external ID conflicts with existing record")
	}

	return s.updateWindow(&st)
}

func (s *DowntimeStore) validate(w *dbDowntimeWindow) error {
	if w.StartTime <= 0 {
		return errors.New("validation error: start time must be set")
	}
	if w.EndTime > 0 && w.StartTime > w.EndTime {
		return errors.New("validation error: end time must be after start time")
	}
	return nil
}

func (s *DowntimeStore) updateWindow(w *dbDowntimeWindow) (*types.DowntimeWindow, error) {
	q := `UPDATE downtime SET id = :id, start_time = :start_time,  end_time = :end_time, title = :title, description = :description, external_id = :external_id, external_link = :external_link, affects = :affects WHERE id == :id`
	_, err := s.db.NamedExec(q, w)
	if err != nil {
		return nil, fmt.Errorf("unable to update downtime window: %w", err)
	}

	rv, err := convertFromDbStruct(w)
	if err != nil {
		return nil, fmt.Errorf("unable to convert update result: %w", err)
	}

	return rv, nil
}

func (s *DowntimeStore) getWindowById(id string) (dbDowntimeWindow, error) {
	result := dbDowntimeWindow{}
	err := s.db.Get(&result, "SELECT * FROM downtime WHERE id == ?", id)
	if err != nil {
		return dbDowntimeWindow{}, fmt.Errorf("error while querying record by ID: %w", err)
	}
	return result, nil
}

func (s *DowntimeStore) idFromExternalID(externalID string) (string, error) {
	if len(externalID) == 0 {
		//NOTE(aa): empty externalIDs are not considered
		return "", nil
	}

	results := []dbDowntimeWindow{}
	err := s.db.Select(&results, "SELECT * FROM downtime WHERE external_id == ? LIMIT 1", externalID)
	if err != nil {
		return "", fmt.Errorf("error while querying for external ID: %w", err)
	}

	if len(results) > 0 {
		return results[0].ID, nil
	}
	return "", nil
}

func convertToDbStruct(w *types.DowntimeWindow) (dbDowntimeWindow, error) {
	affects, err := json.Marshal(w.Affects)
	if err != nil {
		return dbDowntimeWindow{}, fmt.Errorf("could not convert downtime window: %w", err)
	}
	nw := dbDowntimeWindow{
		ID:           w.ID,
		Title:        w.Title,
		Description:  w.Description,
		ExternalID:   w.ExternalID,
		ExternalLink: w.ExternalLink,
		Affects:      string(affects),
	}

	if w.StartTime != nil {
		nw.StartTime = w.StartTime.Unix()
	}
	if w.EndTime != nil {
		nw.EndTime = w.EndTime.Unix()
	}

	if len(nw.ID) == 0 {
		nw.ID = uuid.New().String()
	}

	return nw, nil
}

func convertFromDbStruct(w *dbDowntimeWindow) (*types.DowntimeWindow, error) {
	affects := []types.AffectedClusterMatcher{}
	err := json.Unmarshal([]byte(w.Affects), &affects)
	if err != nil {
		return nil, fmt.Errorf("could not convert downtime window: %w", err)
	}
	st, en := time.Unix(w.StartTime, 0), time.Unix(w.EndTime, 0)
	nw := types.DowntimeWindow{
		ID:           w.ID,
		Title:        w.Title,
		Description:  w.Description,
		ExternalID:   w.ExternalID,
		ExternalLink: w.ExternalLink,
		Affects:      affects,
	}

	if w.StartTime > 0 {
		nw.StartTime = &st
	}
	if w.EndTime > 0 {
		nw.EndTime = &en
	}

	return &nw, nil
}

func updateDbStruct(e dbDowntimeWindow, w *types.DowntimeWindow) (dbDowntimeWindow, error) {
	if e.ID != w.ID {
		return dbDowntimeWindow{}, errors.New("cannot patch record: ID mismatch")
	}

	if w.StartTime != nil {
		e.StartTime = w.StartTime.Unix()
	}
	if w.EndTime != nil {
		e.EndTime = w.EndTime.Unix()
	}
	if len(w.Title) > 0 {
		e.Title = w.Title
	}
	if len(w.Description) > 0 {
		e.Description = w.Description
	}
	if len(w.ExternalID) > 0 {
		e.ExternalID = w.ExternalID
	}
	if len(w.ExternalLink) > 0 {
		e.ExternalLink = w.ExternalLink
	}
	if len(w.Affects) > 0 {
		affects, err := json.Marshal(w.Affects)
		if err != nil {
			return dbDowntimeWindow{}, fmt.Errorf("could not convert downtime window: %w", err)
		}
		e.Affects = string(affects)
	}
	return e, nil
}
