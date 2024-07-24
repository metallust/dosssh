## DOSSSH (dos meaning 2 over ssh) 
### This is a framwork for building 2 player games over ssh

We can make TUI game application using bubble tea, which can be served via ssh using wish.
This project is suppose to help make communication between 2 game instances possible.
There are @ parts to this project the server side and the client side
1. The server side creates a new game instance when a new user join the server via ssh most of that is handled by wish
2. The client side is what you will mostly need when creating you game using bubbletea use client side function to connect and communicate with other game instance

### Server code
This will start a ssh server
```go
package main

import (
	"github.com/metallust/chessssh/pkg/tictactoe" //this can be your bubble tea TUI game application
	"github.com/metallust/dosssh/server"
)

func main() {
  server.StartServer(tictactoe.InitialModel)
}

```
### Client side code
a bubble tea application need a model which is a struct with init, update and view method
```go
//model
type Model struct {
  page  int
  gameClient     *client.GameClient
}

func InitialModel(user string, conn *connector.Connector) tea.Model {
	m := Model{}
	m.gameClient = client.NewGameClient(conn, user)
	m.Page = MENUPAGE
	return m
}

// update method
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msgString {
        case "n", "N":
            m.Page = LOADINGPAGE
            return m, m.gameClient.Create("create")
        }
    case client.DoneMsg:
        switch msg.Msg {
        case "create":
            m.Page = GAMEPAGE
            m.OpponentStatus = "Waiting ..."
            return m, m.gameClient.ListenServer()
        }
    case client.ClientMsg:
    case client.GameClientMsg:
    }
}
```
After each function return a bubble tea command which is returned and when the function is finished executing a done msg is emitted where the model can be updated
There are other 2 type of msg you have to handle
1. GameClientMsg (rename to ServerMsg)
     1. JOINREQMSG
        ``` go
        case client.JOINREQMSG:
            m.Page = ACCEPTPAGE
            // Opponent := msg.Data.(string)
            Opponent := msg.Data.(string)
            cmd := m.acceptPage.SetQuestionAnswer("Accept request from "+Opponent+" ?", []string{"Yes", "No"})
            return m, tea.Batch(m.gameClient.ListenServer(), cmd)
        ```
     3. ERRORMSG
        ```go
        case client.ERRORMSG:
            m.ErrorPage.SetError(msg.Data.(string))
            m.Page = ERRORPAGE
            return m, tea.Batch(doTick("errortimeup"), m.gameClient.ListenServer())
        ```
2. ClientMsg (rename this to opponent msg)
      1. MOVEMSG
         ```go
         case client.MOVEMSG:
            move := msg.Data.([2]int)
            m.Game.Board[move[0]][move[1]] = m.Game.CurrentPlayer
            m.Game.CurrentPlayer = m.Game.Player
         ```
      2. ERRORMSG
         ```go
         case client.ERRORMSG:
         ```
3. DoneMsg -> this msg is emites when function is done executing it, if fails emits "error" if passed it emmits the string which was passed
    for example
   ```go
       return m, m.gameClient.Create("create")
   ```
   here create was passed so when done DONEMSG with "create" is emmited
   ```go
       case client.DoneMsg:
        switch msg.Msg {
        case "create":
           //handle after create
        }
   ```


list of client side function
### Create
### List
### JOIN -> rename to REQUEST
### ACCEPTREQUEST
### MOVE
### Create
### QUIT 

list of important functions rename the following to command
### LISTENTOSERVER -> return listen cmd after every server msg receive
### LISTENOPPONENT -> return listen cmd after every opponent msg receive

<!-- TODO: Examples--> 
<!-- paste the tictactoe example-->
