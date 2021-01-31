package main

import (
	"fmt"
	"math"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/text"
)

//

type Actor interface {
	// Meant to be overridden.
	Update(dt float64)
	Draw()

	// Also sometimes useful to override.
	Bounds() pixel.Rect
	ScaledBounds() pixel.Rect
	// TODO: CollisionPolygon?

	// Needed by Stage.
	Id() int
	Kind() string
	Position() pixel.Vec
	Scale() float64
	Rotation() float64
	Transform() pixel.Matrix

	// TODO: So Stage can verify Actors are mounted.
	Stage() *Stage
	SetStage(staget *Stage)
}

type BaseActor struct {
	id               int
	stage            *Stage
	kind             string
	position         pixel.Vec
	rotation         float64
	scale            float64
	velocity         pixel.Vec
	rotationVelocity float64
	// TODO: a way to control ordering such that e.g. text is can always be on top
	// layer int
}

var nextId = 1

func MakeBaseActor(stage *Stage, kind string) BaseActor {
	id := nextId
	nextId++
	return BaseActor{
		id:               id,
		stage:            stage,
		scale:            1,
		position:         pixel.ZV,
		rotation:         0.0,
		velocity:         pixel.ZV,
		rotationVelocity: 0.0,
		kind:             kind}
}

func (a *BaseActor) Id() int {
	return a.id
}

// TODO: here because I couldn't figure out how to actor.(*BaseActor).stage
func (a *BaseActor) Stage() *Stage {
	return a.stage
}

// TODO: sucks to have to expose this
func (a *BaseActor) SetStage(stage *Stage) {
	a.stage = stage
}

func (a *BaseActor) Kind() string {
	return a.kind
}

func (a *BaseActor) Position() pixel.Vec {
	return a.position
}

func (a *BaseActor) Scale() float64 {
	return a.scale
}

func (a *BaseActor) Rotation() float64 {
	return a.rotation
}

func (a *BaseActor) Transform() pixel.Matrix {
	return pixel.IM.Scaled(pixel.ZV, a.scale).Rotated(pixel.ZV, a.rotation).Moved(a.position)
}

func (a *BaseActor) Bounds() pixel.Rect {
	return pixel.Rect{Min: a.position, Max: a.position}
}

func (a *BaseActor) ScaledBounds() pixel.Rect {
	return pixel.Rect{Min: a.position, Max: a.position}
}

func (a *BaseActor) Update(dt float64) {
	// TODO: dt
	a.position = a.position.Add(a.velocity)
	a.rotation += a.rotationVelocity * dt
}

func (a *BaseActor) Draw() {
	// This space intentially left blank.
}

type SpriteActor struct {
	BaseActor
	sprite *pixel.Sprite
}

func MakeSpriteActor(frame int, stage *Stage, kind string) SpriteActor {
	return SpriteActor{
		sprite:    pixel.NewSprite(stage.spritesheet, stage.frames[frame]),
		BaseActor: MakeBaseActor(stage, kind),
	}
}

func (a *SpriteActor) Bounds() pixel.Rect {
	halfW := a.sprite.Frame().W() / 2
	halfH := a.sprite.Frame().H() / 2
	return pixel.R(-halfW, -halfH, halfW, halfH)
}

func (a *SpriteActor) ScaledBounds() pixel.Rect {
	w := a.sprite.Frame().W() * a.scale
	h := a.sprite.Frame().H() * a.scale
	offsetPosition := a.position.Add(pixel.V(-w/2, -h/2))
	return pixel.R(offsetPosition.X, offsetPosition.Y, offsetPosition.X+w, offsetPosition.Y+h)
}

func (a *SpriteActor) Draw() {
	a.sprite.Draw(a.stage.win, a.Transform())
}

//

type TextActor struct {
	BaseActor
	text                string
	txt                 *text.Text
	horizontalAlignment string
}

func MakeTextActor(position pixel.Vec, stage *Stage) TextActor {
	a := TextActor{BaseActor: MakeBaseActor(stage, "text")}
	a.position = position
	a.txt = text.New(pixel.ZV, a.stage.textAtlas)
	return a
}

