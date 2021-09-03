// -------------------------------------------------------------------------
// The game manager runs a sequence of games.
//
// Siam's gonna be the witness
// To the ultimate test of cerebral fitness
// This grips me more than would a
// Muddy old river or reclining Buddha
// And thank God I'm only watching the game, controlling it
// -------------------------------------------------------------------------

package main

import (
	"fmt"
	"log"
	"strconv"
	"sync"

	"math/rand"
)

// -------------------------------------------------------------------------
// STRUCTURES & CONSTANTS
// -------------------------------------------------------------------------

const FACEDOWN int = 0
const FACEUP_P1 int = 1
const FACEUP_P2 int = 2
const WON_BY_P1 int = 11
const WON_BY_P2 int = 22
const NOVAL int = 0

type tile_t struct {
	disp int
	val  int
}
type tilearray_t []tile_t

// -------------------------------------------------------------------------
// GLOBALS
// -------------------------------------------------------------------------

var game_wg sync.WaitGroup = sync.WaitGroup{}

// -------------------------------------------------------------------------
// Game Manager
// -------------------------------------------------------------------------

func gameManager(game *game_t, verbose bool) {

	// -------------------------------------------------------------------------
	// Define the board
	// -------------------------------------------------------------------------

	var board = make(tilearray_t, game.Tmax)

	// -------------------------------------------------------------------------
	// PLay the game
	// -------------------------------------------------------------------------

	for {
		game.GameCounter++
		game.moveCounter = 0

		initBoard(game.Tmax, board[:])

		if game.P1.IsBot {
			game_wg.Add(1)
			go memBot(game.P1, game.Tmax, verbose, board[:])
		}

		if game.P2.IsBot {
			game_wg.Add(1)
			go memBot(game.P2, game.Tmax, verbose, board[:])
		}

	read_moves_loop:
		for {

			// Block until a (F)lip or a (N)o Move received from a player
			game.moveCounter++
			select {
			case msg := <-game.P1.move:
				if msg[0:1] == "F" {
					idx, _ := strconv.Atoi(msg[1:4])
					flipTile(game, 1, idx, board[:])
				}
				if msg[0:1] == "N" {
					if isGameFinished(board[:]) {
						break read_moves_loop
					}
				}
			case msg := <-game.P2.move:
				if msg[0:1] == "F" {
					idx, _ := strconv.Atoi(msg[1:4])
					flipTile(game, 2, idx, board[:])
				}
				if msg[0:1] == "N" {
					if isGameFinished(board[:]) {
						fmt.Println("Game is finished")
						break read_moves_loop
					}
				}
			}

			if verbose {
				gameTextDisp(board[:])
			}
		}
		fmt.Println("Waiting for bots to finish")
		game_wg.Wait() // Wait for all bots to terminate
		fmt.Println("All bots finished")

		winner := gameWinner(board[:])
		if verbose {
			fmt.Println("===========", game)
			if winner == 0 {
				fmt.Println("=========== Game tied ")
			} else if winner == 1 {
				game.P1won++
				fmt.Println("=========== Game won by player 1 ")
			} else {
				game.P2won++
				fmt.Println("=========== Game won by player 2 ")
			}

			fmt.Println("LEADERBOARD", game.P1won, "VS", game.P2won)
		}
	} // Loop - play games indefinitely
}

func initBoard(tMax int, board tilearray_t) {
	for _, tile := range board {
		tile.disp = FACEDOWN
		tile.val = NOVAL
	}
	for v := 1; v < int(tMax/2)+1; v++ { // Half as many values as tiles
		for t := 1; t < 3; t++ { // Two tiles per value
			idx := rand.Intn(tMax)
			for {
				if board[idx].val == NOVAL {
					board[idx].val = v
					break
				}
				idx++
				if idx == tMax {
					idx = 0
				}
			}
		}
	}
	//log.Println("Board initialised")
	gameTextDisp(board[:])
}

func isGameFinished(board tilearray_t) bool {
	for _, tile := range board {
		if tile.disp == FACEDOWN {
			return false // Game over when no face-down tiles remain
		}
	}
	return true
}

func gameWinner(board tilearray_t) int {
	player1 := 0
	for _, tile := range board {
		if tile.disp == WON_BY_P1 {
			player1++
		} else if tile.disp == WON_BY_P2 {
			player1--
		}
	}

	if player1 > 0 {
		return 1
	} else if player1 < 0 {
		return 2
	}
	return 0
}

func gameTextDisp(board tilearray_t) {
	fmt.Print("  +")
	for t, _ := range board {
		fmt.Printf("%02d+", t)
	}
	fmt.Print("\n  |")
	for _, tile := range board {
		fmt.Printf("%02d|", tile.val)
	}
	fmt.Print("\n  |")
	for _, tile := range board {
		fmt.Printf("%02d|", tile.disp)
	}
	fmt.Print("\n\n")
}

// ---------------------------------------------------------------------------
// Process the flipping of a tile.
//
// This function is synchronous, as no async update of board is permitted.
//
// Messages sent: (F)lip, (O)pponent flip, (H)ide tiles, (R)emove tiles
// ---------------------------------------------------------------------------

func flipTile(game *game_t, p, flip_idx int, board tilearray_t) {

	// First ensure the tile can be flipped - if not, then ignore and do not
	// advise players. This is expected frequently due to race conditions.
	if board[flip_idx].disp != FACEDOWN {
		return
	}

	// Determine set of previously upturned tiles
	me_up1 := -1
	me_up2 := -1
	opp_up1 := -1

	for idx, tile := range board {
		if tile.disp == FACEUP_P1 || tile.disp == FACEUP_P2 {
			if tile.disp == p {
				if me_up1 == -1 {
					me_up1 = idx
				} else {
					me_up2 = idx
				}
			} else {
				if opp_up1 == -1 {
					opp_up1 = idx
				}
			}
		}
	}

	if VerboseGlobal {
		log.Println("Player", p, "has face-up tiles", me_up1, me_up2)
	}

	// If current player has two tiles up, face them both down
	if me_up1 >= 0 && me_up2 >= 0 {
		board[me_up1].disp = FACEDOWN
		board[me_up2].disp = FACEDOWN
		d_str := fmt.Sprintf("H%03d%03d", me_up1, me_up2)
		game.P1.board <- d_str
		game.P2.board <- d_str
		me_up1 = -1
		me_up2 = -1
	}

	// Flip tile and advise both players of revealed tile value
	flip_val := board[flip_idx].val
	board[flip_idx].disp = p // FACEUP
	f_str := fmt.Sprintf("F%03d%03d", flip_idx, flip_val)
	o_str := fmt.Sprintf("O%03d%03d", flip_idx, flip_val)
	if p == 1 {
		game.P1.board <- f_str
		game.P2.board <- o_str
	} else {
		game.P1.board <- o_str
		game.P2.board <- f_str
	}

	// Determine if flipped tile is part of a matched pair.
	// Note that a guzump is only possible if only upface tile is opponent's.

	match_idx := -1
	if me_up1 >= 0 {
		if flip_val == board[me_up1].val {
			match_idx = me_up1 // Normal two-tile win
		}
	} else if opp_up1 >= 0 && flip_val == board[opp_up1].val {
		match_idx = opp_up1 // Guzump win
	}

	if match_idx == -1 {
		return
	}

	// Mark both as won by current player. Advise both players of removal.
	win := p * 11
	board[match_idx].disp = win
	board[flip_idx].disp = win
	rm_str := fmt.Sprintf("R%03d%03d", flip_idx, match_idx)
	game.P1.board <- rm_str
	game.P2.board <- rm_str

	return
}
