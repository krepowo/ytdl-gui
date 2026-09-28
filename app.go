package main

import (
	"context"
)

// App is the Wails-bound application struct. It is the single entry point the
// frontend talks to; it delegates to the internal modules (settings, engine,
// queue) which are wired up in later tasks.
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved so we can call
// runtime methods (dialogs, events) later.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
