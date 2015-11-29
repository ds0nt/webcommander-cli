package main

import (
	"os"
	"log"
	"github.com/codegangsta/cli"
	"github.com/gorilla/websocket"
	"encoding/json"
	"strings"
)

const (
  socketBufferSize  = 1024
  messageBufferSize = 256
)

var name = "ds0nt-bot"

type client struct {
  socket *websocket.Conn
  send   chan message
}

type message struct {
  Type string `json:"type"`
  Payload interface{} `json:"payload"`
}

var responder *processor

type processor struct {
	In  chan *message
	Out chan *message
}

func newProcessor(client *client) *processor {
	in := make(chan *message, messageBufferSize)
	out := make(chan *message, messageBufferSize)
	return &processor{in, out}
}

func (p *processor) process() {
	p.Out <-&message{"nick", name}
	log.Println("Starting processor")
	for msg := range p.In {
		if msg.Type == "chat" {
			text := msg.Payload.(string)
			if strings.HasPrefix(text, name) {
				continue
			}
			log.Println("I hear stuff", text)
			if strings.Contains(text, name) {
				log.Println("Someone is talking to me:", text)
				p.Out <-&message{"chat", "Du hast mich gefragt, und ich hab Nichsts gesact..."}
			}
		}
	}
}

func (p *processor) resultsTo(client *client) {
	for msg := range p.Out {
    client.send <- *msg
	}
}

func newClient(socket *websocket.Conn, name string) *client {
  return &client{socket, make(chan message, messageBufferSize)}
}

func (c *client) read() {
  for {
    _, msg, err := c.socket.ReadMessage()
    if err == nil {
      var f message
      err := json.Unmarshal(msg, &f)
      if err != nil {
        log.Printf("Evil JSON Detected: %v, %v", err, string(msg))
        continue
      }
      log.Printf("Message Received: %v", f)
			responder.In <-&f
    } else {
      break
    }
  }
  c.socket.Close()
}

func (c *client) write() {
  for msg := range c.send {
    bytes, err := json.Marshal(&msg)
    if err != nil {
      log.Printf("Client Write Json Marshal Error: %v, %v", err, msg)
    }
    if err := c.socket.WriteMessage(websocket.TextMessage, bytes); err != nil {
      log.Printf("Client Write Error: %v", err)
      break
    } else {
      log.Printf("Client Send: %v", msg)
    }
  }
}

func dial(url string) {
	log.Printf("Dialing: %s", url)
	d := websocket.Dialer{}
	ws, _, err := d.Dial(url, nil)
	if err != nil {
		log.Panicf("An error occured while dialing: %s", err)
	}

  client := newClient(ws, "ds0nt-bot")
	responder = newProcessor(client)
	go responder.process()
	go responder.resultsTo(client)
  go client.write()
  client.read()
}


func main() {
	log.Println("Web Commander CLI Client")
	// https://github.com/codegangsta/cli
	app := cli.NewApp()
	app.Name = "webcommander-cli"
	app.Usage = "Web Commander CLI Client"
	app.Commands = []cli.Command{
		{
			Name:    "dial",
			Action: func(c *cli.Context) {
				dial(c.Args().First())
			},
		},
	}
	app.Run(os.Args)
}
