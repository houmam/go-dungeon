package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DungeonConfig is the main params that are being set for the dungeon config
type DungeonConfig struct {
	width  int
	height int
}

// DungeonRoomsConfig is the params set for dungeon room configuration
type DungeonRoomsConfig struct {
	roomAttempts int
	minRoomSize  int
	maxRoomSize  int
	pixelSize    int
}

type Material int

const (
	WALL Material = iota
	FLOOR
	DOOR
	TUNNEL
)

type Point struct {
	x int
	y int
}

type Tile struct {
	region   int
	material Material
}

type Room struct {
	width    int
	height   int
	location Point
	edges    []Point
}

type Dungeon struct {
	tiles      [][]Tile
	rooms      []Room
	width      int
	height     int
	numRegions int
}

func createEmptyDungeon(width int, height int) Dungeon {
	fmt.Println("Creating empty dungeon...")

	dungeon := Dungeon{width: width, height: height}

	dungeon.tiles = make([][]Tile, height)

	for i := range dungeon.tiles {
		dungeon.tiles[i] = make([]Tile, width)
	}

	return dungeon
}

func createRooms(dungeon Dungeon, minSize, maxSize, attempts int) Dungeon {
	fmt.Println("Creating rooms...")

	var rooms []Room

	for i := 0; i < attempts; i++ {
		width := rand.Intn(maxSize-minSize) + minSize
		height := rand.Intn(maxSize-minSize) + minSize

		maxX := dungeon.width - width - 2
		maxY := dungeon.height - height - 2

		x := rand.Intn(maxX-3) + 3
		y := rand.Intn(maxY-3) + 3

		shouldAppend := true

		for r := range rooms {
			if x+width < rooms[r].location.x || // to the left
				x > rooms[r].location.x+rooms[r].width || // to the right
				y+height < rooms[r].location.y || // fully above
				y > rooms[r].location.y+rooms[r].height { // fully below
				// do nothing
			} else {
				shouldAppend = false
				break
			}
		}

		if shouldAppend {
			rooms = append(rooms, Room{width: width, height: height, location: Point{x: x, y: y}})
		}
	}

	for r := range rooms {
		dungeon.numRegions++

		for i := rooms[r].location.x; i < rooms[r].location.x+rooms[r].width; i++ {
			for j := rooms[r].location.y; j < rooms[r].location.y+rooms[r].height; j++ {
				dungeon.tiles[j][i].material = FLOOR
				dungeon.tiles[j][i].region = dungeon.numRegions
			}
		}
	}

	dungeon.rooms = rooms

	return dungeon
}

func createMaze(dungeon Dungeon) Dungeon {
	fmt.Println("Creating tunnels...")

	for x := 1; x < dungeon.width-1; x++ {
		for y := 1; y < dungeon.height-1; y++ {
			if dungeon.tiles[y-1][x-1].material == WALL &&
				dungeon.tiles[y][x-1].material == WALL &&
				dungeon.tiles[y+1][x-1].material == WALL &&
				dungeon.tiles[y-1][x].material == WALL &&
				dungeon.tiles[y][x].material == WALL &&
				dungeon.tiles[y+1][x].material == WALL &&
				dungeon.tiles[y-1][x+1].material == WALL &&
				dungeon.tiles[y][x+1].material == WALL &&
				dungeon.tiles[y+1][x+1].material == WALL {
				dungeon.numRegions++
				continueMaze(dungeon, x, y)
			}
		}
	}

	return dungeon
}

func lookDown(dungeon Dungeon, x int, y int) bool {
	if dungeon.tiles[y][x-2].material == WALL &&
		dungeon.tiles[y+1][x-2].material == WALL &&
		dungeon.tiles[y-1][x-2].material == WALL &&
		dungeon.tiles[y+1][x-1].material == WALL &&
		dungeon.tiles[y-1][x-1].material == WALL {
		return true
	}

	return false
}

