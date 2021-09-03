// ---------------------------------------------------------------------------
// A two-player tile-turning memory game in real time (not turn-based)
//  - CLient-server architecture
//  - One, two, or zero computer players (membots)
//
// Files in package:
//   memory.go - the HTTP server and client socket
//   gamemanager.go - controls a sequence of two-player games
//   membot.go - implements a computer player with variable ability
// ---------------------------------------------------------------------------

package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"math/rand"

	"github.com/gorilla/websocket"
)

// -------------------------------------------------------------------------
// STRUCTURES & CONSTANTS
// -------------------------------------------------------------------------

const GAME_SLOTS int = 2

const GAME_EMPTY int = 0
const GAME_WAITING int = 1
const GAME_RUNNING int = 2

type client_t struct {
	gameIdx int
	player  int
}

type clientMap_t map[string]client_t

type player_t struct {
	Name  string
	Num   int
	IsBot bool

	// Below not shared with client
	clientIP string
	slowPc   int
	memPc    int
	move     chan string
	board    chan string
}

type game_t struct {
	Status      int // Waiting, Running
	Tmax        int
	P1          player_t
	P2          player_t
	P1won       int
	P2won       int
	GameCounter int
	// Below not shared with client
	moveCounter int
}

type gameTable_t [20]game_t

// ---------------------------------------------------------------------------
// GLOBALS
// ---------------------------------------------------------------------------

var SessWG sync.WaitGroup = sync.WaitGroup{}

var VerboseGlobal = true

var Games gameTable_t

var TileFaces map[int]string

// ---------------------------------------------------------------------------
// Main
//  - Start an (immortal) webserver. This will serve the game page and images
//    for the life of the game
//  - From the game page, a player nominates number of bots and other
//    parameters and starts a game by upgrading to a socket connection.
//  - The socket upgrade triggers
// ---------------------------------------------------------------------------

func main() {
	setTileFaces()

	HttpsServer(8088)
	return
}

func HttpsServer(port int) {
	var serverAddr string = fmt.Sprintf(":%d", port)

	// HTTP request multiplexer. URL of request is matched against the
	// registered patterns to findhandler function.

	mux := http.NewServeMux()
	mux.HandleFunc("/", httpHandleRequest)
	mux.HandleFunc("/game/", wssGame)

	if VerboseGlobal {
		fmt.Printf("Listening on port %d (%s)...\n", port, serverAddr)
	}

	// ListenAndServeTLS blocks and loops indefinitely
	log.Fatal(http.ListenAndServeTLS(serverAddr, "cert.pem", "key.pem", mux))

	// To generate certificates:
	// go run "C:\Program Files\Go\src\crypto\tls\generate_cert.go" -host="127.0.0.1"
}

// ----------------------------------------------------------------------------
// Standard file server to return initial page, javascript, favicons, etc
// ----------------------------------------------------------------------------

func httpHandleRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Serve file for request:", r)

	fmt.Println("File path:", r.URL.Path[1:])

	if r.URL.Path == "/" {
		http.ServeFile(w, r, "memgame.html")
	} else {
		http.ServeFile(w, r, r.URL.Path[1:])
	}
}

// ---------------------------------------------------------------------------
// This function handles a player starting a new game as player 1 or joining
// a waiting game as player 2:
//  - Spawned as a goroutine by HTTP server
//  - If player1 asks for a bot opponent, this function starts the game
//    immendiately. Otherwise, wait for player2 to join.
//  - Player2 can also specify a bot to run. TODO
// ---------------------------------------------------------------------------

func wssGame(w http.ResponseWriter, r *http.Request) {

	// -------------------------------------------------------------------------
	// Startup a websocket for this connection
	// -------------------------------------------------------------------------

	var wssConn *websocket.Conn
	wssConn = startWebsocket(w, r)
	defer wssConn.Close()

	// -------------------------------------------------------------------------
	// Create the socket I/O channels
	// -------------------------------------------------------------------------

	move_chan := make(chan string, 10)  // Socket Reader to Game Manager
	board_chan := make(chan string, 10) // Game Manager to Socket Writer

	defer close(move_chan)
	defer close(board_chan)

	// -------------------------------------------------------------------------
	// Advise client of all games in progress
	// -------------------------------------------------------------------------

	success := SendGamesInProgress(wssConn)
	if !success {
		log.Fatalln("Could not send game table to client")
	}
	// TODO   - SendBotProfiles(wssConn)

	// -------------------------------------------------------------------------
	// Wait (block) for a new game request or a join game request
	// -------------------------------------------------------------------------

	humanPlayer, tMax, gameIdx, bot, success := startOrJoin(wssConn)
	if !success {
		log.Fatalln("Bad attempt to start or join a game")
	}

	humanPlayer.clientIP = r.RemoteAddr
	humanPlayer.move = move_chan
	humanPlayer.board = board_chan

	// -------------------------------------------------------------------------
	// Waitgroup
	// -------------------------------------------------------------------------

	SessWG.Add(2)

	go socketReader(wssConn, move_chan, humanPlayer.Num)
	go socketWriter(wssConn, board_chan, humanPlayer.Num)

	// -------------------------------------------------------------------------
	// Create game and players
	// -------------------------------------------------------------------------

	if humanPlayer.Num == 1 {
		Games[gameIdx] = game_t{GAME_WAITING, tMax, humanPlayer, player_t{}, 0, 0, 0, 0}

		if bot > 0 {
			bot_move_chan := make(chan string, 10)  // Bot to Game Manager
			bot_board_chan := make(chan string, 10) // Game Manager to Bot

			defer close(bot_move_chan)
			defer close(bot_board_chan)

			// TODO - map bot index to slow and mem bot params and name.
			botPlayer := player_t{"MEMBOT", 2, true, "", 50, 99, bot_move_chan, bot_board_chan}
			Games[gameIdx].P2 = botPlayer
		}
	} else {
		//player2 := player_t{r.RemoteAddr, 2, name, botDelegate, 50, 99, move_chan, board_chan}
		Games[gameIdx].P2 = humanPlayer
	}

	// -------------------------------------------------------------------------
	// Start game when player 2 is connected
	// -------------------------------------------------------------------------

	emptyPlayer := player_t{}
	if Games[gameIdx].P2 != emptyPlayer {
		Games[gameIdx].Status = GAME_RUNNING
		gameManager(&Games[gameIdx], VerboseGlobal)
	}

	SessWG.Wait()
}

// ----------------------------------------------------------------------------
// Set tile face image file paths
// ----------------------------------------------------------------------------

func setTileFaces() {
	TileFaces = make(map[int]string)
	TileFaces[0] = "/static/Tile1_150.png"
	TileFaces[1] = "/static/Bhutto150.png"
	TileFaces[2] = "/static/Churchill150.png"
	TileFaces[3] = "/static/DeGaulle150.png"
	TileFaces[4] = "/static/Elizabeth150.png"
	TileFaces[5] = "/static/Gandhi150.png"
	TileFaces[6] = "/static/JohnPaul150.png"
	TileFaces[7] = "/static/Mao150.png"
	TileFaces[8] = "/static/Marley150.png"
	TileFaces[9] = "/static/Monroe150.png"
	TileFaces[10] = "/static/YokoOno150.png"
	TileFaces[99] = "/static/Empty150.png"
}

// ----------------------------------------------------------------------------
// Seed random generator
// Set tile face image file paths
// ----------------------------------------------------------------------------

func init() {
	rand.Seed(time.Now().UnixNano())
}
