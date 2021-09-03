package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"math/rand"
)

// ---------------------------------------------------------------------------
// A memory-game bot
//  p = player 1 or 2
//  tMax = number of tiles in tile arrays
//  mem = a slice/array of remembered tile IDs indexed by tile number
//  state = a slice/array of tile states
//          (0 = face-down, 1 = taken, 2 = face-up-you, 3 = face-up-opp)
//  known = a slice/array of tile numbers, indexed by tile ID
//  board = chan for game board to advise state-change on board
//  move =  chan to advise game board of next move (flip, hide)
// ---------------------------------------------------------------------------

const REMOVED int = 1
const FACEUP_ME int = 2
const FACEUP_OPP int = 3

func memBot(p player_t, tMax int, verbose bool, board tilearray_t) {
	var botmem = make(tilearray_t, tMax)

	if p.slowPc < 10 || p.slowPc > 100 {
		p.slowPc = 100
	}
	if p.memPc < 20 || p.memPc > 100 {
		p.memPc = 100
	}
	if verbose {
		log.Printf("Bot %d started - Slow%% %d - Memory%% %d\n", p.Num, p.slowPc, p.memPc)
		log.Printf("Capacity is %d\n", cap(board))
	}

	for {
		// Read board updates, and update in the bot's memory of the board
	channel_read_loop:
		for {
			select {
			case b := <-p.board:
				if b[0] == 'F' { // (F)lipped by this bot
					botmem[:].botRevealTile(p.Num, b, verbose)
				} else if b[0] == 'O' { // (O)pponent flipped tile
					botmem[:].botRevealTile(p.Num, b, verbose)
				} else if b[0] == 'H' { // (H)ide unmatched tiles
					botmem[:].botHideTiles(p.Num, b, verbose, p.memPc)
				} else if b[0] == 'R' { // (R)emove matched tiles
					botmem[:].botRemoveTiles(p.Num, b, verbose)
				}
			default:
				break channel_read_loop
			}
		}

		// For debug purposes, display bot's view of tiles
		if VerboseGlobal {
			botmem[:].dispBotmem(p)
		}

		// Choose and communicate a move
		tile_idx, noMove := botChoose(p.Num, botmem[:])

		if noMove {
			p.move <- "N"
			if VerboseGlobal {
				log.Println("Bot", p.Num, "could not make a move. Bot terminated.")
			}
			game_wg.Done()
			return
		}
		if VerboseGlobal {
			log.Println("Bot", p.Num, "chose tile", tile_idx)
		}
		flip_str := fmt.Sprintf("F%03d", tile_idx)
		p.move <- flip_str

		// Sleepy time for 'lil bot-bot
		// 100% slow (max): Sleep between 1000 and 2000 milliseconds
		// 10% slow (min): Sleep between 100 and 200 milliseconds
		s := 10 * p.slowPc
		time.Sleep(time.Duration(s+rand.Intn(s)) * time.Millisecond)
	}
}

func (botmem tilearray_t) dispBotmem(p player_t) {
	fmt.Print("Bot ", p.Num, " ")
	for _, tile := range botmem {
		if tile.val == NOVAL {
			fmt.Print("|--")
		} else {
			fmt.Printf("|%02d", tile.val)
		}
	}
	fmt.Print("|\n      ")
	for _, tile := range botmem {
		if tile.disp == FACEDOWN {
			fmt.Print("|--")
		} else if tile.disp == REMOVED {
			fmt.Print("|  ")
		} else {
			fmt.Printf("|%02d", tile.disp)
		}
	}
	fmt.Println("|")
}

func (botmem tilearray_t) botRevealTile(p int, msg string, verbose bool) {
	msg_id := msg[0]
	idx, _ := strconv.Atoi(msg[1:4])
	revealed_val, _ := strconv.Atoi(msg[4:7])
	if msg_id == 'F' {
		botmem[idx].disp = FACEUP_ME
	} else {
		botmem[idx].disp = FACEUP_OPP
	}
	botmem[idx].val = revealed_val
	if verbose {
		log.Println("Bot", p, "I flipped", idx, "revealing", revealed_val)
	}
}

func (botmem tilearray_t) botHideTiles(p int, msg string, verbose bool, memPercent int) {
	idx1, _ := strconv.Atoi(msg[1:4])
	idx2, _ := strconv.Atoi(msg[4:7])

	botmem[idx1].disp = FACEDOWN
	botmem[idx2].disp = FACEDOWN

	// Possibly forget both cards
	r := rand.Intn(100)
	if r >= memPercent {
		botmem[idx1].val = 0
		botmem[idx2].val = 0
	}

	if verbose {
		log.Println("Bot", p, "vvvv", idx1, "vvvv", idx2, "vvvv")
	}
}