func lookUp(dungeon Dungeon, x int, y int) bool {
	if dungeon.tiles[y][x+2].material == WALL &&
		dungeon.tiles[y-1][x+2].material == WALL &&
		dungeon.tiles[y+1][x+2].material == WALL &&
		dungeon.tiles[y+1][x+1].material == WALL &&
		dungeon.tiles[y-1][x+1].material == WALL {
		return true
	}

	return false
}

func lookLeft(dungeon Dungeon, x int, y int) bool {
	// Check Left
	if dungeon.tiles[y-2][x].material == WALL &&
		dungeon.tiles[y-2][x-1].material == WALL &&
		dungeon.tiles[y-2][x+1].material == WALL &&
		dungeon.tiles[y-1][x-1].material == WALL &&
		dungeon.tiles[y-1][x+1].material == WALL {
		return true
	}

	return false
}

func lookRight(dungeon Dungeon, x int, y int) bool {
	if dungeon.tiles[y+2][x].material == WALL &&
		dungeon.tiles[y+2][x-1].material == WALL &&
		dungeon.tiles[y+2][x+1].material == WALL &&
		dungeon.tiles[y+1][x-1].material == WALL &&
		dungeon.tiles[y+1][x+1].material == WALL {
		return true
	}

	return false
}

func getValidTiles(dungeon Dungeon, x int, y int) []Point {
	validTiles := []Point{}

	if x-2 >= 0 && dungeon.tiles[y][x-1].material == WALL {
		if lookDown(dungeon, x, y) {
			validTiles = append(validTiles, Point{y: y, x: x - 1})
		}
	}

	if x+2 < dungeon.width && dungeon.tiles[y][x+1].material == WALL {
		if lookUp(dungeon, x, y) {
			validTiles = append(validTiles, Point{y: y, x: x + 1})
		}
	}

	if y-2 >= 0 && dungeon.tiles[y-1][x].material == WALL {
		if lookLeft(dungeon, x, y) {
			validTiles = append(validTiles, Point{y: y - 1, x: x})
		}
	}

	if y+2 < dungeon.height && dungeon.tiles[y+1][x].material == WALL {
		if lookRight(dungeon, x, y) {
			validTiles = append(validTiles, Point{y: y + 1, x: x})
		}
	}

	return validTiles
}

func continueMaze(dungeon Dungeon, x int, y int) {
	validTiles := getValidTiles(dungeon, x, y)

	if len(validTiles) > 1 {
		i := rand.Intn(len(validTiles))
		point := validTiles[i]
		dungeon.tiles[point.y][point.x].material = TUNNEL
		dungeon.tiles[point.y][point.x].region = dungeon.numRegions

		continueMaze(dungeon, point.x, point.y)
		continueMaze(dungeon, x, y)
	} else if len(validTiles) == 1 {
		point := validTiles[0]
		dungeon.tiles[point.y][point.x].material = TUNNEL
		dungeon.tiles[point.y][point.x].region = dungeon.numRegions

		continueMaze(dungeon, point.x, point.y)
		continueMaze(dungeon, x, y)
	}
}

func identifyEdges(dungeon Dungeon) Dungeon {
	fmt.Println("Identifying edges...")

	for i := range dungeon.rooms {
		x := dungeon.rooms[i].location.x
		y := dungeon.rooms[i].location.y

		for j := x; j < x+dungeon.rooms[i].width; j++ {
			if dungeon.tiles[y-2][j].material == TUNNEL ||
				dungeon.tiles[y-2][j].material == FLOOR {
				dungeon.rooms[i].edges = append(dungeon.rooms[i].edges, Point{x: j, y: y - 1})
			}

			if dungeon.tiles[y+dungeon.rooms[i].height+1][j].material == TUNNEL ||
				dungeon.tiles[y+dungeon.rooms[i].height+1][j].material == FLOOR {
				dungeon.rooms[i].edges = append(dungeon.rooms[i].edges, Point{x: j, y: y + dungeon.rooms[i].height})
			}
		}

		for k := y; k < y+dungeon.rooms[i].height; k++ {
			if dungeon.tiles[k][x-2].material == TUNNEL ||
				dungeon.tiles[k][x-2].material == FLOOR {
				dungeon.rooms[i].edges = append(dungeon.rooms[i].edges, Point{x: x - 1, y: k})
			}

			if dungeon.tiles[k][x+dungeon.rooms[i].width+1].material == TUNNEL ||
				dungeon.tiles[k][x+dungeon.rooms[i].width+1].material == FLOOR {
				dungeon.rooms[i].edges = append(dungeon.rooms[i].edges, Point{x: x + dungeon.rooms[i].width, y: k})
			}
		}
	}

	return dungeon
}

