// TODO:
// - randomize seed each run
// - ship respawn
// - good collision detection
// - sound effects
// - saucers
// - remaining lives indicator
// - new graphics
// - center score
// - high score
// - smaller = faster
// - timing variability
// - firehose bug
// - explosions

package main

import (
	"fmt"
	"image"
	"math"
	"math/rand"
	"os"
	"time"

	_ "image/png"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
)

func run() {
	bounds := pixel.R(0, 0, 1024, 768)

	cfg := pixelgl.WindowConfig{
		Title:  "Go Rocks!",
		Bounds: bounds,
		VSync:  true,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	treesheet, treesheetImage, err := loadPicture("trees.png")
	if err != nil {
		panic(err)
	}

	var treeFrames []pixel.Rect
	for x := treesheet.Bounds().Min.X; x < treesheet.Bounds().Max.X; x += 32 {
		for y := treesheet.Bounds().Min.Y; y < treesheet.Bounds().Max.Y; y += 32 {
			treeFrames = append(treeFrames, pixel.R(x, y, x+32, y+32))
		}
	}

	// TODO: do something with or remove these
	camPos := pixel.ZV
	camZoom := 1.0

	// The stage's origin 0,0 is at its center.
	stageBounds := bounds.Moved(pixel.V(-bounds.W()/2, -bounds.H()/2))
	stage := MakeStage(Stage{win: win, bounds: stageBounds, spritesheet: treesheet,
		spritesheetImage: treesheetImage, frames: treeFrames})

	game := makeGame(&stage)

	last := time.Now()

	for !win.Closed() {
		dt := time.Since(last).Seconds()
		last = time.Now()

		cam := pixel.IM.Scaled(camPos, camZoom).Moved(win.Bounds().Center().Sub(camPos))
		win.SetMatrix(cam)

		// Clear window to the background color.
		// TODO: move this to Stage?
		win.Clear(colornames.Black)

		game.update(dt)

		// Update the display and wait for the next frame.
		win.Update()
	}
}

type Game struct {
	stage *Stage
	level int
	ships int
	score int

	largeRockPoints  int
	mediumRockPoints int
	smallRockPoints  int
	numberOfShips    int
}

func makeGame(stage *Stage) Game {
	g := Game{stage: stage, largeRockPoints: 20, mediumRockPoints: 50, smallRockPoints: 100, numberOfShips: 3}
	g.reset()
	return g
}

func (g *Game) reset() {
	g.stage.reset()

	// TODO: wait for the area near the ship to be clear before spawning
	makeShip(g)

	makeScore(g)
	g.ships = g.numberOfShips
	g.score = 0
	g.newLevel(1)
}

func (g *Game) newLevel(level int) {
	g.level = level
	for i := 0; i < level; i++ {
		makeRock(g.stage, 1, nil)
	}
}

func (g *Game) update(dt float64) {
	// Press r to reset the game.
	if g.stage.win.Pressed(pixelgl.KeyR) {
		g.reset()
	}

	// TODO: If the ship has been destroyed start over.

	// If all the rocks have been destroyed go to the next level.
	// TODO: if stage.findActorsByKind("rock") == nil
	noRocks := true
	for _, actor := range g.stage.actors {
		if actor.Kind() == "rock" {
			noRocks = false
			break
		}
	}

	if noRocks {
		g.newLevel(g.level + 1)
	}

	// Give every actor a chance to update.
	g.stage.update(dt)

	// Ask every actor to draw.
	g.stage.draw()
}

//

type WrapAroundActor struct {
	SpriteActor
}

func makeWrapAroundActor(frame int, stage *Stage, kind string) WrapAroundActor {
	return WrapAroundActor{SpriteActor: MakeSpriteActor(frame, stage, kind)}
}

func (a *WrapAroundActor) Update(dt float64) {
	a.BaseActor.Update(dt)
	wrapAroundVec(&a.position, &a.stage.bounds)
}

//

type Score struct {
	TextActor
	game *Game // TODO: retain game instead of stage in all actors?
}

func makeScore(game *Game) Score {
	stage := game.stage
	s := Score{TextActor: MakeTextActor(pixel.V(0, stage.bounds.Max.Y-30), stage), game: game}
	s.scale = 2
	s.horizontalAlignment = "center"

	stage.addActor(&s)
	return s
}

func (a *Score) Update(dt float64) {
	a.SetText(fmt.Sprintf("%d", a.game.score))
	a.TextActor.Update(dt)
}

//

type Ship struct {
	WrapAroundActor
	game         *Game
	acceleration float64
	rotateSpeed  float64
	fireCooldown float64
}

// TODO: why is it oriented to the right?
func makeShip(game *Game) Ship {
	stage := game.stage
	s := Ship{
		WrapAroundActor: makeWrapAroundActor(8, stage, "ship"),
		acceleration:    10.0,
		rotateSpeed:     5.0,
		fireCooldown:    0.0}
	s.game = game
	s.scale = 1.5

	stage.addActor(&s)
	return s
}

func (s *Ship) Update(dt float64) {
	stage := s.stage
	win := stage.win

	s.fireCooldown -= dt

	if win.Pressed(pixelgl.KeyA) {
		s.rotateLeft(dt)
	}

	if win.Pressed(pixelgl.KeyD) {
		s.rotateRight(dt)
	}

	if win.Pressed(pixelgl.KeyW) {
		s.thrust(dt)
	}

	if s.fireCooldown <= 0.0 && win.Pressed(pixelgl.KeyS) || win.Pressed(pixelgl.KeySpace) {
		// Limit the firing rate.
		s.fireCooldown = 0.1

		vector := pixel.Unit(s.rotation + math.Pi/2)
		position := s.position.Add(vector.Scaled(25))
		velocity := s.velocity.Add(vector.Scaled(5))
		makeShot(position, velocity, stage, s.game)
	}

	s.WrapAroundActor.Update(dt)

	// Check for collision with a rock.
	for _, actor := range stage.actors {
		if actor.Kind() == "rock" && intersects(s, actor) {
			stage.removeActor(s)
			stage.removeActor(actor)

			// TODO: explode ship, rock
			break
		}
	}
}

func (s *Ship) thrust(dt float64) {
	s.velocity = s.velocity.Add(pixel.Unit(s.rotation + math.Pi/2).Scaled(s.acceleration * dt))
}

func (s *Ship) rotateLeft(dt float64) {
	s.rotation += s.rotateSpeed * dt
}

func (s *Ship) rotateRight(dt float64) {
	s.rotation -= s.rotateSpeed * dt
}

//

type Rock struct {
	WrapAroundActor
	generation int
}

func makeRock(stage *Stage, generation int, parent *Rock) *Rock {
	frame := rand.Intn(8)
	rock := Rock{WrapAroundActor: makeWrapAroundActor(frame, stage, "rock"), generation: generation}
	if parent != nil {
		// TODO: something better
		picture := rock.SpriteActor.sprite.Picture()
		rock.SpriteActor.sprite.Set(picture, parent.SpriteActor.sprite.Frame())
	}

	// Scale the rock according to its generation.
	rock.scale = []float64{5.0, 3.0, 1.5}[generation-1]

	// Pick a random spin direction.
	rock.rotationVelocity = 0.5
	if rand.Float32() < 0.5 {
		rock.rotationVelocity = -0.5
	}

	// Pick a random orentation.
	angle := (math.Pi * 2) * rand.Float64()
	rock.velocity = pixel.Unit(angle)

	// Pick a random position.
	// TODO: not cool to spawn on top or close to the ship
	w := int(stage.bounds.W())
	h := int(stage.bounds.H())
	x := float64(rand.Intn(w) - w/2)
	y := float64(rand.Intn(h) - h/2)
	rock.position = pixel.V(x, y)

	stage.addActor(&rock)
	return &rock
}

//

type Shot struct {
	WrapAroundActor
	game    *Game
	timeout float64
}

func makeShot(position pixel.Vec, velocity pixel.Vec, stage *Stage, game *Game) *Shot {
	s := Shot{WrapAroundActor: makeWrapAroundActor(6, stage, "shot"), timeout: 1.5}
	s.position = position
	s.velocity = velocity
	s.scale = 0.4
	s.game = game

	stage.addActor(&s)
	return &s
}

func (s *Shot) Update(dt float64) {
	stage := s.stage

	s.timeout -= dt
	if s.timeout < 0 {
		stage.removeActor(s)
		return
	}

	// Check for collision with a rock.
	actors := stage.actors
	for _, actor := range actors {
		if intersects(actor, s) && actor.Kind() == "rock" {
			stage.removeActor(s)
			stage.removeActor(actor)

			rock := actor.(*Rock)

			points := []int{s.game.largeRockPoints, s.game.mediumRockPoints, s.game.smallRockPoints}
			s.game.score += points[rock.generation-1]

			// TODO: explode rock

			// Create two smaller rocks.
			if rock.generation < 3 {
				var newRock *Rock
				for i := 0; i < 2; i++ {
					fmt.Println("makeRock", i)
					newRock = makeRock(stage, rock.generation+1, rock)
					newRock.position = actor.Position()
				}
			}
			break
		}
	}

	s.BaseActor.Update(dt)
}

func wrapAroundVec(vec *pixel.Vec, bounds *pixel.Rect) {
	if vec.X < bounds.Min.X {
		vec.X = bounds.Max.X
	}
	if vec.X > bounds.Max.X {
		vec.X = bounds.Min.X
	}
	if vec.Y < bounds.Min.Y {
		vec.Y = bounds.Max.Y
	}
	if vec.Y > bounds.Max.Y {
		vec.Y = bounds.Min.Y
	}
}

func loadPicture(path string) (pixel.Picture, image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, nil, err
	}
	return pixel.PictureDataFromImage(img), img, nil
}

func main() {
	pixelgl.Run(run)
}