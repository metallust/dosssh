package server

import (
	"math/rand"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/metallust/dosssh/client"
	"github.com/metallust/dosssh/connector"
)

type User struct {
	connection   *connector.Connector
	OpponentConn *connector.Connector
	Opponent     string
	stage        string
}

var (
	Users   map[string]User = make(map[string]User)
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

func ReturnToLobby(user string, msg connector.Msg) {
	//check if the user is in the lobby
	UserMut.Lock()
	//add user to read
	Users[user] = User{
		connection: Users[user].connection,
		stage:      "lobby",
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
	oppreqdata := client.JoinBody{
		Opponent:          user,
		Turn:              playerA,
		Opponentconnector: oppconn,
	}

	//send opponent join message with name of user
	oppreply := Users[opponent].connection.SendMsg(connector.JOINREQMSG, oppreqdata, true)
	oppreplymsg := <-oppreply
	if oppreplymsg.Name != connector.OKMSG {
		msg.Reply(connector.ERRORMSG, oppreplymsg.Data, false)
		return
	}

	//replying user
	data := client.JoinBody{
		Opponent:          opponent,
		Turn:              playerB,
		Opponentconnector: connector.CreateConnectorPair(oppconn),
	}
	msg.Reply(connector.OKMSG, data, false)

	u := Users[user]
	u.Opponent = opponent
	u.stage = "ingame"
	u.OpponentConn = data.Opponentconnector
	Users[user] = u

	o := Users[opponent]
	o.Opponent = user
	o.stage = "ingame"
	o.OpponentConn = oppreqdata.Opponentconnector
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

func randomPlayer() (string, string) {
	// randomly decide who is going first
	if randbool := rand.Intn(2) == 0; randbool {
		return "first", "second"
	}
	return "second", "first"
}

func ListenClient(user string, c *connector.Connector) {
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
}

func ExitMiddleware(next ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		next(s)
		user := getUser(s)
        log.Info("EXITMIDDLEWARE :Turning on all the channels", "user", user)
		UserMut.Lock()
        //close connector
		Users[user].connection.Close()

        //check if in game
		opponent := Users[user].Opponent
		if opponent != "" {
			o := Users[opponent]

            //clean opponent and add opponent back to lobby
			o.Opponent = ""
			o.stage = "lobby"
			o.OpponentConn.Close()
			o.OpponentConn = nil
			Users[opponent] = o

            //send error msg to client
			Users[opponent].connection.SendMsg(connector.ERRORMSG, "Opponent disconnected ...", false)
		}

        //delete user from []User
		delete(Users, user)
		UserMut.Unlock()
		log.Info("EXITMIDDLEWARE: SUCCESSFULL User removed", "user", user, "Users", Users)
	}
}

func InitUser(s ssh.Session) (string, *connector.Connector) {

	UserMut.Lock()
	user := getUser(s)
	//create user
	c := connector.NewConnector()
	cpair := connector.CreateConnectorPair(c)
	Users[user] = User{
		connection: c,
		stage:      "lobby",
	}
	UserMut.Unlock()
	//listen to client
	go ListenClient(user, c)
	return user, cpair
}

func getUser(s ssh.Session) string {
	blob := s.Context().SessionID()
	user := s.User() + blob[:5]
	return user
}
