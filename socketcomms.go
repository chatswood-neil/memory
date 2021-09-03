// ----------------------------------------------------------------------------
// Server to Client - Message type is first Json field
//
//    {Type: "GamesInProgress"
//     Games: [Array of Games]}
//
//    {Type: "Flipped"
//     Tile:  int}
//
//    {Type: "Hidden"
//     Tile1:  int
//     Tile2:  int}
//
//    {Type: "Removed"
//     Tile1:  int
//     Tile2:  int}
//
// Client to Server - Message type in clear text, followed by Json payload
//
//    NewGame
//    {Idx: int
//     Tmax: int
//     OppBot: int
//     Name: string}
//
//    JoinGame
//    {Idx: int
//     Name: string}
//
//    FlipTile
//    {Tile: int}
//
//    End
// ----------------------------------------------------------------------------

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

// ---------------------------------------------------------------------------
// Start Websocket by sending upgrade HTTP message
// ---------------------------------------------------------------------------

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func startWebsocket(w http.ResponseWriter, r *http.Request) *websocket.Conn {

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket connection

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatalln(err)
	}
	if VerboseGlobal {
		log.Println("Client upgraded to WebSocket")
	}

	if conn == nil {
		log.Fatalln("WS connection closed or lost")
	}

	return conn
}

// ---------------------------------------------------------------------------
// Blocking function waits until a valid NewGame or JoinGame message is read
//
//    NewGame
//    {Idx: int
//     Tmax: int
//     OppBot: int
//     Name: string}
//
//    JoinGame
//    {Idx: int
//     Name: string}
// ---------------------------------------------------------------------------

func startOrJoin(conn *websocket.Conn) (player_t, int, int, int, bool) {
	nullPlayer := player_t{}

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return nullPlayer, 0, 0, 0, false
		}

		if messageType != websocket.TextMessage {
			return nullPlayer, 0, 0, 0, false
		}

		if VerboseGlobal {
			log.Println("startOrJoin: received [", string(msg), "]")
		}

		if string(msg[0:7]) == "NewGame" {
			type newgame_t struct {
				Idx    int
				Tmax   int
				OppBot int
				Name   string
			}
			var ng newgame_t
			log.Println("startOrJoin: unmarshal")
			json.Unmarshal(msg[7:], &ng)

			if Games[ng.Idx].Status != GAME_EMPTY ||
				ng.Tmax <= 0 || ng.OppBot < 0 || len(ng.Name) == 0 {
				return nullPlayer, 0, 0, 0, false
			}

			player1 := player_t{ng.Name, 1, false, "", 50, 50, nil, nil}

			return player1, ng.Tmax, ng.Idx, ng.OppBot, true
		}

		if string(msg[0:8]) == "JoinGame" {
			type joingame_t struct {
				Idx  int
				Name string
			}
			var jg joingame_t
			json.Unmarshal(msg[8:], &jg)

			if Games[jg.Idx].Status == GAME_WAITING || len(jg.Name) == 0 {
				return nullPlayer, 0, 0, 0, false
			}

			player2 := player_t{jg.Name, 2, false, "", 0, 0, nil, nil}

			return player2, Games[jg.Idx].Tmax, jg.Idx, 0, true
		}

		if VerboseGlobal {
			log.Println("startOrJoin: Ignored message [", string(msg), "]")
		}
	} // for loop
}

// ---------------------------------------------------------------------------
// Read Flip and End messages from the socket, and put onto the Move channel
// ---------------------------------------------------------------------------

func socketReader(conn *websocket.Conn, move chan string, p int) {
	var messageType int
	var msg []byte
	var err error

	// Indefinite loop terminates when client asks to close socket/server or
	// socket read fails.
	for {
		messageType, msg, err = conn.ReadMessage()
		if err != nil {
			move <- "N" // close TODO
			log.Println(err)
			break
		}

		if messageType != websocket.TextMessage {
			if VerboseGlobal {
				log.Println("Reader: Ignored non-text message")
			}
			continue
		}

		if VerboseGlobal {
			log.Println("Reader: received [", string(msg), "]")
		}

		var msgMap map[string]interface{}

		json.Unmarshal(msg, &msgMap)

		if msgMap["Type"] == "Flip" {
			move <- fmt.Sprintf("F%1d%03d", p, msgMap["Tile"])
		} else if msgMap["Type"] == "End" {
			move <- "E"
		}
	} // For loop
	SessWG.Done()
}

