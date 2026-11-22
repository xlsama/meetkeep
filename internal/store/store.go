// Package store keeps meetings and what was said in them.
package store

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
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
	// Video names the file in Directory that holds the picture of an
	// imported video, or is "".
	Video string `json:"video"`
}

// Speaker is someone who talks in a meeting.
type Speaker struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Self marks the user.
	Self       bool  `json:"self"`
	DurationMs int64 `json:"durationMs"`
	// PersonID is the person whose voice this is, or 0.
	PersonID int `json:"personId"`
	// Remembered tells that the voice heard here is kept as a sample of
	// that person's.
	Remembered bool `json:"remembered"`
	// SampleStartMs and SampleEndMs bound a stretch of the meeting in which
	// the speaker is heard alone, or are both 0.
	SampleStartMs int64 `json:"sampleStartMs"`
	SampleEndMs   int64 `json:"sampleEndMs"`
	// Voice is the speaker's voiceprint in this meeting, or empty.
	Voice []float32 `json:"-"`
}

// Person is someone whose voice is remembered.
type Person struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Meetings is how many meetings the person talks in, LastAt when the
	// newest of them began, in Unix milliseconds.
	Meetings int   `json:"meetings"`
	LastAt   int64 `json:"lastAt"`
	// Samples is how many samples of the voice are kept.
	Samples int `json:"samples"`
	// SampleMeeting, SampleStartMs and SampleEndMs tell where to hear the
	// voice; SampleMeeting is "" when no recording of it is left.
	SampleMeeting string `json:"sampleMeeting"`
	SampleStartMs int64  `json:"sampleStartMs"`
	SampleEndMs   int64  `json:"sampleEndMs"`
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

