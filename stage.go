package main

import (
	"fmt"
	"image"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"golang.org/x/image/font/basicfont"
)

// Stage retains, updates, and draws Actors.
type Stage struct {
	win              *pixelgl.Window
	actors           []Actor
	bounds           pixel.Rect
	spritesheet      pixel.Picture
	spritesheetImage image.Image
	frames           []pixel.Rect
	imd              *imdraw.IMDraw
	textAtlas        *text.Atlas

	drawActorBounds bool
	actorIDs        map[Actor]int
	nextActorID     int
}

// MakeStage creates and initializes a Stage object.
func MakeStage(stage Stage) Stage {
	s := stage
	s.actorIDs = make(map[Actor]int)
	s.nextActorID = 1 // ActorID 0 is reserved (means "not on the actors list")
	s.imd = imdraw.New(nil)
	s.textAtlas = text.NewAtlas(basicfont.Face7x13, text.ASCII)
	return s
}

// Reset the Stage to its initial state. All Actors are removed.
func (s *Stage) Reset() {
	s.actors = make([]Actor, 0)
}

// AddActor adds the specified Actor to the Stage.
func (s *Stage) AddActor(actor Actor) {
	if s.actorIDs[actor] != 0 {
		panic(fmt.Sprintf("Actor has already been added. %#v", actor))
	}
	s.actors = append(s.actors, actor)
	s.actorIDs[actor] = s.nextActorID
	s.nextActorID++
}

// RemoveActor removes the specified Actor from the Stage.
func (s *Stage) RemoveActor(actor Actor) {
	actorID := s.actorIDs[actor]
	if actorID == 0 {
		panic(fmt.Sprintf("Actor not found. %#v", actor))
	}

	for i, actorT := range s.actors {
		if s.actorIDs[actorT] == actorID {
			delete(s.actorIDs, actor)
			s.actors = append(s.actors[:i], s.actors[i+1:]...)
			return
		}
	}
}

// FindActorsByKind returns an array of all Actors matching the requested 'kind', or nil if none.
func (s *Stage) FindActorsByKind(kind string) []Actor {
	actors := make([]Actor, 0)
	for _, actor := range s.actors {
		if actor.Kind() == kind {
			actors = append(actors, actor)
		}
	}
	if len(actors) == 0 {
		return nil
	}
	return actors
}

// Update all Actors.
func (s *Stage) Update(dt float64) {
	// Make a copy to protect from Update mutations.
	actors := make([]Actor, len(s.actors))
	copy(actors, s.actors)
	for _, actor := range actors {
		if s.actorIDs[actor] != 0 {
			actor.Update(dt)
		}
	}
}

// Draw all Actors.
func (s *Stage) Draw() {
	// Draw all the Actors.
	// Make a copy to protect from Draw mutations (that be would be dumb, but just in case).
	actors := make([]Actor, len(s.actors))
	copy(actors, s.actors)
	for _, actor := range actors {
		if s.actorIDs[actor] != 0 {
			actor.Draw()
		}
	}

	s.imd.Clear()

	// Draw bounding boxes of all actors.
	if s.drawActorBounds {
		for _, actor := range s.actors {
			polygon := polygonFromRect(actor.Bounds())
			transform := actor.Transform()
			polygon = projectPolygon(&polygon, &transform)
			for _, v := range polygon {
				s.imd.Push(v)
			}
			s.imd.Polygon(1)
		}
	}

	s.imd.Draw(s.win)
}