func connectRegions(dungeon Dungeon) Dungeon {
	fmt.Println("Conneting regions...")

	for i := range dungeon.rooms {
		room := dungeon.rooms[i]
		edge := room.edges[rand.Intn(len(dungeon.rooms[i].edges))]
		roomRegion := dungeon.tiles[dungeon.rooms[i].location.y][dungeon.rooms[i].location.x].region

		// check if edge is unconnected
		surroundingTiles := [8]Tile{
			dungeon.tiles[edge.y-1][edge.x-1],
			dungeon.tiles[edge.y-1][edge.x],
			dungeon.tiles[edge.y-1][edge.x+1],
			dungeon.tiles[edge.y][edge.x-1],
			dungeon.tiles[edge.y][edge.x+1],
			dungeon.tiles[edge.y+1][edge.x-1],
			dungeon.tiles[edge.y+1][edge.x],
			dungeon.tiles[edge.y+1][edge.x+1],
		}

		for j := range surroundingTiles {
			if (surroundingTiles[j].material == FLOOR || surroundingTiles[j].material == TUNNEL) &&
				surroundingTiles[j].region != roomRegion {
				dungeon.tiles[edge.y][edge.x].material = DOOR

				for x := room.location.x; x < room.location.x+room.width; x++ {
					for y := room.location.y; y < room.location.y+room.height; y++ {
						dungeon.tiles[y][x].region = surroundingTiles[j].region
					}
				}

				break
			}
		}
	}

	// go through the rooms and their edges in random order
	// to see if any of them are still a separate region
	connectedRegions := map[int]bool{}
RoomsLoop:
	for i := range rand.Perm(len(dungeon.rooms)) {
		for j := range rand.Perm(len(dungeon.rooms[i].edges)) {
			room := dungeon.rooms[i]
			edge := room.edges[j]
			x := edge.x
			y := edge.y

			surroundingPoints := [4]Point{
				{x: x - 1, y: y},
				{x: x + 1, y: y},
				{x: x, y: y - 1},
				{x: x, y: y + 1},
			}

			curRegion := -1
			for k := range surroundingPoints {
				tile := dungeon.tiles[surroundingPoints[k].y][surroundingPoints[k].x]
				if curRegion == -1 && tile.region != 0 {
					curRegion = tile.region
				} else if tile.region != curRegion && tile.region != 0 && !connectedRegions[tile.region] {
					dungeon.tiles[y][x].material = DOOR
					connectedRegions[tile.region] = true
					connectedRegions[curRegion] = true

					continue RoomsLoop
				}
			}
		}
	}

	return dungeon
}

func trimTunnels(dungeon Dungeon) {
	fmt.Println("Trimming tunnels...")

	for x := 1; x < dungeon.width-1; x++ {
		for y := 1; y < dungeon.height-1; y++ {
			continueTrimTunnels(dungeon, x, y)
		}
	}
}

