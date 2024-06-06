package connector

import "errors"

const (
	OKMSG = iota
	JOINREQMSG
	CREATEMSG
	LISTMSG
	MOVEMSG
	EXITMSG
	ERRORMSG
	RETURNLOBBYMSG
)

type Msg struct {
	Name      int
	Data      interface{}
	reply     bool
	replyChan chan Msg
}

func (m Msg) Reply(msgName int, data interface{}, wantReply bool) (<-chan Msg, error) {
	if !m.reply {
		return nil, errors.New("is msg is of non-replyable type")
	}
	replymsg := Msg{
		Name:  msgName,
		Data:  data,
		reply: wantReply,
	}
	if wantReply {
		replymsg.replyChan = make(chan Msg)
	}

	m.replyChan <- replymsg
	close(m.replyChan)
	return replymsg.replyChan, nil
}

type Connector struct {
	Sender   chan Msg
	Reciever chan Msg
}

func NewConnector() *Connector {
	return &Connector{
		Sender:   make(chan Msg, 4),
		Reciever: make(chan Msg, 4),
	}
}

func CreateConnectorPair(c *Connector) *Connector {
	return &Connector{
		Sender:   c.Reciever,
		Reciever: c.Sender,
	}
}

func (c *Connector) SendMsg(msgName int, data interface{}, reply bool) <-chan Msg {
	msg := Msg{
		Name:  msgName,
		Data:  data,
		reply: reply,
	}
	if reply {
		msg.replyChan = make(chan Msg)
	}
	c.Sender <- msg
	return msg.replyChan
}

func (c *Connector) GetMsg() (Msg, bool) {
	msg, more := <-c.Reciever
	return msg, more
}

func (c *Connector) Close() {
	close(c.Sender)
	close(c.Reciever)
}
