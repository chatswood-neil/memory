// ---------------------------------------------------------------------------
// Session state
// ---------------------------------------------------------------------------

const state = { UNCONNECTED:0, CONNECTED: 1, WAITING:2, PLAYING:3, FINISHED:4 }
var SessionStatus = state.UNCONNECTED;

// ---------------------------------------------------------------------------
// Button protection
// ---------------------------------------------------------------------------

const btn = { CANCLICK:0, PROTECTED:1 }
var buttonArray = [];

// ---------------------------------------------------------------------------
// Start up the websocket
// ---------------------------------------------------------------------------

document.addEventListener("DOMContentLoaded", startWebsocket())

function startWebsocket() {
  socket = new WebSocket("wss://127.0.0.1:8088/game/");
  console.log("Attempting Connection...");

  socket.addEventListener('open', function (event) {
    console.log("Successfully Connected");

    SessionStatus = state.CONNECTED;
  });

  // Listen for messages

  socket.addEventListener('message', function (event) {
      console.log('Message from server ', event.data);
      var msg_obj
      try {
        msg_obj = JSON.parse(event.data);
      } catch (e) {
        alert(e);
        return;
      }

      switch (msg_obj.Type)
      {
        case "GamesInProgress":
                         createGameSelector(msg_obj.Games);
                         break;
        case "Flipped":  flipTile(msg_obj);
                         break;
        case "Hidden":   hideTiles(msg_obj);
                         break;
        case "Removed":  removeTiles(msg_obj);
                         break;
        case "Finished": removeTiles(msg_obj);
                         break;
        default:         alert("Unknown message", msg_obj);
      }
  });

  socket.onclose = event => {
    console.log("Socket Closed Connection: ", event);
    SessionStatus = state.UNCONNECTED;
  };

  socket.onerror = error => {
    console.log("Socket Error: ", error);
    SessionStatus = state.UNCONNECTED;
  };
};

// ---------------------------------------------------------------------------
// Game Selector setup
// ---------------------------------------------------------------------------

function createGameSelector(gameArray) {
  if (SessionStatus != state.CONNECTED) {
    console.log("Cannot show game selector in this state")
    return
  }

  // Find last non-empty game, and provide one extra new game below
  for (maxg = gameArray.length; maxg > 0 ; maxg--) {
    if (gameArray[maxg-1].Status != 0) break;
  }
  if (maxg != gameArray.length) {maxg++}

  gameSel = document.querySelector(".gameSelect");
  for (let g = 0; g < maxg; g++) {
    var newGameLineSpc = document.createElement("div")
    newGameLineSpc.setAttribute("class", "gameLineSpace")
    gameSel.appendChild(newGameLineSpc)

    var newGameLine = document.createElement("div")
    newGameLine.setAttribute("class", "gameLine")
    newGameLine.setAttribute("id", g)
    newGameLineSpc.appendChild(newGameLine)

    var newGameStatus = document.createElement("div")
    newGameStatus.setAttribute("class", "gameStatus")
    newGameStatus.setAttribute("id", g)
    if (gameArray[g].Status === 2) {
      newGameStatus.innerHTML = "In Progress"
    } else if (gameArray[g].Status === 1) {
      newGameStatus.innerHTML = "Waiting"
    } else if (gameArray[g].Status === 0) {
      var newGameButton = document.createElement("button")
      var buttonText = document.createTextNode("New Game");
      newGameButton.appendChild(buttonText);
      newGameButton.setAttribute("id", g);
      newGameButton.onclick = newGameReq
      newGameStatus.appendChild(newGameButton);
    }
    newGameLine.appendChild(newGameStatus)

    var newGameP1 = document.createElement("div")
    newGameP1.setAttribute("class", "gameP1")
    if (gameArray[g].Status != 0) {
      newGameP1.innerHTML = gameArray[g].P1.Name
    } else {
      var nameForm = document.createElement("form")
      var nameInput = document.createElement("input")
      nameForm.appendChild(nameInput);
      newGameP1.appendChild(nameForm);
    }
    newGameLine.appendChild(newGameP1)

    var newGameP2 = document.createElement("div")
    newGameP2.setAttribute("class", "gameP2")
    /*newGameStatus.addEventListener("click", flipTile)*/
    newGameLine.appendChild(newGameP2)
  }
}

// ---------------------------------------------------------------------------
// Board setup
// ---------------------------------------------------------------------------

function createBoard(tMax) {
  let grid = document.querySelector(".grid");
  for (let i = 0; i < tMax; i++) {
    var newTileSpc = document.createElement("div");
    newTileSpc.setAttribute("class", "tilespace");
    grid.appendChild(newTileSpc);
    var newTile = document.createElement("img");
    newTile.setAttribute("src", "static/Tile1_150.png");
    newTile.setAttribute("class", "faceDown");
    newTile.setAttribute("id", "tile"+i);
    newTile.addEventListener("click", flipTileReq);
    newTileSpc.appendChild(newTile);
    buttonArray[i] = btn.CANCLICK;
  }
};

// ---------------------------------------------------------------------------
// Reset the board after a game has been played
// ---------------------------------------------------------------------------

function resetBoard(tMax) {
  let grid = document.querySelector(".grid");
  for (let i = 0; i < tMax; i++) {
    var tile = document.getElementById("tile"+i);
    tile.setAttribute.source = "static/Tile1_150.png";
    tile.setAttribute.class = "faceDown";
    buttonArray[i] = btn.CANCLICK;
  }
};

// ---------------------------------------------------------------------------
// Send request to start new game
//    NewGame
//    {Idx: int
//     Tmax: int
//     Bot: int
//     Name: string}
// ---------------------------------------------------------------------------

