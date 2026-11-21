// Package store keeps meetings and what was said in them.
package store

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/plugins/sqlite"

	"github.com/xlsama/meetkeep/internal/config"
)

type MeetingStatus string

const (
	Recording  MeetingStatus = "recording"
	Processing MeetingStatus = "processing"
	Done       MeetingStatus = "done"
	Failed     MeetingStatus = "failed"
)

type Track string

const (
	Mic    Track = "mic"
	System Track = "system"
	// File is the audio of an imported file.
	File Track = "file"
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
	// Error tells why Status is failed.
	Error string `json:"error"`
	// Audio names the file in Directory to play the meeting from, once it
	// is processed.
	Audio string `json:"audio"`
}

// Speaker is someone who talks in a meeting.
type Speaker struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Self marks the user.
	Self       bool  `json:"self"`
	DurationMs int64 `json:"durationMs"`
}

type Utterance struct {
	Seq     int    `json:"seq"`
	Track   Track  `json:"track"`
	StartMs int64  `json:"startMs"`
	EndMs   int64  `json:"endMs"`
	Text    string `json:"text"`
	// Speaker is the ID of who says it, or 0 before the meeting is
	// processed.
	Speaker int `json:"speaker"`
	// Uncertain marks a guess at the speaker.
	Uncertain bool `json:"uncertain"`
}

var ErrNotFound = errors.New("找不到这场会议")

// NewID returns the ID of a meeting that starts now: the time, so that
// the meetings' directories sort by it, and a random suffix.
func NewID(now time.Time) string {
	var suffix [2]byte
	_, _ = rand.Read(suffix[:])
	return fmt.Sprintf("%s-%x", now.Format("20060102-150405"), suffix)
}

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
	{
		`ALTER TABLE meetings ADD COLUMN error TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE meetings ADD COLUMN audio TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE utterances ADD COLUMN speaker INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE utterances ADD COLUMN uncertain INTEGER NOT NULL DEFAULT 0`,
		`CREATE TABLE speakers (
			meeting_id TEXT NOT NULL REFERENCES meetings (id) ON DELETE CASCADE,
			id INTEGER NOT NULL,
			name TEXT NOT NULL,
			self INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (meeting_id, id)
		)`,
	},
	{
		`CREATE TABLE notes (
			meeting_id TEXT PRIMARY KEY REFERENCES meetings (id) ON DELETE CASCADE,
			data TEXT NOT NULL DEFAULT '',
			text TEXT NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE boards (
			meeting_id TEXT PRIMARY KEY REFERENCES meetings (id) ON DELETE CASCADE,
			scene TEXT NOT NULL
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

// Recover ends the meetings a crash left recording and returns those that
// still have to be processed.
func (s *Store) Recover(ctx context.Context) ([]string, error) {
	_, err := s.db.Execute(ctx, `UPDATE meetings SET status = ?, duration_ms = MAX(duration_ms,
		COALESCE((SELECT MAX(end_ms) FROM utterances WHERE meeting_id = meetings.id), 0))
		WHERE status = ?`, string(Processing), string(Recording))
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, "SELECT id FROM meetings WHERE status = ? ORDER BY started_at", string(Processing))
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows.Values))
	for i, row := range rows.Values {
		ids[i] = row[0].(string)
	}
	return ids, nil
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

// FinishMeeting ends the recording of a meeting, which then waits to be
// processed.
func (s *Store) FinishMeeting(ctx context.Context, id string, durationMs int64) error {
	_, err := s.db.Execute(ctx, "UPDATE meetings SET status = ?, duration_ms = ? WHERE id = ?",
		string(Processing), durationMs, id)
	return err
}

// SetStatus changes a meeting's status; failure tells why it is failed.
func (s *Store) SetStatus(ctx context.Context, id string, status MeetingStatus, failure string) error {
	return s.changed(s.db.Execute(ctx, "UPDATE meetings SET status = ?, error = ? WHERE id = ?",
		string(status), failure, id))
}

func (s *Store) SetDuration(ctx context.Context, id string, durationMs int64) error {
	return s.changed(s.db.Execute(ctx, "UPDATE meetings SET duration_ms = ? WHERE id = ?", durationMs, id))
}

// SaveNotes stores a meeting's notes: data as they are read back, and text,
// their words, to search.
func (s *Store) SaveNotes(ctx context.Context, meetingID, data, text string) error {
	_, err := s.db.Execute(ctx, `INSERT INTO notes (meeting_id, data, text, error) VALUES (?, ?, ?, '')
		ON CONFLICT (meeting_id) DO UPDATE SET data = excluded.data, text = excluded.text, error = ''`,
		meetingID, data, text)
	return err
}

// SetNotesError records why a meeting's notes could not be written, and
// keeps the notes it has.
func (s *Store) SetNotesError(ctx context.Context, meetingID, failure string) error {
	_, err := s.db.Execute(ctx, `INSERT INTO notes (meeting_id, error) VALUES (?, ?)
		ON CONFLICT (meeting_id) DO UPDATE SET error = excluded.error`, meetingID, failure)
	return err
}

// Notes returns a meeting's notes as SaveNotes took them, "" when it has
// none, and why the last attempt to write them failed, if it did.
func (s *Store) Notes(ctx context.Context, meetingID string) (data, failure string, err error) {
	rows, err := s.db.Query(ctx, "SELECT data, error FROM notes WHERE meeting_id = ?", meetingID)
	if err != nil || len(rows.Values) == 0 {
		return "", "", err
	}
	return rows.Values[0][0].(string), rows.Values[0][1].(string), nil
}

// SaveBoard stores the scene of a meeting's whiteboard as the page gave
// it.
func (s *Store) SaveBoard(ctx context.Context, meetingID, scene string) error {
	_, err := s.db.Execute(ctx, `INSERT INTO boards (meeting_id, scene) VALUES (?, ?)
		ON CONFLICT (meeting_id) DO UPDATE SET scene = excluded.scene`, meetingID, scene)
	return err
}

// Board returns the scene of a meeting's whiteboard, or "" when it has
// none.
func (s *Store) Board(ctx context.Context, meetingID string) (string, error) {
	rows, err := s.db.Query(ctx, "SELECT scene FROM boards WHERE meeting_id = ?", meetingID)
	if err != nil || len(rows.Values) == 0 {
		return "", err
	}
	return rows.Values[0][0].(string), nil
}

// SearchNotes returns the notes whose words contain query, by meeting.
func (s *Store) SearchNotes(ctx context.Context, query string) (map[string]string, error) {
	rows, err := s.db.Query(ctx, `SELECT meeting_id, data FROM notes WHERE text LIKE ? ESCAPE '\' LIMIT 50`, like(query))
	if err != nil {
		return nil, err
	}
	found := make(map[string]string, len(rows.Values))
	for _, row := range rows.Values {
		found[row[0].(string)] = row[1].(string)
	}
	return found, nil
}

