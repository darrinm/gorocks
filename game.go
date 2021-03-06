package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
)

// Game is the root of all game state and implements the game logic.
type Game struct {
	stage *Stage
	level int
	lives int
	score int

	largeRockPoints  int
	mediumRockPoints int
	smallRockPoints  int
	numberOfLives    int
	newShipPoints    int

	heldKeys      map[pixelgl.Button]bool
	previousScore int
}

func makeGame(stage *Stage) *Game {
	// Get fresh random numbers every run.
	rand.Seed(time.Now().Unix())

	g := Game{stage: stage, heldKeys: make(map[pixelgl.Button]bool),
		largeRockPoints: 20, mediumRockPoints: 50, smallRockPoints: 100, newShipPoints: 10000, numberOfLives: 4,
	}
	g.reset()

	// We must return a pointer to Game now that it has been initialized with Actors that reference it.
	// Be aware that returning a local var actually returns a copy of it!
	return &g
}

func (g *Game) reset() {
	g.stage.Reset()

	g.lives = g.numberOfLives
	g.score = 0

	makeScore(g)
	makeLives(g)
	g.newLevel(1)
}

func (g *Game) newLevel(level int) {
	g.level = level
	for i := 0; i < level; i++ {
		makeRock(g, 1, nil)
	}
}

func (g *Game) update(dt float64) {
	stage := g.stage

	// Press r to reset the game.
	if stage.win.Pressed(pixelgl.KeyR) {
		if !g.heldKeys[pixelgl.KeyR] {
			g.heldKeys[pixelgl.KeyR] = true
			g.reset()
		}
	} else {
		g.heldKeys[pixelgl.KeyR] = false
	}

	// Press b to toggle Actor bounds drawing.
	if stage.win.Pressed(pixelgl.KeyB) {
		if !g.heldKeys[pixelgl.KeyB] {
			g.heldKeys[pixelgl.KeyB] = true

			g.stage.drawActorBounds = !g.stage.drawActorBounds
		}
	} else {
		g.heldKeys[pixelgl.KeyB] = false
	}

	// Press p to add 1,000 points to the score.
	if stage.win.Pressed(pixelgl.KeyP) {
		if !g.heldKeys[pixelgl.KeyP] {
			g.heldKeys[pixelgl.KeyP] = true

			g.score += 1000
		}
	} else {
		g.heldKeys[pixelgl.KeyP] = false
	}

	// If the ship has been destroyed spawn a new one until all are gone.
	if stage.FindActorsByKind("ship") == nil {
		g.lives--
		if g.lives > 0 {
			// TODO: wait for the area near the ship to be clear before spawning
			makeShip(g)
		} else {
			// TODO: game over
			g.reset()
		}
	}

	// If all rocks have been destroyed go to the next level.
	if stage.FindActorsByKind("rock") == nil {
		g.newLevel(g.level + 1)
	}

	// If the player has crossed a scoring threshold give them another ship.
	if g.previousScore%g.newShipPoints > g.score%g.newShipPoints {
		g.lives++
	}

	g.previousScore = g.score

	// Give every actor a chance to update.
	stage.Update(dt)

	// Ask every actor to draw.
	stage.Draw()
}

// WrapAroundActor upgrades SpriteActors to wrap around screen edges when they move off them.
type WrapAroundActor struct {
	SpriteActor
}

func makeWrapAroundActor(frame int, stage *Stage, kind string) WrapAroundActor {
	return WrapAroundActor{SpriteActor: MakeSpriteActor(frame, stage, kind)}
}

// Update brings the Actor back on the opposite side of the screen from where it exited.
func (a *WrapAroundActor) Update(dt float64) {
	a.BaseActor.Update(dt)
	wrapAroundVec(&a.position, &a.stage.bounds)
}

/* TODO: If striding the boundary draw on both sides.
func (a *WrapAroundActor) Draw() {
}
*/

// Score displays the current game score.
type Score struct {
	TextActor
	game *Game // TODO: retain game instead of stage in all actors?
}

func makeScore(game *Game) *Score {
	stage := game.stage
	s := Score{TextActor: MakeTextActor(pixel.V(0, stage.bounds.Max.Y-30), stage), game: game}
	s.scale = 2
	s.horizontalAlignment = "center"

	stage.AddActor(&s)
	return &s
}

// Update the Score's TextActor with the current game score.
func (a *Score) Update(dt float64) {
	a.SetText(fmt.Sprintf("%v", a.game.score))
	a.TextActor.Update(dt)
}

