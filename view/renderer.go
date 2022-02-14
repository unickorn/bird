package view

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"image/color"
	"sync"
)

// Renderer implements the ebiten.Game interface.
type Renderer struct {
	scale int
	drift float64
	pos   mgl64.Vec2

	w *world.World

	needsRerender bool
	shouldCenter  bool
	centerPos     world.ChunkPos

	chunkMu *sync.Mutex
	chunks  map[world.ChunkPos]*chunk.Chunk

	renderMu    *sync.Mutex
	renderCache map[world.ChunkPos]*ebiten.Image

	playersMu *sync.Mutex
	players   map[string]mgl64.Vec3
}

// NewRenderer creates a new Renderer instance.
func NewRenderer(scale int, drift float64, w *world.World) *Renderer {
	r := &Renderer{
		scale:        scale,
		drift:        drift,
		chunkMu:      new(sync.Mutex),
		renderMu:     new(sync.Mutex),
		playersMu:    new(sync.Mutex),
		players:      make(map[string]mgl64.Vec3),
		shouldCenter: true,
	}
	r.centerPos, r.chunks = loadWorld(w)
	r.renderCache = renderWorld(scale, r.chunkMu, r.chunks)
	r.w = w
	return r
}

// Update proceeds the renderer state.
func (r *Renderer) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		r.pos = r.pos.Add(mgl64.Vec2{0, -r.drift})
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		r.pos = r.pos.Add(mgl64.Vec2{0, r.drift})
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		r.pos = r.pos.Add(mgl64.Vec2{-r.drift, 0})
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		r.pos = r.pos.Add(mgl64.Vec2{r.drift, 0})
	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		r.shouldCenter = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyR) {
		r.needsRerender = true
		refreshCachedChunks(r.w, r.chunks)
	}

	oldScale := r.scale
	_, yOff := ebiten.Wheel()
	if yOff > 0 {
		r.scale++
	} else if yOff < 0 {
		r.scale--
	}
	if r.scale <= 0 {
		r.scale = 1
	}
	if oldScale != r.scale || len(r.renderCache) != len(r.chunks) {
		r.Rerender()
		r.pos = mgl64.Vec2{
			r.pos.X() / (float64(oldScale) * 16),
			r.pos.Y() / (float64(oldScale) * 16),
		}.Mul(float64(r.scale) * 16)
	}
	if r.needsRerender {
		r.renderCache = renderWorld(r.scale, r.chunkMu, r.chunks)
		r.needsRerender = false
	}

	return nil
}

// Draw draws the screen.
func (r *Renderer) Draw(screen *ebiten.Image) {
	screen.Fill(materialColours[0])

	w, h := screen.Size()
	chunkScale := float64(r.scale) * 16
	centerX, centerZ := float64(w/2), float64(h/2)
	if r.shouldCenter {
		r.pos = mgl64.Vec2{float64(r.centerPos.X()), float64(r.centerPos.Z())}.Mul(chunkScale)
		r.shouldCenter = false
	}

	r.renderMu.Lock()
	defer r.renderMu.Unlock()
	for pos, ch := range r.renderCache {
		chunkW, chunkH := ch.Bounds().Dx(), ch.Bounds().Dy()
		offsetX, offsetZ := float64(chunkW/2)+r.pos.X(), float64(chunkH/2)+r.pos.Y()

		chunkX, chunkZ := centerX+(float64(pos.X())*chunkScale), centerZ+(float64(pos.Z())*chunkScale)

		geo := ebiten.GeoM{}
		geo.Translate(chunkX-offsetX, chunkZ-offsetZ)
		screen.DrawImage(ch, &ebiten.DrawImageOptions{GeoM: geo})
	}

	r.playersMu.Lock()
	defer r.playersMu.Unlock()
	for _, pos := range r.players {
		s := ebiten.NewImage(r.scale, r.scale)
		s.Fill(color.RGBA{
			R: 237,
			G: 69,
			B: 49,
			A: 255,
		})

		offsetX, offsetZ := float64(r.scale/2)+r.pos.X(), float64(r.scale/2)+r.pos.Y()
		playerX, playerZ := centerX+(pos.X()*float64(r.scale)), centerZ+(pos.Z()*float64(r.scale))

		geo := ebiten.GeoM{}
		geo.Translate(playerX-offsetX-float64(8*r.scale), playerZ-offsetZ-float64(8*r.scale))
		screen.DrawImage(s, &ebiten.DrawImageOptions{GeoM: geo})
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %.2f", ebiten.CurrentTPS()))
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
func (r *Renderer) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

// Rerender rerenders the world.
func (r *Renderer) Rerender() {
	r.needsRerender = true
}

// RerenderChunk rerenders the chunk at the given position.
func (r *Renderer) RerenderChunk(pos world.ChunkPos) {
	r.chunkMu.Lock()
	r.renderMu.Lock()
	defer r.chunkMu.Unlock()
	defer r.renderMu.Unlock()
	refreshCachedChunks(r.w, r.chunks)
	r.renderCache[pos] = renderChunk(r.scale, r.chunks[pos])
}

// Recenter centers the renderer on the given chunk.
func (r *Renderer) Recenter(pos world.ChunkPos) {
	r.centerPos = pos
	r.shouldCenter = true
}

// MovePlayer ...
func (r *Renderer) MovePlayer(name string, pos mgl64.Vec3) {
	r.playersMu.Lock()
	defer r.playersMu.Unlock()
	r.players[name] = pos
}