// Step is what one step of processing a meeting took.
type Step struct {
	// Run is when the processing the step belongs to was asked for, in
	// Unix milliseconds: the steps of one processing share it.
	Run int64 `json:"run"`
	// Kind is what was asked for: process, import, speakers or notes.
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	StartedAt  int64  `json:"startedAt"`
	DurationMs int64  `json:"durationMs"`
	// MemoryPeak is the most memory the app held during the step, in
	// bytes, or 0 when it was not measured.
	MemoryPeak int64 `json:"memoryPeak"`
	// Error tells why the step failed, or is "".
	Error string `json:"error"`
	// Detail is a JSON object of what the step worked on and made.
	Detail string `json:"detail"`
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
	{
		`CREATE TABLE people (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE voiceprints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			person_id INTEGER NOT NULL REFERENCES people (id) ON DELETE CASCADE,
			meeting_id TEXT REFERENCES meetings (id) ON DELETE SET NULL,
			speaker INTEGER NOT NULL,
			embedding BLOB NOT NULL,
			start_ms INTEGER NOT NULL,
			end_ms INTEGER NOT NULL
		)`,
		`ALTER TABLE speakers ADD COLUMN person_id INTEGER REFERENCES people (id) ON DELETE SET NULL`,
		`ALTER TABLE speakers ADD COLUMN embedding BLOB`,
		`ALTER TABLE speakers ADD COLUMN sample_start INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE speakers ADD COLUMN sample_end INTEGER NOT NULL DEFAULT 0`,
	},
	{
		`ALTER TABLE meetings ADD COLUMN video TEXT NOT NULL DEFAULT ''`,
	},
	{
		`ALTER TABLE notes ADD COLUMN doc TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE notes ADD COLUMN previous_data TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE notes ADD COLUMN previous_doc TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE notes ADD COLUMN previous_text TEXT NOT NULL DEFAULT ''`,
	},
	{
		`CREATE TABLE steps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			meeting_id TEXT NOT NULL REFERENCES meetings (id) ON DELETE CASCADE,
			run INTEGER NOT NULL,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			started_at INTEGER NOT NULL,
			duration_ms INTEGER NOT NULL,
			memory_peak INTEGER NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '',
			detail TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE INDEX steps_meeting ON steps (meeting_id, id)`,
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
		(id, title, started_at, duration_ms, status, directory, microphone, system, apps, video)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.Title, m.StartedAt, m.DurationMs, string(m.Status), m.Directory, m.Microphone, m.System, string(apps), m.Video)
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

// StoredNotes are a meeting's notes as they are kept.
type StoredNotes struct {
	// Data is what SaveNotes took, "" when the meeting has no notes.
	Data string
	// Doc is the document the user made of the notes by editing them, or
	// "", and Text the words to search: the document's, or the notes'.
	Doc  string
	Text string
	// Error tells why the last attempt to write the notes failed.
	Error string
	// Replaced tells that the notes written before these are kept.
	Replaced bool
}

// Notes returns a meeting's notes.
func (s *Store) Notes(ctx context.Context, meetingID string) (StoredNotes, error) {
	rows, err := s.db.Query(ctx, `SELECT data, doc, text, error, previous_data <> '' FROM notes
		WHERE meeting_id = ?`, meetingID)
	if err != nil || len(rows.Values) == 0 {
		return StoredNotes{}, err
	}
	row := rows.Values[0]
	return StoredNotes{
		Data: row[0].(string), Doc: row[1].(string), Text: row[2].(string), Error: row[3].(string),
		Replaced: row[4].(int64) != 0,
	}, nil
}

// SaveNotesDoc stores the document the user made of a meeting's notes,
// and its words, to search.
func (s *Store) SaveNotesDoc(ctx context.Context, meetingID, doc, text string) error {
	return s.changed(s.db.Execute(ctx, "UPDATE notes SET doc = ?, text = ? WHERE meeting_id = ?", doc, text, meetingID))
}

// ReplaceNotes stores notes written anew for a meeting and keeps those
// they replace, with what the user made of them, for RestoreNotes.
func (s *Store) ReplaceNotes(ctx context.Context, meetingID, data, text string) error {
	return s.changed(s.db.Execute(ctx, `UPDATE notes SET previous_data = data, previous_doc = doc,
		previous_text = text, data = ?, doc = '', text = ?, error = '' WHERE meeting_id = ?`, data, text, meetingID))
}

// RestoreNotes brings back the notes that ReplaceNotes replaced, and
// keeps those it replaces in their place.
func (s *Store) RestoreNotes(ctx context.Context, meetingID string) error {
	return s.changed(s.db.Execute(ctx, `UPDATE notes SET
		data = previous_data, doc = previous_doc, text = previous_text,
		previous_data = data, previous_doc = doc, previous_text = text
		WHERE meeting_id = ? AND previous_data <> ''`, meetingID))
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
func (s *Store) SearchNotes(ctx context.Context, query string) (map[string]StoredNotes, error) {
	rows, err := s.db.Query(ctx, `SELECT meeting_id, data, doc, text FROM notes
		WHERE text LIKE ? ESCAPE '\' LIMIT 50`, like(query))
	if err != nil {
		return nil, err
	}
	found := make(map[string]StoredNotes, len(rows.Values))
	for _, row := range rows.Values {
		found[row[0].(string)] = StoredNotes{Data: row[1].(string), Doc: row[2].(string), Text: row[3].(string)}
	}
	return found, nil
}

// ReplaceTranscript stores what processing a meeting found, in place of
// what was transcribed while it was recorded.
func (s *Store) ReplaceTranscript(ctx context.Context, id string, utterances []Utterance, speakers []Speaker, audio string) error {
	batch := []sqlite.Statement{
		{SQL: "DELETE FROM utterances WHERE meeting_id = ?", Args: []any{id}},
		{SQL: "DELETE FROM speakers WHERE meeting_id = ?", Args: []any{id}},
		{SQL: "UPDATE voiceprints SET speaker = 0 WHERE meeting_id = ?", Args: []any{id}},
	}
	for _, sp := range speakers {
		var person any
		if sp.PersonID != 0 {
			person = sp.PersonID
			batch = append(batch, sqlite.Statement{
				SQL:  "UPDATE voiceprints SET speaker = ? WHERE meeting_id = ? AND person_id = ? AND speaker = 0",
				Args: []any{sp.ID, id, sp.PersonID},
			})
		}
		batch = append(batch, sqlite.Statement{
			SQL: `INSERT INTO speakers (meeting_id, id, name, self, person_id, embedding, sample_start, sample_end)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			Args: []any{id, sp.ID, sp.Name, sp.Self, person, encode(sp.Voice), sp.SampleStartMs, sp.SampleEndMs},
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
		COALESCE((SELECT SUM(end_ms - start_ms) FROM utterances u WHERE u.meeting_id = s.meeting_id AND u.speaker = s.id), 0),
		COALESCE(s.person_id, 0), s.embedding, s.sample_start, s.sample_end,
		EXISTS (SELECT 1 FROM voiceprints v WHERE v.meeting_id = s.meeting_id AND v.speaker = s.id)
		FROM speakers s WHERE s.meeting_id = ? ORDER BY s.id`, meetingID)
	if err != nil {
		return nil, err
	}
	speakers := make([]Speaker, 0, len(rows.Values))
	for _, row := range rows.Values {
		voice, _ := row[5].([]byte)
		speakers = append(speakers, Speaker{
			ID: int(row[0].(int64)), Name: row[1].(string), Self: row[2].(int64) != 0, DurationMs: row[3].(int64),
			PersonID: int(row[4].(int64)), Voice: decode(voice),
			SampleStartMs: row[6].(int64), SampleEndMs: row[7].(int64), Remembered: row[8].(int64) != 0,
		})
	}
	return speakers, nil
}

func encode(vector []float32) []byte {
	if len(vector) == 0 {
		return nil
	}
	out := make([]byte, 4*len(vector))
	for i, v := range vector {
		binary.LittleEndian.PutUint32(out[4*i:], math.Float32bits(v))
	}
	return out
}

func decode(data []byte) []float32 {
	vector := make([]float32, len(data)/4)
	for i := range vector {
		vector[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[4*i:]))
	}
	return vector
}

// ErrNoVoice means the speaker has no voiceprint to remember.
var ErrNoVoice = errors.New("这个说话人还没有可以记住的声音")

// Remember keeps the voice of a meeting's speaker as a sample of the
// person of the speaker's name, whom it makes if need be.
func (s *Store) Remember(ctx context.Context, meetingID string, speaker int) error {
	rows, err := s.db.Query(ctx, `SELECT name, embedding, sample_start, sample_end FROM speakers
		WHERE meeting_id = ? AND id = ?`, meetingID, speaker)
	if err != nil {
		return err
	}
	if len(rows.Values) == 0 {
		return ErrNotFound
	}
	row := rows.Values[0]
	voice, _ := row[1].([]byte)
	if len(voice) == 0 {
		return ErrNoVoice
	}
	name := row[0].(string)
	people, err := s.db.Query(ctx, "SELECT id FROM people WHERE name = ? ORDER BY id LIMIT 1", name)
	if err != nil {
		return err
	}
	var person int64
	if len(people.Values) > 0 {
		person = people.Values[0][0].(int64)
	} else {
		made, err := s.db.Execute(ctx, "INSERT INTO people (name, created_at) VALUES (?, ?)", name, time.Now().UnixMilli())
		if err != nil {
			return err
		}
		person = made.LastInsertID
	}
	_, err = s.db.Transaction(ctx, []sqlite.Statement{
		{SQL: "DELETE FROM voiceprints WHERE meeting_id = ? AND speaker = ?", Args: []any{meetingID, speaker}},
		{
			SQL:  "INSERT INTO voiceprints (person_id, meeting_id, speaker, embedding, start_ms, end_ms) VALUES (?, ?, ?, ?, ?, ?)",
			Args: []any{person, meetingID, speaker, voice, row[2], row[3]},
		},
		{SQL: "UPDATE speakers SET person_id = ? WHERE meeting_id = ? AND id = ?", Args: []any{person, meetingID, speaker}},
	})
	return err
}

// Forget drops the sample kept of a meeting's speaker, and the person it
// was the last sample of.
func (s *Store) Forget(ctx context.Context, meetingID string, speaker int) error {
	_, err := s.db.Transaction(ctx, []sqlite.Statement{
		{SQL: "DELETE FROM voiceprints WHERE meeting_id = ? AND speaker = ?", Args: []any{meetingID, speaker}},
		{SQL: "DELETE FROM people WHERE NOT EXISTS (SELECT 1 FROM voiceprints v WHERE v.person_id = people.id)"},
	})
	return err
}

// People returns those whose voices are remembered, by name.
func (s *Store) People(ctx context.Context) ([]Person, error) {
	rows, err := s.db.Query(ctx, `SELECT p.id, p.name,
		(SELECT COUNT(DISTINCT s.meeting_id) FROM speakers s WHERE s.person_id = p.id),
		COALESCE((SELECT MAX(m.started_at) FROM speakers s JOIN meetings m ON m.id = s.meeting_id WHERE s.person_id = p.id), 0),
		(SELECT COUNT(*) FROM voiceprints v WHERE v.person_id = p.id),
		COALESCE((SELECT v.meeting_id || ':' || v.start_ms || ':' || v.end_ms FROM voiceprints v
			WHERE v.person_id = p.id AND v.meeting_id IS NOT NULL ORDER BY v.id DESC LIMIT 1), '')
		FROM people p ORDER BY p.name`)
	if err != nil {
		return nil, err
	}
	people := make([]Person, 0, len(rows.Values))
	for _, row := range rows.Values {
		p := Person{
			ID: int(row[0].(int64)), Name: row[1].(string), Meetings: int(row[2].(int64)), LastAt: row[3].(int64),
			Samples: int(row[4].(int64)),
		}
		if parts := strings.Split(row[5].(string), ":"); len(parts) == 3 {
			p.SampleMeeting = parts[0]
			p.SampleStartMs, _ = strconv.ParseInt(parts[1], 10, 64)
			p.SampleEndMs, _ = strconv.ParseInt(parts[2], 10, 64)
		}
		people = append(people, p)
	}
	return people, nil
}

// Voices returns the samples kept of each person's voice.
func (s *Store) Voices(ctx context.Context) (map[int][][]float32, map[int]string, error) {
	rows, err := s.db.Query(ctx, `SELECT v.person_id, p.name, v.embedding FROM voiceprints v
		JOIN people p ON p.id = v.person_id`)
	if err != nil {
		return nil, nil, err
	}
	voices, names := map[int][][]float32{}, map[int]string{}
	for _, row := range rows.Values {
		id := int(row[0].(int64))
		names[id] = row[1].(string)
		voices[id] = append(voices[id], decode(row[2].([]byte)))
	}
	return voices, names, nil
}

// RenamePerson gives a person another name, in every meeting they talk in.
func (s *Store) RenamePerson(ctx context.Context, id int, name string) error {
	_, err := s.db.Transaction(ctx, []sqlite.Statement{
		{SQL: "UPDATE people SET name = ? WHERE id = ?", Args: []any{name, id}},
		{SQL: "UPDATE speakers SET name = ? WHERE person_id = ?", Args: []any{name, id}},
	})
	return err
}

// DeletePerson forgets a person's voice. The meetings keep their name.
func (s *Store) DeletePerson(ctx context.Context, id int) error {
	return s.changed(s.db.Execute(ctx, "DELETE FROM people WHERE id = ?", id))
}

// Self returns in how many meetings the user's microphone was recorded,
// and when the newest of them began.
func (s *Store) Self(ctx context.Context) (meetings int, lastAt int64, err error) {
	rows, err := s.db.Query(ctx, "SELECT COUNT(*), COALESCE(MAX(started_at), 0) FROM meetings WHERE microphone = 1")
	if err != nil {
		return 0, 0, err
	}
	return int(rows.Values[0][0].(int64)), rows.Values[0][1].(int64), nil
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

const meetingColumns = "id, title, started_at, duration_ms, status, directory, microphone, system, apps, error, audio, video"

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

// Glance is what a meeting shows of itself in a list.
type Glance struct {
	// Notes is what SaveNotes took, or "".
	Notes string
	// Opening is how the meeting begins.
	Opening  string
	Speakers []Speaker
}

// opening is how many utterances a meeting's opening is read from.
const opening = 8

// Glances returns a glance at every meeting that has something to show.
func (s *Store) Glances(ctx context.Context) (map[string]Glance, error) {
	glances := map[string]Glance{}
	rows, err := s.db.Query(ctx, "SELECT meeting_id, data FROM notes WHERE data != ''")
	if err != nil {
		return nil, err
	}
	for _, row := range rows.Values {
		g := glances[row[0].(string)]
		g.Notes = row[1].(string)
		glances[row[0].(string)] = g
	}
	rows, err = s.db.Query(ctx, `SELECT meeting_id, group_concat(text, '') FROM
		(SELECT meeting_id, text FROM utterances WHERE seq <= ? ORDER BY meeting_id, seq) GROUP BY meeting_id`, opening)
	if err != nil {
		return nil, err
	}
	for _, row := range rows.Values {
		g := glances[row[0].(string)]
		g.Opening, _ = row[1].(string)
		glances[row[0].(string)] = g
	}
	rows, err = s.db.Query(ctx, "SELECT meeting_id, id, name, self, COALESCE(person_id, 0) FROM speakers ORDER BY meeting_id, id")
	if err != nil {
		return nil, err
	}
	for _, row := range rows.Values {
		g := glances[row[0].(string)]
		g.Speakers = append(g.Speakers, Speaker{
			ID: int(row[1].(int64)), Name: row[2].(string), Self: row[3].(int64) != 0, PersonID: int(row[4].(int64)),
		})
		glances[row[0].(string)] = g
	}
	return glances, nil
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
		Video:      row[11].(string),
	}
	_ = json.Unmarshal([]byte(row[8].(string)), &m.Apps)
	return m
}

func (s *Store) AddStep(ctx context.Context, meetingID string, step Step) error {
	_, err := s.db.Execute(ctx, `INSERT INTO steps (meeting_id, run, kind, name, started_at, duration_ms, memory_peak, error, detail)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, meetingID, step.Run, step.Kind, step.Name, step.StartedAt, step.DurationMs,
		step.MemoryPeak, step.Error, step.Detail)
	return err
}

// Steps returns the steps of every processing of a meeting, oldest first.
func (s *Store) Steps(ctx context.Context, meetingID string) ([]Step, error) {
	rows, err := s.db.Query(ctx, `SELECT run, kind, name, started_at, duration_ms, memory_peak, error, detail FROM steps
		WHERE meeting_id = ? ORDER BY id`, meetingID)
	if err != nil {
		return nil, err
	}
	steps := make([]Step, 0, len(rows.Values))
	for _, row := range rows.Values {
		steps = append(steps, Step{
			Run: row[0].(int64), Kind: row[1].(string), Name: row[2].(string), StartedAt: row[3].(int64),
			DurationMs: row[4].(int64), MemoryPeak: row[5].(int64), Error: row[6].(string), Detail: row[7].(string),
		})
	}
	return steps, nil
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