func continueTrimTunnels(dungeon Dungeon, x int, y int) {
	if dungeon.tiles[y][x].material == TUNNEL || dungeon.tiles[y][x].material == DOOR {
		wallCount := 0
		nextPoint := Point{}

		surroundingPoints := [4]Point{
			{x: x - 1, y: y},
			{x: x + 1, y: y},
			{x: x, y: y - 1},
			{x: x, y: y + 1},
		}

		for i := range surroundingPoints {
			tile := dungeon.tiles[surroundingPoints[i].y][surroundingPoints[i].x]
			if tile.material == WALL {
				wallCount++
			} else if tile.material == TUNNEL || tile.material == DOOR {
				nextPoint = Point{x: surroundingPoints[i].x, y: surroundingPoints[i].y}
			}
		}

		if wallCount >= 3 {
			dungeon.tiles[y][x].material = WALL
			dungeon.tiles[y][x].region = 0

			if nextPoint.x != 0 || nextPoint.y != 0 {
				continueTrimTunnels(dungeon, nextPoint.x, nextPoint.y)
			}
		}
	}
}

func renderDungeon(dungeon Dungeon) {
	fmt.Println("Dungeon: (", dungeon.width, ",", dungeon.height, ") Regions: ", dungeon.numRegions)

	for y := 0; y < dungeon.height; y++ {
		for x := 0; x < dungeon.width; x++ {
			switch dungeon.tiles[y][x].material {
			case WALL:
				fmt.Print("0 ")
			case FLOOR:
				fmt.Print("= ")
			case DOOR:
				fmt.Print("| ")
			case TUNNEL:
				fmt.Print("- ")
			default:
				fmt.Print("ER")
			}
		}

		fmt.Println()
	}
}

func generateTileMask(size int) image.Image {
	lightUniform := &image.Uniform{color.RGBA{255, 255, 255, 120}}
	mediumUniform := &image.Uniform{color.RGBA{255, 255, 255, 180}}
	darkUniform := &image.Uniform{color.RGBA{255, 255, 255, 220}}

	mask := image.NewRGBA(image.Rect(0, 0, size, size))

	// background color
	draw.Draw(
		mask,                         // dst image
		image.Rect(0, 0, size, size), // rectangle
		mediumUniform,                // src image
		image.ZP,                     // point
		draw.Src,                     // OP
	)

	// lighter lines
	draw.Draw(
		mask,                        // dst image
		image.Rect(0, 0, size-1, 1), // rectangle
		lightUniform,                // src image
		image.ZP,                    // point
		draw.Src,                    // OP
	)

	draw.Draw(
		mask,                        // dst image
		image.Rect(0, 1, 1, size-1), // rectangle
		lightUniform,                // src image
		image.ZP,                    // point
		draw.Src,                    // OP
	)

	// darker lines
	draw.Draw(
		mask,                                // dst image
		image.Rect(size-1, 1, size, size-1), // rectangle
		darkUniform,                         // src image
		image.ZP,                            // point
		draw.Src,                            // OP
	)

	draw.Draw(
		mask,                              // dst image
		image.Rect(1, size-1, size, size), // rectangle
		darkUniform,                       // src image
		image.ZP,                          // point
		draw.Src,                          // OP
	)

	return mask
}

func dungeonToImage(dungeon Dungeon, drConfig DungeonRoomsConfig) image.Image {
	mask := generateTileMask(drConfig.pixelSize)

	m := image.NewRGBA(image.Rect(0, 0, dungeon.width*drConfig.pixelSize, dungeon.height*drConfig.pixelSize))
	draw.Draw(
		m, // dst image
		image.Rect(0, 0, dungeon.width*drConfig.pixelSize, dungeon.height*drConfig.pixelSize), // rectangle
		&image.Uniform{color.RGBA{255, 255, 255, 255}},                                        // src image
		image.ZP,  // point
		draw.Over, // OP
	)

	for y := 0; y < dungeon.height; y++ {
		for x := 0; x < dungeon.width; x++ {
			var pixelColor color.RGBA

			switch dungeon.tiles[y][x].material {
			case WALL:
				pixelColor = color.RGBA{0, 0, 0, 255}
			case FLOOR:
				pixelColor = color.RGBA{128, 128, 128, 255}
			case DOOR:
				pixelColor = color.RGBA{150, 100, 0, 255}
			case TUNNEL:
				pixelColor = color.RGBA{200, 200, 200, 255}
			default:
				pixelColor = color.RGBA{255, 0, 0, 255}
			}

			draw.DrawMask(
				m, // dst image
				image.Rect(x*drConfig.pixelSize, y*drConfig.pixelSize,
					(x+1)*drConfig.pixelSize, (y+1)*drConfig.pixelSize), // rectangle
				&image.Uniform{pixelColor}, // src image
				image.ZP,                   // point
				mask,                       // mask image
				image.ZP,                   // mask point
				draw.Over,                  // OP
			)
		}
	}

	return m
}

