package client

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/metallust/dosssh/connector"
)

type GameClientOpponentMsg struct {
	Msg  int
	Data interface{}
}

type GameClientMsg struct {
	Msg  int
	Data interface{}
}

const (
	JOINREQMSG = iota
	MOVEMSG
	ERRORMSG
	UNKNOWMSG
)

type DoneMsg struct {
	Msg  string
	Data interface{}
}

type GameClient struct {
	serverconnector   *connector.Connector
	opponentConnector *connector.Connector
	user              string
	joinrequestmsg    connector.Msg
}

func NewGameClient(c *connector.Connector, u string) *GameClient {
	return &GameClient{
		serverconnector: c,
		user:            u,
	}
}

// this function return as tea msg which is gameclientmsg which contain name and data
// name is set to the donemsg given by the caller
// working: this will send "list" message to the server and wait for the reply
// the server if successfull will return "ok"
// if there is any error server will return "error" and the data will be the error message which will be forwarded to
// the caller will msg name set to "error" and data set to the error message
func (gc *GameClient) List(doneMsg string) tea.Cmd {
	return func() tea.Msg {
		replychan := gc.serverconnector.SendMsg(connector.LISTMSG, nil, true)
		msg := <-replychan
		if msg.Name == connector.ERRORMSG {
			//handle that error
			log.Fatal("Error in list ... Here is the msg : ", msg)
			return DoneMsg{Msg: "error", Data: msg.Data}
		}
		return DoneMsg{Msg: doneMsg, Data: msg.Data.([]string)}
	}
}

// this function return as tea msg which is gameclientmsg which contain name and data
// name is set to the donemsg given byc the caller
// working: this will send "create" message to the server and wait for the reply
// the server if successfull will return "ok"
// if there is any error server will return "error" and the data will be the error message which will be forwarded to
// the caller will msg name set to "error" and data set to the error message
func (gc *GameClient) Create(doneMsg string) tea.Cmd {
	return func() tea.Msg {
		replychan := gc.serverconnector.SendMsg(connector.CREATEMSG, nil, true)
		msg, _ := <-replychan
		if msg.Name != connector.OKMSG {
			log.Fatal("Error in Create ... Here is the msg : ", msg)
			return DoneMsg{Msg: "error", Data: msg.Data}
		}
		return DoneMsg{Msg: doneMsg}
	}
}

// this function return as tea msg which is gameclientmsg which contain name and data
// name is set to the donemsg given by the caller
// working: this will send "move" message and move array to the server and wait for the reply
// the server if successfull will return "ok"
// if there is any error server will return "error" and the data will be the error message which will be forwarded to
// the caller will msg name set to "error" and data set to the error message
func (gc *GameClient) Move(move [2]int, doneMsg string) tea.Cmd {
	return func() tea.Msg {
		replychan := gc.opponentConnector.SendMsg(connector.MOVEMSG, move, true)
		msg, _ := <-replychan
		if msg.Name != connector.OKMSG {
			//handle that error
			return GameClientMsg{Msg: ERRORMSG, Data: msg.Data}
		}
		return DoneMsg{Msg: doneMsg}
	}
}

type JoinBody struct {
	Opponent          string
	Turn              string
	Opponentconnector *connector.Connector
}

// this function return as tea msg which is gameclientmsg which contain name and data
// name is set to the donemsg given by the caller
// working: this will send "join" message and opponent name string to the server and wait for the reply
// the server if successfull will return "ok" and the data will be either "first" or "second"
// if there is any error server will return "error" and the data will be the error message which will be forwarded to
// the caller will msg name set to "error" and data set to the error message
func (gc *GameClient) Join(opponent, doneMsg string) tea.Cmd {
	return func() tea.Msg {
		replychan := gc.serverconnector.SendMsg(connector.JOINREQMSG, opponent, true)
		msg, _ := <-replychan
		if msg.Name != connector.OKMSG {
			//handle that error
			log.Fatal("Error in Join ... Here is the msg : ", msg)
			return DoneMsg{Msg: "error", Data: msg.Data}
		}
		gc.opponentConnector = msg.Data.(JoinBody).Opponentconnector
		return DoneMsg{Msg: doneMsg, Data: []string{opponent, msg.Data.(JoinBody).Turn}}
	}
}

func (gc *GameClient) AcceptRequest(accept bool, donemsg string) tea.Cmd {
	return func() tea.Msg {
		if gc.joinrequestmsg.Name != connector.JOINREQMSG {
			return DoneMsg{Msg: "error", Data: "No join request found"}
		}
		if !accept {
			gc.joinrequestmsg.Reply(connector.ERRORMSG, nil, false)
			gc.joinrequestmsg = connector.Msg{}
			return DoneMsg{Msg: donemsg}
		}
		gc.joinrequestmsg.Reply(connector.OKMSG, nil, false)
		data := gc.joinrequestmsg.Data.(JoinBody)
		gc.opponentConnector = data.Opponentconnector
		return DoneMsg{Msg: donemsg, Data: []string{data.Opponent, data.Turn}}
	}
}

func (gc *GameClient) ListenServer() tea.Cmd {
	return func() tea.Msg {
		msg, more := gc.serverconnector.GetMsg()
		log.Println(gc.user, "LISTENSERVER :", msg, more)
		if !more {
			log.Println("Bubbletea Application: Server disconnected ...")
			return GameClientMsg{ERRORMSG, "server connection closed"}
		}
		switch msg.Name {
		case connector.JOINREQMSG:
			gc.joinrequestmsg = msg
			return GameClientMsg{Msg: JOINREQMSG, Data: msg.Data.(JoinBody).Opponent}
		case connector.ERRORMSG:
			return GameClientMsg{Msg: ERRORMSG, Data: msg.Data}
		}
		return GameClientMsg{Msg: UNKNOWMSG}
	}
}

func (gc *GameClient) ListenOpponent() tea.Cmd {
	return func() tea.Msg {
		msg, more := gc.opponentConnector.GetMsg()
		log.Println(gc.user, "LISTENOPPONENT :", msg, more)
		if !more {
			log.Println("Bubbletea Application: Opponent disconnected ...")
			return nil
		}
		switch msg.Name {
		case connector.MOVEMSG:
			msg.Reply(connector.OKMSG, nil, false)
			return GameClientOpponentMsg{Msg: MOVEMSG, Data: msg.Data}
		case connector.ERRORMSG:
			return GameClientOpponentMsg{Msg: ERRORMSG, Data: msg.Data}
		}
		return GameClientOpponentMsg{Msg: UNKNOWMSG}
	}
}
