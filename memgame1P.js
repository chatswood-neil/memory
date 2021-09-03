// Scripts for memgame.html

const tMax = 20;
var grid;
var resultDisplay;
var tilesChosen = [];
var tilesChosenId = [];
var tilesWon = [];
var tileArray = [];

document.addEventListener("DOMContentLoaded", () => {
  grid = document.querySelector(".grid");
  resultDisplay = document.querySelector("#result");

  tileArray = [
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

  tileArray.sort(() => 0.5 - Math.random())
  createBoard()
})

// Board setup
function createBoard() {
  for (let i = 0; i < tMax; i++) {
    var newTile = document.createElement("img")
    newTile.setAttribute("src", "static/Tile1_150.png")
    newTile.setAttribute("data-id", i)
    newTile.addEventListener("click", flipTile)
    grid.appendChild(newTile)
  }
}

function checkForMatch() {
  var tiles = document.querySelectorAll("img")
  const optionOneId = tilesChosenId[0]
  const optionTwoId = tilesChosenId[1]
  console.log("optionOneId", optionOneId, "optionTwoId", optionTwoId)
  console.log("tilesChosen[0]", tilesChosen[0], "tilesChosen[1]", tilesChosen[1])
  if (tilesChosen[0] === tilesChosen[1]) {
    alert('You found a match')
    tiles[optionOneId].setAttribute("src", "static/Empty150.png")
    tiles[optionTwoId].setAttribute("src", "static/Empty150.png")
    tilesWon.push(tilesChosen)
  } else {
    alert('No match')
    console.log("no match")
    tiles[optionOneId].setAttribute("src", "static/Tile1_150.png")
    tiles[optionTwoId].setAttribute("src", "static/Tile1_150.png")
    console.log("reverted")
  }
  tilesChosen = []
  tilesChosenId = []
  resultDisplay.textContent = tilesWon.length
  if (tilesWon.length === tileArray.length/2) {
    resultDisplay.textContent = "Congrats"
  }
}

function flipTile() {
  var tileId = this.getAttribute("data-id")
  tilesChosen.push(tileArray[tileId].name)
  tilesChosenId.push(tileId)
  console.log("tilesChosenId", tilesChosenId)
  this.setAttribute("src", tileArray[tileId].img)
  if (tilesChosen.length === 2) {
    setTimeout(checkForMatch, 500)
  }
}
