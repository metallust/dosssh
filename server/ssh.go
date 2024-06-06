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
		//generate random blob
		blob := s.Context().SessionID()
		user := s.User() + blob[:5]

		// NOTE : Actual user is create here
		c := connector.NewConnector()
		cpair := connector.CreateConnectorPair(c)
		Users[user] = User{
			connection: c,
			stage:      "lobby",
		}
		go func() {
			for {
				clientMsg, more := c.GetMsg()
				if more == false {
					log.Info("Channel closed stopping go routing", "user", user)
					return
				}
				switch clientMsg.Name {
				//TODO: add error handling if the function return any error forward that to client
				case connector.CREATEMSG:
					CreateGame(user, clientMsg)
				case connector.RETURNLOBBYMSG:
					ReturnToLobby(user, clientMsg)
				case connector.LISTMSG:
					ListGames(user, clientMsg)
				case connector.JOINREQMSG:
					JoinGame(user, clientMsg)
				}
			}
		}()
		m := InitialModel(user, cpair)
		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
}

func StartServer(initModel func(user string, conn *connector.Connector) tea.Model) {
	s, err := wish.NewServer(wish.WithAddress(net.JoinHostPort(host, port)), wish.WithHostKeyPath(".ssh/id_ed25519"), wish.WithMiddleware(
		bubbletea.Middleware(teaHandler(initModel)),
		activeterm.Middleware(),
		logging.Middleware(),
		func(next ssh.Handler) ssh.Handler {
			return func(s ssh.Session) {
				next(s)
				user := s.User() + s.Context().SessionID()[:5]
				log.Info("Turning on all the channels", "user", user)
				ExitGame(user)
			}
		},
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