func (botmem tilearray_t) botRemoveTiles(p int, msg string, verbose bool) {
	idx1, _ := strconv.Atoi(msg[1:4])
	idx2, _ := strconv.Atoi(msg[4:7])

	botmem[idx1].disp = REMOVED
	botmem[idx2].disp = REMOVED

	if verbose {
		log.Println("Bot", p, "MATCHED and removed", idx1, "and", idx2)
	}
}

// ---------------------------------------------------------------------------
// Bot chooses tile to flip.
//
// This function has no state other than the memory-map of the board. i.e. A
// call to this function independently re-evaluates next move based solely on
// the current state of the board upon entry.
//
// Returns: tile index to flip next
// ---------------------------------------------------------------------------

func botChoose(p int, botmem tilearray_t) (int, bool) {
	if VerboseGlobal {
		log.Printf("Bot %d Make a choice\n", p)
	}

	// Are there any upturned tiles? This decides whether move 1 or 2 or guzump
	// Also ensure there is at least one face-down tile on board.
	myTilesUpCnt := 0
	myTileVal := 0
	oppTilesUpCnt := 0
	oppTileVal := 0
	faceDownCnt := 0

	for _, tile := range botmem {
		if tile.disp == FACEUP_ME {
			myTilesUpCnt++
			myTileVal = tile.val
			//myTileNum = t
		} else if tile.disp == FACEUP_OPP {
			oppTilesUpCnt++
			oppTileVal = tile.val
		} else if tile.disp == FACEDOWN {
			faceDownCnt++
		}
	}

	// If no tiles face-down, end of game for this bot
	if faceDownCnt == 0 {
		return 0, true
	}

	if VerboseGlobal {
		log.Println("Bot", p, "found", myTilesUpCnt, "and", oppTilesUpCnt, "tiles face up")
	}

	// If you have one tile upturned (second move), there are three possibilities:
	// 1. We remember a hidden tile with same value -> choose it
	// 2. Our opponent has upturned same tile as us -> choose a random tile
	// 3. No knowledge of tile -> choose a random tile

	if myTilesUpCnt == 1 {
		for t, tile := range botmem {
			if tile.disp == FACEDOWN && tile.val == myTileVal {
				if VerboseGlobal {
					log.Println("Bot", p, "choose known match", t)
				}
				return t, false
			}
		}

		randTile := randomChoice(faceDownCnt, botmem[:])
		if VerboseGlobal {
			log.Println("Bot", p, "chooses random tile for second move", randTile)
		}
		return randTile, false
	}

	// Guzump is possible on move 1, scan memory for face-down match of
	// opponent's tile
	if oppTilesUpCnt == 1 {
		for t, tile := range botmem {
			if tile.disp == FACEDOWN && tile.val == oppTileVal {
				// Try to guzump. This is a race most likely won by opponent.
				if VerboseGlobal {
					log.Println("Bot", p, "try guzump tile", t)
				}
				return t, false
			}
		}
	}

	// If move 1 (bot has either zero or two tiles upturned)
	// Scan for face-down pairs: If pair found, pick one of these tile randomly
	// Otherwise, choose a random tile

	knownValues := make([]int, int(cap(botmem)/2))
	for t, tile := range botmem {
		if tile.disp == FACEDOWN && tile.val != NOVAL {
			idx := tile.val - 1
			if knownValues[idx] == NOVAL {
				knownValues[idx] = t
				continue
			}

			// This ID is non-zero, so have found a pair
			// Randomly choose either the first or second tile of the pair
			choice := knownValues[idx]
			if rand.Intn(2) == 0 {
				choice = t
			}
			if VerboseGlobal {
				log.Println("Bot", p, "chooses first tile of known pair", choice)
			}
			return choice, false
		}
	}

	randTile := randomChoice(faceDownCnt, botmem[:])
	if VerboseGlobal {
		log.Println("Bot", p, "chooses random tile for first move", randTile)
	}
	return randTile, false
}

// ---------------------------------------------------------------------------
// Random choice of face-down tiles (state == 0)
//
// Returns: tile to flip next
// ---------------------------------------------------------------------------

func randomChoice(faceDownCnt int, botmem tilearray_t) int {
	r := rand.Intn(faceDownCnt)

	for t, tile := range botmem {
		if tile.disp == FACEDOWN {
			if r == 0 {
				return t
			}
			r--
		}
	}

	return 0
}
