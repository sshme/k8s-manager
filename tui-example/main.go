package main

import (
	"log"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	listView uint = iota
	titleView
	bodyView
)

type model struct {
	store     *Store
	state     uint
	textarea  textarea.Model
	textinput textinput.Model
	currNote  Note
	notes     []Note
	listIndex int
}

func NewModel(store *Store) model {
	notes, err := store.GetNotes()
	if err != nil {
		log.Fatalf("unable to get notes: %v", err)
	}

	return model{
		store:     store,
		state:     listView,
		textarea:  textarea.New(),
		textinput: textinput.New(),
		notes:     notes,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func main() {

}