func generateDungeon(dConfig DungeonConfig, drConfig DungeonRoomsConfig) Dungeon {
	dungeon := createEmptyDungeon(dConfig.width, dConfig.height)
	dungeon = createRooms(dungeon, drConfig.minRoomSize, drConfig.maxRoomSize, drConfig.roomAttempts)
	dungeon = createMaze(dungeon)
	dungeon = identifyEdges(dungeon)
	dungeon = connectRegions(dungeon)
	trimTunnels(dungeon)

	return dungeon
}

func parseIntOption(option []string, defaultValue int, min int, max int) int {
	if len(option) == 0 {
		return defaultValue
	}

	value, err := strconv.Atoi(option[0])

	if err != nil || value < min || value > max {
		return defaultValue
	}

	return value
}

func main() {
	serverFlag := flag.Bool("server", false, "Run as a server on port 8080 and serve PNG files")
	flag.Parse()

	if !*serverFlag {
		dConfig := DungeonConfig{
			width:  50,
			height: 50,
		}

		drConfig := DungeonRoomsConfig{
			roomAttempts: 200,
			minRoomSize:  5,
			maxRoomSize:  15,
			pixelSize:    10,
		}

		rand.Seed(time.Now().UTC().UnixNano())

		dungeon := generateDungeon(dConfig, drConfig)
		renderDungeon(dungeon)
	} else {
		fs := http.FileServer(http.Dir(""))
		http.Handle("/", fs)

		http.HandleFunc("/generate/", func(w http.ResponseWriter, r *http.Request) {
			query, _ := url.ParseQuery(r.URL.RawQuery)

			dConfig := DungeonConfig{}
			drConfig := DungeonRoomsConfig{}
			dConfig.width = parseIntOption(query["dungeonWidth"], 50, 20, 1000)
			dConfig.height = parseIntOption(query["dungeonHeight"], 50, 20, 1000)
			maxAllowedRoomSize := int(math.Min(float64(dConfig.width-2), float64(dConfig.height-2)))
			drConfig.roomAttempts = parseIntOption(query["roomAttempts"], 200, 1, 100000)
			drConfig.minRoomSize = parseIntOption(query["minRoomSize"], 5, 1, maxAllowedRoomSize)
			drConfig.maxRoomSize = parseIntOption(query["maxRoomSize"], drConfig.minRoomSize+1,
				drConfig.minRoomSize, maxAllowedRoomSize)
			drConfig.pixelSize = parseIntOption(query["pixelSize"], 10, 1, 20)

			if len(query["seed"]) != 0 {
				seed, err := strconv.ParseInt(query["seed"][0], 10, 64)
				if err == nil {
					rand.Seed(seed)
				} else {
					rand.Seed(time.Now().UTC().UnixNano())
				}
			} else {
				rand.Seed(time.Now().UTC().UnixNano())
			}

			dungeon := generateDungeon(dConfig, drConfig)

			if r.URL.Path == "/generate/json/" {
				tiles := make([][]int, dConfig.height)
				for i := 0; i < dConfig.height; i++ {
					tiles[i] = make([]int, dConfig.width)
					for j := 0; j < dConfig.width; j++ {
						tiles[i][j] = int(dungeon.tiles[i][j].material)
					}
				}

				dungeonJSON, _ := json.Marshal(tiles)
				fmt.Fprintf(w, "%s", string(dungeonJSON))
			} else {
				w.Header().Set("Content-Type", "image/png")
				_ = png.Encode(w, dungeonToImage(dungeon, drConfig))
			}
		})

		_ = http.ListenAndServe(":8080", nil)
	}
}
