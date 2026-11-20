package main

import (
	"log"

	"github.com/egoist/mygo"
)

func main() {
	mygo.Bind(newSession())

	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:           "MeetKeep",
			Width:           720,
			Height:          640,
			MinWidth:        420,
			MinHeight:       360,
			BackgroundColor: "light-dark(#ffffff, #111113)",
			StateKey:        "main",
			URL:             "/",
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
