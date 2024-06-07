package server

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/metallust/dosssh/connector"
)

const (
	host = "localhost"
	port = "23234"
)

func teaHandler(InitialModel func(string, *connector.Connector) tea.Model) func(ssh.Session) (tea.Model, []tea.ProgramOption) {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
        user, cpair := InitUser(s)
        //init model and return to ssh server
		m := InitialModel(user, cpair)
		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
}

func StartServer(initModel func(user string, conn *connector.Connector) tea.Model) {
	s, err := wish.NewServer(wish.WithAddress(net.JoinHostPort(host, port)), wish.WithHostKeyPath(".ssh/id_ed25519"), wish.WithMiddleware(
		bubbletea.Middleware(teaHandler(initModel)),
		activeterm.Middleware(),
		logging.Middleware(),
        //middleware to handle user exit
        ExitMiddleware,
		))

	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