// ---------------------------------------------------------------------------
// Read board channel and tell client which tiles to flip/hide/remove
// ---------------------------------------------------------------------------

func socketWriter(conn *websocket.Conn, board chan string, p int) {
	var sent bool
	for {
		select {
		case b := <-board:
			if b[0] == 'F' || b[0] == 'O' { // (F)lipped
				idx, _ := strconv.Atoi(b[1:4])
				revealed_val, _ := strconv.Atoi(b[4:7])
				myTile := true
				if b[0] == 'O' {
					myTile = false
				}
				sent = SendFlipTiles(conn, myTile, idx, revealed_val)
			} else if b[0] == 'H' { // (H)ide unmatched tiles
				idx1, _ := strconv.Atoi(b[1:4])
				idx2, _ := strconv.Atoi(b[4:7])
				sent = SendHideTiles(conn, idx1, idx2)
			} else if b[0] == 'R' { // (R)emove matched tiles
				idx1, _ := strconv.Atoi(b[1:4])
				idx2, _ := strconv.Atoi(b[4:7])
				sent = SendRemoveTiles(conn, idx1, idx2)
			}
		}
		if !sent {
			break
		}
	}
	SessWG.Done()
}

func SendFlipTiles(conn *websocket.Conn, myTile bool, idx, val int) bool {
	msgMap := map[string]string{
		"Type":    "Flipped",
		"Tile":    fmt.Sprint(idx),
		"MyTile":  strconv.FormatBool(myTile),
		"Display": TileFaces[val],
	}

	return sendJsonMsg(conn, &msgMap)
}

func sendJsonMsg(conn *websocket.Conn, msgMap *map[string]string) bool {
	msgJson, err := json.Marshal(msgMap)
	if err != nil {
		log.Fatalln(err)
	}
	if VerboseGlobal {
		fmt.Println("Json Message to client:", string(msgJson))
	}

	err = conn.WriteMessage(websocket.TextMessage, msgJson)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func SendHideTiles(conn *websocket.Conn, tile1, tile2 int) bool {
	msgMap := map[string]string{
		"Type":  "Hidden",
		"Tile1": fmt.Sprint(tile1),
		"Tile2": fmt.Sprint(tile2),
	}
	msgJson, err := json.Marshal(msgMap)
	if err != nil {
		log.Fatalln(err)
	}
	if VerboseGlobal {
		fmt.Println("Json Message to client:", string(msgJson))
	}

	err = conn.WriteMessage(websocket.TextMessage, msgJson)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func SendRemoveTiles(conn *websocket.Conn, tile1, tile2 int) bool {
	msgMap := map[string]string{
		"Type":  "Removed",
		"Tile1": fmt.Sprint(tile1),
		"Tile2": fmt.Sprint(tile2),
	}
	msgJson, err := json.Marshal(msgMap)
	if err != nil {
		log.Fatalln(err)
	}
	if VerboseGlobal {
		fmt.Println("Json Message to client:", string(msgJson))
	}

	err = conn.WriteMessage(websocket.TextMessage, msgJson)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Send client a list of games in progress, including an empty "New" game
// ---------------------------------------------------------------------------

func SendGamesInProgress(conn *websocket.Conn) bool {

	msgJson, err := json.Marshal(Games)
	if err != nil {
		log.Fatalln(err)
	}
	if VerboseGlobal {
		fmt.Println("Json Message to client:", string(msgJson))
	}

	// Manually prepend type
	fullJson := fmt.Sprintf("{\"Type\":\"GamesInProgress\",\"Games\":%s}", string(msgJson))
	err = conn.WriteMessage(websocket.TextMessage, []byte(fullJson))
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}
