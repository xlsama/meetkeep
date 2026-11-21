// Package store keeps meetings and what was said in them.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/egoist/mygo/plugins/sqlite"

	"github.com/xlsama/meetkeep/internal/config"
)

type MeetingStatus string

const (
	Recording MeetingStatus = "recording"
	Done      MeetingStatus = "done"
)

type Track string

const (
	Mic    Track = "mic"
	System Track = "system"
)

type Meeting struct {
	ID         string          `json:"id"`
	Title      string          `json:"title"`
	StartedAt  int64           `json:"startedAt"`
	DurationMs int64           `json:"durationMs"`
	Status     MeetingStatus   `json:"status"`
	Directory  string          `json:"directory"`
	Microphone bool            `json:"microphone"`
	System     bool            `json:"system"`
	Apps       []config.AppRef `json:"apps"`
}

type Utterance struct {
	Seq     int    `json:"seq"`
	Track   Track  `json:"track"`
	StartMs int64  `json:"startMs"`
	EndMs   int64  `json:"endMs"`
	Text    string `json:"text"`
}

var ErrNotFound = errors.New("找不到这场会议")

var migrations = [][]string{
	{
		`CREATE TABLE meetings (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			started_at INTEGER NOT NULL,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			directory TEXT NOT NULL,
			microphone INTEGER NOT NULL,
			system INTEGER NOT NULL,
			apps TEXT NOT NULL
		)`,
		`CREATE INDEX meetings_started ON meetings (started_at DESC)`,
		`CREATE TABLE utterances (
			meeting_id TEXT NOT NULL REFERENCES meetings (id) ON DELETE CASCADE,
			seq INTEGER NOT NULL,
			track TEXT NOT NULL,
			start_ms INTEGER NOT NULL,
			end_ms INTEGER NOT NULL,
			text TEXT NOT NULL,
			PRIMARY KEY (meeting_id, seq)
		)`,
	},
}

type Store struct {
	db *sqlite.DB
}

func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sqlite.Open(ctx, path, sqlite.OpenOptions{})
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	rows, err := s.db.Query(ctx, "PRAGMA user_version")
	if err != nil {
		return err
	}
	version := int(rows.Values[0][0].(int64))
	if version > len(migrations) {
		return fmt.Errorf("数据库来自更新的版本（%d）", version)
	}
	for ; version < len(migrations); version++ {
		var batch []sqlite.Statement
		for _, sql := range migrations[version] {
			batch = append(batch, sqlite.Statement{SQL: sql})
		}
		batch = append(batch, sqlite.Statement{SQL: fmt.Sprintf("PRAGMA user_version = %d", version+1)})
		if _, err := s.db.Transaction(ctx, batch); err != nil {
			return err
		}
	}
	return nil
}

// Recover ends the meetings a crash left recording.
func (s *Store) Recover(ctx context.Context) error {
	_, err := s.db.Execute(ctx, `UPDATE meetings SET status = ?, duration_ms = MAX(duration_ms,
		COALESCE((SELECT MAX(end_ms) FROM utterances WHERE meeting_id = meetings.id), 0))
		WHERE status = ?`, string(Done), string(Recording))
	return err
}

func (s *Store) CreateMeeting(ctx context.Context, m Meeting) error {
	apps, err := json.Marshal(m.Apps)
	if err != nil {
		return err
	}
	_, err = s.db.Execute(ctx, `INSERT INTO meetings
		(id, title, started_at, duration_ms, status, directory, microphone, system, apps)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.Title, m.StartedAt, m.DurationMs, string(m.Status), m.Directory, m.Microphone, m.System, string(apps))
	return err
}

func (s *Store) FinishMeeting(ctx context.Context, id string, durationMs int64) error {
	_, err := s.db.Execute(ctx, "UPDATE meetings SET status = ?, duration_ms = ? WHERE id = ?",
		string(Done), durationMs, id)
	return err
}

func (s *Store) Rename(ctx context.Context, id, title string) error {
	return s.changed(s.db.Execute(ctx, "UPDATE meetings SET title = ? WHERE id = ?", title, id))
}

func (s *Store) Delete(ctx context.Context, id string) error {
	return s.changed(s.db.Execute(ctx, "DELETE FROM meetings WHERE id = ?", id))
}

func (s *Store) changed(result sqlite.Result, err error) error {
	if err == nil && result.Changes == 0 {
		return ErrNotFound
	}
	return err
}

const meetingColumns = "id, title, started_at, duration_ms, status, directory, microphone, system, apps"

func (s *Store) Meetings(ctx context.Context) ([]Meeting, error) {
	rows, err := s.db.Query(ctx, "SELECT "+meetingColumns+" FROM meetings ORDER BY started_at DESC")
	if err != nil {
		return nil, err
	}
	meetings := make([]Meeting, 0, len(rows.Values))
	for _, row := range rows.Values {
		meetings = append(meetings, meeting(row))
	}
	return meetings, nil
}

func (s *Store) Meeting(ctx context.Context, id string) (Meeting, error) {
	rows, err := s.db.Query(ctx, "SELECT "+meetingColumns+" FROM meetings WHERE id = ?", id)
	if err != nil {
		return Meeting{}, err
	}
	if len(rows.Values) == 0 {
		return Meeting{}, ErrNotFound
	}
	return meeting(rows.Values[0]), nil
}

func meeting(row []any) Meeting {
	m := Meeting{
		ID:         row[0].(string),
		Title:      row[1].(string),
		StartedAt:  row[2].(int64),
		DurationMs: row[3].(int64),
		Status:     MeetingStatus(row[4].(string)),
		Directory:  row[5].(string),
		Microphone: row[6].(int64) != 0,
		System:     row[7].(int64) != 0,
		Apps:       []config.AppRef{},
	}
	_ = json.Unmarshal([]byte(row[8].(string)), &m.Apps)
	return m
}

func (s *Store) AddUtterance(ctx context.Context, meetingID string, u Utterance) error {
	_, err := s.db.Execute(ctx, `INSERT INTO utterances (meeting_id, seq, track, start_ms, end_ms, text)
		VALUES (?, ?, ?, ?, ?, ?)`, meetingID, u.Seq, string(u.Track), u.StartMs, u.EndMs, u.Text)
	return err
}

func (s *Store) Utterances(ctx context.Context, meetingID string) ([]Utterance, error) {
	rows, err := s.db.Query(ctx, `SELECT seq, track, start_ms, end_ms, text FROM utterances
		WHERE meeting_id = ? ORDER BY start_ms, seq`, meetingID)
	if err != nil {
		return nil, err
	}
	utterances := make([]Utterance, 0, len(rows.Values))
	for _, row := range rows.Values {
		utterances = append(utterances, Utterance{
			Seq:     int(row[0].(int64)),
			Track:   Track(row[1].(string)),
			StartMs: row[2].(int64),
			EndMs:   row[3].(int64),
			Text:    row[4].(string),
		})
	}
	return utterances, nil
}
