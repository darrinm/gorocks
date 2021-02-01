// TODO:
// - game over
// - randomize seed each run
// - ship deceleration
// - good collision detection
// - sound effects
// - saucers
// - new graphics
// - high score
// - smaller = faster
// - timing variability
// - explosions

package main

import (
	"image"
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
