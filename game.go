package main

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
)

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

func makeGame(stage *Stage) *Game {
	g := Game{stage: stage, largeRockPoints: 20, mediumRockPoints: 50, smallRockPoints: 100, numberOfShips: 4}
	g.reset()

	// We must return a pointer to Game now that it has been initialized with Actors that reference it.
	// Be aware that returning a local var actually returns a copy of it!
	return &g
}

func (g *Game) reset() {
	g.stage.reset()

	g.ships = g.numberOfShips
	g.score = 0

	makeScore(g)
	makeLives(g)
	g.newLevel(1)
}

func (g *Game) newLevel(level int) {
	g.level = level
	for i := 0; i < level; i++ {
		makeRock(g.stage, 1, nil)
	}
}

func (g *Game) update(dt float64) {
	stage := g.stage

	// Press r to reset the game.
	if stage.win.Pressed(pixelgl.KeyR) {
		g.reset()
	}

	// If the ship has been destroyed spawn a new one until all are gone.
	if stage.findActorsByKind("ship") == nil {
		g.ships--
		if g.ships > 0 {
			// TODO: wait for the area near the ship to be clear before spawning
			makeShip(g)
		} else {
			// TODO: game over
			g.reset()
		}
	}

	// If all rocks have been destroyed go to the next level.
	if stage.findActorsByKind("rock") == nil {
		g.newLevel(g.level + 1)
	}

	// Give every actor a chance to update.
	stage.update(dt)

	// Ask every actor to draw.
	stage.draw()
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

/* TODO: If striding the boundary draw on both sides.
func (a *WrapAroundActor) Draw() {
}
*/

//

type Score struct {
	TextActor
	game *Game // TODO: retain game instead of stage in all actors?
}

func makeScore(game *Game) *Score {
	stage := game.stage
	s := Score{TextActor: MakeTextActor(pixel.V(0, stage.bounds.Max.Y-30), stage), game: game}
	s.scale = 2
	s.horizontalAlignment = "center"

	stage.addActor(&s)
	return &s
}

func (a *Score) Update(dt float64) {
	a.SetText(fmt.Sprintf("%v", a.game.score))
	a.TextActor.Update(dt)
}

//

type Lives struct {
	BaseActor
	game *Game
}

func makeLives(game *Game) *Lives {
	stage := game.stage
	l := Lives{BaseActor: MakeBaseActor(stage, "lives"), game: game}
	l.position = pixel.V(stage.bounds.Min.X+20, stage.bounds.Max.Y-25)

	stage.addActor(&l)
	return &l
}

func (a *Lives) Draw() {
	ships := a.stage.findActorsByKind("ship")
	if len(ships) == 0 {
		return
	}
	ship := ships[0].(*Ship)
	for i := 0; i < a.game.ships; i++ {
		ship.sprite.Draw(a.stage.win, a.Transform().Moved(pixel.V(float64(i)*30.0, 0)))
	}
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
func makeShip(game *Game) *Ship {
	stage := game.stage
	s := Ship{
		WrapAroundActor: makeWrapAroundActor(8, stage, "ship"),
		acceleration:    10.0,
		rotateSpeed:     5.0,
		fireCooldown:    0.0,
		game:            game}
	s.scale = 1.5

	stage.addActor(&s)
	return &s
}

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
	s := Shot{WrapAroundActor: makeWrapAroundActor(6, stage, "shot"), timeout: 1.5, game: game}
	s.position = position
	s.velocity = velocity
	s.scale = 0.4

	stage.addActor(&s)
	return &s
}

func (s *Shot) Update(dt float64) {
	game := s.game
	stage := s.stage

	s.timeout -= dt
	if s.timeout < 0 {
		stage.removeActor(s)
		return
	}

	s.WrapAroundActor.Update(dt)

	// Check for collision with a rock.
	actors := stage.actors
	for _, actor := range actors {
		if actor.Kind() == "rock" && intersects(actor, s) {
			rock := actor.(*Rock)
			points := []int{game.largeRockPoints, game.mediumRockPoints, game.smallRockPoints}
			game.score += points[rock.generation-1]

			stage.removeActor(s)
			stage.removeActor(actor)

			// TODO: explode rock

			// Create two smaller rocks.
			if rock.generation < 3 {
				for i := 0; i < 2; i++ {
					newRock := makeRock(stage, rock.generation+1, rock)
					newRock.position = actor.Position()
				}
			}
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
