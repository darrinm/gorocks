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

type Stage struct {
	win              *pixelgl.Window
	actors           []Actor
	bounds           pixel.Rect
	spritesheet      pixel.Picture
	spritesheetImage image.Image
	frames           []pixel.Rect
	imd              *imdraw.IMDraw
	textAtlas        *text.Atlas
}

func MakeStage(stage Stage) Stage {
	s := stage
	s.imd = imdraw.New(nil)
	s.textAtlas = text.NewAtlas(basicfont.Face7x13, text.ASCII)
	return s
}

func (s *Stage) reset() {
	s.actors = make([]Actor, 0)
}

func (s *Stage) addActor(actor Actor) {
	for _, a := range s.actors {
		if actor == a {
			panic(fmt.Sprintf("Actor has already been added"))
		}
	}
	s.actors = append(s.actors, actor)
}

func (s *Stage) removeActor(actor Actor) {
	for i, actorT := range s.actors {
		if actorT == actor {
			actor.(*BaseActor).stage = nil
			s.actors = append(s.actors[:i], s.actors[i+1:]...)
			return
		}
	}
	panic(fmt.Sprintf("failed to remove actor", actor))
}

func (s *Stage) update(dt float64) {
	// Make a copy because it may be mutated by Update.
	var actors []Actor
	copy(actors, s.actors)
	for _, actor := range actors {
		if actor.(*BaseActor).stage != nil {
			actor.Update(dt)
		}
	}
}

func (s *Stage) draw() {
	// Draw all the Actors.
	// Make a copy because it may be mutated by Draw (that be would be dumb, but just in case).
	var actors []Actor
	copy(actors, s.actors)
	for _, actor := range actors {
		if actor.(*BaseActor).stage != nil {
			actor.Draw()
		}
	}

	s.imd.Clear()

	// Draw bounding boxes of all actors.
	if false {
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
