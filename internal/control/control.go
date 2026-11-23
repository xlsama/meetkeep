// Package control carries what the command line asks of the running app:
// the app listens on a socket next to its database, and answers each
// connection's request.
package control

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/store"
)

// ErrNotRunning tells that no app answers.
var ErrNotRunning = errors.New("MeetKeep is not running: open the app and try again")

// Request is what the command line asks for: one of its fields is set.
type Request struct {
	// Import is the audio or video file to make a meeting of.
	Import string `json:"import"`
	// Progress is the meeting to tell the progress of.
	Progress string `json:"progress"`
}

// Answer is what the app answers a request with.
type Answer struct {
	// Meeting is the meeting made of an imported file.
	Meeting store.Meeting `json:"meeting"`
	// Working tells that the meeting is queued or being processed, Step at
	// what, "" while it waits its turn, and Percent how far it got.
	Working bool   `json:"working"`
	Step    string `json:"step"`
	Percent int    `json:"percent"`
}

type response struct {
	Answer
	Error string `json:"error"`
}

func path() string { return filepath.Join(config.Dir(), "meetkeep.sock") }

// Listen has handle answer the requests of the command line until the
// listener it returns is closed. Only the one running app may call it: it
// takes the socket over.
func Listen(handle func(context.Context, Request) (Answer, error)) (net.Listener, error) {
	if err := os.Remove(path()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	listener, err := net.Listen("unix", path())
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path(), 0o600); err != nil {
		listener.Close()
		return nil, err
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				var request Request
				if json.NewDecoder(conn).Decode(&request) != nil {
					return
				}
				var answer response
				if answer.Answer, err = handle(context.Background(), request); err != nil {
					answer.Error = err.Error()
				}
				_ = json.NewEncoder(conn).Encode(answer)
			}()
		}
	}()
	return listener, nil
}

// Send asks the running app for what request says and returns its answer.
func Send(ctx context.Context, request Request) (Answer, error) {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", path())
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED) {
		return Answer{}, ErrNotRunning
	}
	if err != nil {
		return Answer{}, err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return Answer{}, err
	}
	var answer response
	if err := json.NewDecoder(conn).Decode(&answer); err != nil {
		if ctx.Err() != nil {
			return Answer{}, ctx.Err()
		}
		return Answer{}, err
	}
	if answer.Error != "" {
		return Answer{}, errors.New(answer.Error)
	}
	return answer.Answer, nil
}
