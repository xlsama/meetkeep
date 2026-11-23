package control

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/xlsama/meetkeep/internal/store"
)

func TestSend(t *testing.T) {
	// A socket's path has to be short, which that of t.TempDir is not.
	home, err := os.MkdirTemp("", "mk")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("MEETKEEP_HOME", home)
	ctx := context.Background()

	if _, err := Send(ctx, Request{Import: "/a.wav"}); !errors.Is(err, ErrNotRunning) {
		t.Errorf("with no app it says %v", err)
	}
	serve := func() func() {
		listener, err := Listen(func(_ context.Context, r Request) (Answer, error) {
			switch {
			case r.Progress != "":
				return Answer{Working: true, Step: "transcribe", Percent: 40}, nil
			case r.Import == "/a.pdf":
				return Answer{}, errors.New("only audio and video files can be imported")
			}
			return Answer{Meeting: store.Meeting{ID: "m", Title: r.Import, Status: store.Processing}}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return func() { listener.Close() }
	}
	stop := serve()
	if got, err := Send(ctx, Request{Import: "/a.wav"}); err != nil || got.Meeting.ID != "m" || got.Meeting.Title != "/a.wav" || got.Meeting.Status != store.Processing {
		t.Errorf("got %+v (%v)", got, err)
	}
	if got, err := Send(ctx, Request{Progress: "m"}); err != nil || !got.Working || got.Step != "transcribe" || got.Percent != 40 {
		t.Errorf("got %+v (%v)", got, err)
	}
	if _, err := Send(ctx, Request{Import: "/a.pdf"}); err == nil || err.Error() != "only audio and video files can be imported" {
		t.Errorf("the app's refusal reads %v", err)
	}
	stop()
	if _, err := Send(ctx, Request{Import: "/a.wav"}); !errors.Is(err, ErrNotRunning) {
		t.Errorf("once the app quit it says %v", err)
	}

	// An app that crashed leaves its socket behind.
	if err := os.WriteFile(path(), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	defer serve()()
	if _, err := Send(ctx, Request{Import: "/a.wav"}); err != nil {
		t.Errorf("the next app does not take the socket over: %v", err)
	}
}