func (a *TextActor) SetText(text string) {
	a.text = text
	a.txt.Clear()
	fmt.Fprintln(a.txt, a.text)
}

func (a *TextActor) Bounds() pixel.Rect {
	bounds := a.txt.Bounds()
	if a.horizontalAlignment == "center" {
		bounds = bounds.Moved(pixel.V(-bounds.W()/2, 0))
	}
	return bounds
}

func (a *TextActor) ScaledBounds() pixel.Rect {
	// TODO: a.horizontalAlignment == "center"
	bounds := a.txt.Bounds()
	w := bounds.W() * a.scale
	h := bounds.H() * a.scale
	offsetPosition := a.position.Add(pixel.V(-w/2, -h/2))
	return pixel.R(offsetPosition.X, offsetPosition.Y, offsetPosition.X+w, offsetPosition.Y+h)
}

func (a *TextActor) Draw() {
	transform := a.Transform()
	bounds := a.ScaledBounds()
	if a.horizontalAlignment == "center" {
		transform = transform.Moved(pixel.V(-bounds.W()/2, 0))
	}
	a.txt.Draw(a.stage.win, transform)
}

//

func intersects(a Actor, b Actor) bool {
	// No self-colliding.
	if a == b {
		return false
	}

	// TODO: seems excessively verbose
	aTransform := a.Transform()
	bTransform := b.Transform()
	aPolygon := polygonFromRect(a.Bounds())
	bPolygon := polygonFromRect(b.Bounds())
	aPolygon = projectPolygon(&aPolygon, &aTransform)
	bPolygon = projectPolygon(&bPolygon, &bTransform)
	return polygonsIntersect(&aPolygon, &bPolygon)
}

type Rect pixel.Rect

func (r *Rect) Scaled(scale float64) pixel.Rect {
	return pixel.Rect{Min: r.Min.Scaled(scale), Max: r.Max.Scaled(scale)}
}

type Polygon []pixel.Vec

func polygonFromRect(r pixel.Rect) Polygon {
	return []pixel.Vec{r.Min, pixel.V(r.Min.X, r.Max.Y), r.Max, pixel.V(r.Max.X, r.Min.Y)}
}

func projectPolygon(p *Polygon, t *pixel.Matrix) Polygon {
	r := make([]pixel.Vec, len(*p))
	for i, v := range *p {
		r[i] = t.Project(v)
	}
	return r
}

// Check if the two polygons are intersecting.
// Using the "Separated Axis Theorem". Thanks https://stackoverflow.com/a/10965077/707320
// For each edge in both polygons, check if it can be used as a separating line.
// If so, you are done: No intersection.
// If no separation line was found, you have an intersection.
func polygonsIntersect(a *Polygon, b *Polygon) bool {
	for _, polygon := range []*Polygon{a, b} {
		for i1 := 0; i1 < len(*polygon); i1++ {
			i2 := (i1 + 1) % len(*polygon)
			p1 := (*polygon)[i1]
			p2 := (*polygon)[i2]

			normal := pixel.V(p2.Y-p1.Y, p1.X-p2.X)

			minA := math.MaxFloat64
			maxA := math.MaxFloat64
			for _, p := range *a {
				projected := normal.X*p.X + normal.Y*p.Y
				if minA == math.MaxFloat64 || projected < minA {
					minA = projected
				}
				if maxA == math.MaxFloat64 || projected > maxA {
					maxA = projected
				}
			}

			minB := math.MaxFloat64
			maxB := math.MaxFloat64
			for _, p := range *b {
				projected := normal.X*p.X + normal.Y*p.Y
				if minB == math.MaxFloat64 || projected < minB {
					minB = projected
				}
				if maxB == math.MaxFloat64 || projected > maxB {
					maxB = projected
				}
			}

			if maxA < minB || maxB < minA {
				return false
			}
		}
	}
	return true
}
