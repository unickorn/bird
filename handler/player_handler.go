package handler

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/unickorn/bird/view"
	"time"
)

// PlayerHandler ...
type PlayerHandler struct {
	player.NopHandler
	p *player.Player
	r *view.Renderer
}

// NewPlayerHandler ...
func NewPlayerHandler(p *player.Player, r *view.Renderer) *PlayerHandler {
	return &PlayerHandler{
		p: p,
		r: r,
	}
}

// HandleBlockPlace ...
func (p *PlayerHandler) HandleBlockPlace(_ *event.Context, pos cube.Pos, _ world.Block) {
	go func() {
		time.Sleep(time.Millisecond * 1)
		w := p.p.World()
		if pos.Y() == w.HighestBlock(pos.X(), pos.Z()) {
			p.r.RerenderChunk(world.ChunkPos{int32(pos.X() >> 4), int32(pos.Z() >> 4)})
		}
	}()
}

// HandleBlockBreak ...
func (p *PlayerHandler) HandleBlockBreak(_ *event.Context, pos cube.Pos, _ *[]item.Stack) {
	w := p.p.World()
	highest := w.HighestBlock(pos.X(), pos.Z())
	if highest == pos.Y() {
		go func() {
			time.Sleep(time.Millisecond * 1)
			p.r.RerenderChunk(world.ChunkPos{int32(pos.X() >> 4), int32(pos.Z() >> 4)})
		}()
	}
}

// HandleMove ...
func (p *PlayerHandler) HandleMove(_ *event.Context, newPos mgl64.Vec3, _, _ float64) {
	p.r.MovePlayer(p.p.Name(), newPos)
}
