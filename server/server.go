package server

import (
	"math/rand"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/metallust/sshGameClient/connector"
)

type User struct {
	connection *connector.Connector
    OpponentConn *connector.Connector
    Opponent   string
	stage      string
}

var (
    Users map[string]User = make(map[string]User)
    UserMut sync.Mutex
)

func CreateGame(user string, msg connector.Msg) {
	//check if the user is in the lobby
    UserMut.Lock()
	if Users[user].stage != "lobby" {
		msg.Reply(connector.ERRORMSG, "My guy you are not in the lobby - (can be mutex issue)", false)
	}

	//add user to ready
	u := Users[user]
	u.stage = "ready"
	Users[user] = u
    UserMut.Unlock()
	msg.Reply(connector.OKMSG, nil, false)
    log.Info("CREATE: user added to ready", "user", user)
}

func ReturnToLobby(user string, msg connector.Msg){
	//check if the user is in the lobby
    UserMut.Lock()
	//add user to read
	Users[user] = User{
        connection: Users[user].connection,
        stage: "lobby",
    }
    UserMut.Unlock()
	msg.Reply(connector.OKMSG, nil, false)
    log.Info("RETURNTOLOBY: user added to lobby", "user", user)
}

func JoinGame(user string, msg connector.Msg) {
    UserMut.Lock()
	opponent := msg.Data.(string)
	//if  not in ready return error
	if Users[user].stage != "lobby" {
        log.Error("User is not in lobby", "user", user, "stage", Users[user].stage)
		msg.Reply(connector.ERRORMSG, "You are not allow your stage should be Inital", false)
		return
	}
	if Users[opponent].stage != "ready" {
        log.Error("Opponent is not ready", "opponent", opponent, "stage", Users[opponent].stage)
		msg.Reply(connector.ERRORMSG, "Opponent is no longer avaiable ..(maybe be entered a different game or went offline)", false)
		return
	}

	//decide turn
	playerA, playerB := randomPlayer()
    oppconn := connector.NewConnector()
	oppreqdata := map[string]interface{}{
		"opponent": user,
		"turn":     playerA,
        "opponentconnector": oppconn,
	}

	//send opponent join message with name of user
	oppreply := Users[opponent].connection.SendMsg(connector.JOINREQMSG, oppreqdata, true)
	oppreplymsg := <-oppreply
	if oppreplymsg.Name != connector.OKMSG {
		msg.Reply(connector.ERRORMSG, oppreplymsg.Data, false)
		return
	}

    
	//replying user
    data := map[string]interface{}{
		"opponent": opponent,
		"turn":     playerB,
        "opponentconnector": connector.CreateConnectorPair(oppconn),
	}
	msg.Reply(connector.OKMSG, data, false)

	u := Users[user]
	u.Opponent = opponent
	u.stage = "ingame"
	Users[user] = u

	o := Users[opponent]
	o.Opponent = user
	o.stage = "ingame"
	Users[opponent] = o

    UserMut.Unlock()
    log.Info("JOIN: Game started", "user", user, "opponent", opponent)
}

func ListGames(user string, msg connector.Msg) {
	data := make([]string, 0)
    UserMut.Lock()
	for k, v := range Users {
		if k != user && v.stage == "ready" {
			data = append(data, k)
		}
	}
    UserMut.Unlock()
	msg.Reply(connector.LISTMSG, data, false)
    log.Info("LIST: list sent", "list", data)
}

func ExitGame(user string) {

    UserMut.Lock()
    Users[user].connection.Close()
	// if game is in progress
	opponent := Users[user].Opponent
	if opponent != "" {
		// send opponent the opponent abort msg
		// send opponent to the lobby
		o := Users[opponent]
		o.Opponent = ""
		o.stage = "lobby"
		Users[Users[user].Opponent] = o
        Users[opponent].connection.SendMsg(connector.ERRORMSG, "Opponent disconnected ...", false)
	}
	// remove user from Users list
    log.Info("Send opponent abort msg --<")
	delete(Users, user)
    UserMut.Unlock()
    log.Info("EXIT: User removed", "user", user, "Users", Users)
}

func randomPlayer() (string, string) {
	// randomly decide who is going first
	if randbool := rand.Intn(2) == 0; randbool {
		return "first", "second"
	}
	return "second", "first"
}