function newGameReq(event) {

  let g = event.target.getAttribute("id");
  console.log("New Game Button selected for game:"+g)

  if (SessionStatus === state.UNCONNECTED || SessionStatus === state.PLAYING) {
    console.log("Cannot start new game in status", SessionStatus)
    return
  }

  let Tmax = 20      // TODO
  let Name = "Neil"  // TODO
  let OppBot = 1     // TODO
  //let OppBot   = document.getElementsByName("mapHeightParam")[0].value;

  newGameStruct = {"Idx":g|0, "Tmax":Tmax|0, "OppBot":OppBot|0, "Name":Name};
  newGameJSON = JSON.stringify(newGameStruct);

  if (socket.readyState != WebSocket.OPEN) {
    console.log("Socket died!");
    return
  }

  socket.send("NewGame"+newGameJSON);

  if (SessionStatus === state.FINISHED) {
    resetBoard(Tmax)
  } else {
    createBoard(Tmax)
  }

  SessionStatus = state.PLAYING;
};

// ---------------------------------------------------------------------------
// Send request to Join game
//    JoinGame
//    {Idx: int
//     Name: string}
// ---------------------------------------------------------------------------

function joinGameReq(g) {
  console.log("Join Game Button selected")
  let Name = "Neil"  // TODO
  //let Bot   = document.getElementsByName("mapHeightParam")[0].value;

  joinGameStruct = {"Idx":g|0, "Name":Name};
  joinGameJSON = JSON.stringify(joinGameStruct);

  if (socket.readyState === WebSocket.OPEN) {
    socket.send("JoinGame"+joinGameJSON);
  } else {
    console.log("Socket died!");
  }
};

// ---------------------------------------------------------------------------
// Send request to Flip a tile
//    FlipTile
//    {Idx: int}
// ---------------------------------------------------------------------------

function flipTileReq(t) {
  flipTileStruct = {"Idx":t|0};
  flipTileJSON = JSON.stringify(flipTileStruct);

  if (socket.readyState === WebSocket.OPEN) {
    socket.send("FlipTile"+flipTileJSON);
  } else {
    console.log("Socket died!");
  }
};

// ---------------------------------------------------------------------------
// Handle Flipped message
//    Tile:  int
//    MyTile: boolean
//    Display: image path string
// ---------------------------------------------------------------------------

function flipTile(msgObj) {
  tile = document.getElementById("tile"+msgObj.Tile)
  tile.setAttribute("src", msgObj.Display)
  tile.setAttribute("class", msgObj.MyTile ? "faceUpRed" : "faceUpBlue")
}

// ---------------------------------------------------------------------------
// Handle Hide message
//    Tile1:  int
//    Tile2:  int
// ---------------------------------------------------------------------------

function hideTiles(msgObj) {
  sleep(400)
  t1 = document.getElementById("tile"+msgObj.Tile1)
  t1.setAttribute("src", "static/Tile1_150.png")
  t1.setAttribute("class", "faceDown")
  t2 = document.getElementById("tile"+msgObj.Tile2)
  t2.setAttribute("src", "static/Tile1_150.png")
  t2.setAttribute("class", "faceDown")
}

// ---------------------------------------------------------------------------
// Handle Remove message
//    Tile1:  int
//    Tile2:  int
// ---------------------------------------------------------------------------

function removeTiles(msgObj) {
  t1 = document.getElementById("tile"+msgObj.Tile1)
  t2 = document.getElementById("tile"+msgObj.Tile2)
  t1.className += ' item-fade';
  t2.className += ' item-fade';
  sleep(800)

  t1.setAttribute("src", "static/Blank150.png")
  t1.setAttribute("class", "empty")

  t2.setAttribute("src", "static/Blank150.png")
  t2.setAttribute("class", "empty")
}

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

var xXtileArray = [
  {
    name: 'Bhutto',
    img: 'static/Bhutto150.png'
  },
  {
    name: 'Churchill',
    img: 'static/Churchill150.png'
  },
  {
    name: 'DeGaulle',
    img: 'static/DeGaulle150.png'
  },
  {
    name: 'Elizabeth',
    img: 'static/Elizabeth150.png'
  },
  {
    name: 'Gandhi',
    img: 'static/Gandhi150.png'
  },
  {
    name: 'JohnPaul',
    img: 'static/JohnPaul150.png'
  },
  {
    name: 'Mao',
    img: 'static/Mao150.png'
  },
  {
    name: 'Marley',
    img: 'static/Marley150.png'
  },
  {
    name: 'Monroe',
    img: 'static/Monroe150.png'
  },
  {
    name: 'YokoOno',
    img: 'static/YokoOno150.png'
  },
  {
    name: 'Bhutto',
    img: 'static/Bhutto150.png'
  },
  {
    name: 'Churchill',
    img: 'static/Churchill150.png'
  },
  {
    name: 'DeGaulle',
    img: 'static/DeGaulle150.png'
  },
  {
    name: 'Elizabeth',
    img: 'static/Elizabeth150.png'
  },
  {
    name: 'Gandhi',
    img: 'static/Gandhi150.png'
  },
  {
    name: 'JohnPaul',
    img: 'static/JohnPaul150.png'
  },
  {
    name: 'Mao',
    img: 'static/Mao150.png'
  },
  {
    name: 'Marley',
    img: 'static/Marley150.png'
  },
  {
    name: 'Monroe',
    img: 'static/Monroe150.png'
  },
  {
    name: 'YokoOno',
    img: 'static/YokoOno150.png'
  }
]