// Lives displays how many lives the player has left.
type Lives struct {
	BaseActor
	game *Game
}

func makeLives(game *Game) *Lives {
	stage := game.stage
	l := Lives{BaseActor: MakeBaseActor(stage, "lives"), game: game}
	l.position = pixel.V(stage.bounds.Min.X+20, stage.bounds.Max.Y-25)

	stage.AddActor(&l)
	return &l
}

// Draw a representation of the number of lives the player currently has.
func (a *Lives) Draw() {
	// Reuse the sprite the Ship object has.
	ships := a.stage.FindActorsByKind("ship")
	if len(ships) == 0 {
		return
	}
	ship := ships[0].(*Ship)
	for i := 0; i < a.game.lives; i++ {
		ship.sprite.Draw(a.stage.win, a.Transform().Moved(pixel.V(float64(i)*30.0, 0)))
	}
}

// Ship is the hero. It handles the UI for the player ship.
type Ship struct {
	WrapAroundActor
	game         *Game
	acceleration float64
	rotateSpeed  float64
	fireCooldown float64
}

func makeShip(game *Game) *Ship {
	stage := game.stage
	s := Ship{
		WrapAroundActor: makeWrapAroundActor(8, stage, "ship"),
		acceleration:    10.0,
		rotateSpeed:     5.0,
		fireCooldown:    0.0,
		game:            game}
	s.scale = 1.5

	stage.AddActor(&s)
	return &s
}

// Update responds to player input for moving and firing.
// It also handles collision detection and response.
func (s *Ship) Update(dt float64) {
	stage := s.stage
	win := stage.win

	s.fireCooldown -= dt

	if win.Pressed(pixelgl.KeyA) || win.Pressed(pixelgl.KeyLeft) {
		s.rotateLeft(dt)
	}

	if win.Pressed(pixelgl.KeyD) || win.Pressed(pixelgl.KeyRight) {
		s.rotateRight(dt)
	}

	if win.Pressed(pixelgl.KeyW) || win.Pressed(pixelgl.KeyUp) {
		s.thrust(dt)
	}

	if s.fireCooldown <= 0.0 && (win.Pressed(pixelgl.KeyS) || win.Pressed(pixelgl.KeyDown) || win.Pressed(pixelgl.KeySpace)) {
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
			stage.RemoveActor(s)

			// TODO: explode ship

			rock := actor.(*Rock)
			rock.subdivide()
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

// Rock is the primary antagonist.
type Rock struct {
	WrapAroundActor
	generation int
	game       *Game
}

func makeRock(game *Game, generation int, parent *Rock) *Rock {
	stage := game.stage
	frame := rand.Intn(8)
	rock := Rock{WrapAroundActor: makeWrapAroundActor(frame, stage, "rock"), generation: generation, game: game}
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

	stage.AddActor(&rock)
	return &rock
}

func (r *Rock) subdivide() {
	game := r.game
	stage := r.stage

	points := []int{game.largeRockPoints, game.mediumRockPoints, game.smallRockPoints}
	game.score += points[r.generation-1]

	stage.RemoveActor(r)

	// TODO: explode rock

	// Create two smaller rocks.
	if r.generation < 3 {
		for i := 0; i < 2; i++ {
			newRock := makeRock(game, r.generation+1, r)
			newRock.position = r.Position()
		}
	}
}

// Shot is the ship's shot. It handles collision detection and response.
type Shot struct {
	WrapAroundActor
	game    *Game
	timeout float64
}

func makeShot(position pixel.Vec, velocity pixel.Vec, stage *Stage, game *Game) *Shot {
	s := Shot{WrapAroundActor: makeWrapAroundActor(6, stage, "shot"), timeout: 1.5, game: game}
	s.position = position
	s.velocity = velocity
	s.scale = 0.4
	s.rotation = velocity.Angle()

	stage.AddActor(&s)
	return &s
}

// Update handles shot-rock collision detection and response.
func (s *Shot) Update(dt float64) {
	stage := s.stage

	s.timeout -= dt
	if s.timeout < 0 {
		stage.RemoveActor(s)
		return
	}

	s.WrapAroundActor.Update(dt)

	// Check for collision with a rock.
	actors := stage.actors
	for _, actor := range actors {
		if actor.Kind() == "rock" && intersects(actor, s) {
			stage.RemoveActor(s)

			rock := actor.(*Rock)
			rock.subdivide()
			break
		}
	}
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