// ReplaceTranscript stores what processing a meeting found, in place of
// what was transcribed while it was recorded.
func (s *Store) ReplaceTranscript(ctx context.Context, id string, utterances []Utterance, speakers []Speaker, audio string) error {
	batch := []sqlite.Statement{
		{SQL: "DELETE FROM utterances WHERE meeting_id = ?", Args: []any{id}},
		{SQL: "DELETE FROM speakers WHERE meeting_id = ?", Args: []any{id}},
	}
	for _, sp := range speakers {
		batch = append(batch, sqlite.Statement{
			SQL:  "INSERT INTO speakers (meeting_id, id, name, self) VALUES (?, ?, ?, ?)",
			Args: []any{id, sp.ID, sp.Name, sp.Self},
		})
	}
	for _, u := range utterances {
		batch = append(batch, sqlite.Statement{
			SQL: `INSERT INTO utterances (meeting_id, seq, track, start_ms, end_ms, text, speaker, uncertain)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			Args: []any{id, u.Seq, string(u.Track), u.StartMs, u.EndMs, u.Text, u.Speaker, u.Uncertain},
		})
	}
	batch = append(batch, sqlite.Statement{
		SQL:  "UPDATE meetings SET audio = ? WHERE id = ?",
		Args: []any{audio, id},
	})
	_, err := s.db.Transaction(ctx, batch)
	return err
}

// Speakers returns who talks in a meeting, with for how long.
func (s *Store) Speakers(ctx context.Context, meetingID string) ([]Speaker, error) {
	rows, err := s.db.Query(ctx, `SELECT s.id, s.name, s.self,
		COALESCE((SELECT SUM(end_ms - start_ms) FROM utterances u WHERE u.meeting_id = s.meeting_id AND u.speaker = s.id), 0)
		FROM speakers s WHERE s.meeting_id = ? ORDER BY s.id`, meetingID)
	if err != nil {
		return nil, err
	}
	speakers := make([]Speaker, 0, len(rows.Values))
	for _, row := range rows.Values {
		speakers = append(speakers, Speaker{
			ID: int(row[0].(int64)), Name: row[1].(string), Self: row[2].(int64) != 0, DurationMs: row[3].(int64),
		})
	}
	return speakers, nil
}

func (s *Store) RenameSpeaker(ctx context.Context, meetingID string, speaker int, name string) error {
	return s.changed(s.db.Execute(ctx, "UPDATE speakers SET name = ? WHERE meeting_id = ? AND id = ?",
		name, meetingID, speaker))
}

// MergeSpeakers gives what one speaker says to another and removes the
// first.
func (s *Store) MergeSpeakers(ctx context.Context, meetingID string, from, into int) error {
	_, err := s.db.Transaction(ctx, []sqlite.Statement{
		{SQL: "UPDATE utterances SET speaker = ? WHERE meeting_id = ? AND speaker = ?", Args: []any{into, meetingID, from}},
		{SQL: "DELETE FROM speakers WHERE meeting_id = ? AND id = ?", Args: []any{meetingID, from}},
	})
	return err
}

// Line is an utterance found by Search.
type Line struct {
	MeetingID string
	Utterance Utterance
	// Speaker is the name of who says it, or "" before the meeting is
	// processed.
	Speaker string
	// Self tells that the user says it.
	Self bool
}

var likeEscaper = strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_")

func like(query string) string { return "%" + likeEscaper.Replace(query) + "%" }

// SearchTitles returns the meetings whose titles contain query.
func (s *Store) SearchTitles(ctx context.Context, query string) ([]Meeting, error) {
	rows, err := s.db.Query(ctx, "SELECT "+meetingColumns+` FROM meetings
		WHERE title LIKE ? ESCAPE '\' ORDER BY started_at DESC LIMIT 50`, like(query))
	if err != nil {
		return nil, err
	}
	meetings := make([]Meeting, 0, len(rows.Values))
	for _, row := range rows.Values {
		meetings = append(meetings, meeting(row))
	}
	return meetings, nil
}

// SearchLines returns the utterances that contain query, those of the
// newest meetings first.
func (s *Store) SearchLines(ctx context.Context, query string) ([]Line, error) {
	rows, err := s.db.Query(ctx, `SELECT u.meeting_id, u.seq, u.track, u.start_ms, u.end_ms, u.text, u.speaker, u.uncertain,
		COALESCE(s.name, ''), COALESCE(s.self, u.track = 'mic')
		FROM utterances u
		JOIN meetings m ON m.id = u.meeting_id
		LEFT JOIN speakers s ON s.meeting_id = u.meeting_id AND s.id = u.speaker
		WHERE u.text LIKE ? ESCAPE '\' ORDER BY m.started_at DESC, u.start_ms LIMIT 500`, like(query))
	if err != nil {
		return nil, err
	}
	lines := make([]Line, 0, len(rows.Values))
	for _, row := range rows.Values {
		lines = append(lines, Line{
			MeetingID: row[0].(string), Utterance: utterance(row[1:]), Speaker: row[8].(string), Self: row[9].(int64) != 0,
		})
	}
	return lines, nil
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

const meetingColumns = "id, title, started_at, duration_ms, status, directory, microphone, system, apps, error, audio"

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
		Error:      row[9].(string),
		Audio:      row[10].(string),
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
	rows, err := s.db.Query(ctx, `SELECT seq, track, start_ms, end_ms, text, speaker, uncertain FROM utterances
		WHERE meeting_id = ? ORDER BY start_ms, seq`, meetingID)
	if err != nil {
		return nil, err
	}
	utterances := make([]Utterance, 0, len(rows.Values))
	for _, row := range rows.Values {
		utterances = append(utterances, utterance(row))
	}
	return utterances, nil
}

func utterance(row []any) Utterance {
	return Utterance{
		Seq:       int(row[0].(int64)),
		Track:     Track(row[1].(string)),
		StartMs:   row[2].(int64),
		EndMs:     row[3].(int64),
		Text:      row[4].(string),
		Speaker:   int(row[5].(int64)),
		Uncertain: row[6].(int64) != 0,
	}
}
