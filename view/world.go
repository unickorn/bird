package view

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/nfnt/resize"
	"image"
	"reflect"
	"sync"
	"unsafe"
)

// loadWorld loads a world and returns all of its chunks.
func loadWorld(w *world.World) (world.ChunkPos, map[world.ChunkPos]*chunk.Chunk) {
	pr := provider(w)
	prov := pr.(*mcdb.Provider)

	var s world.Settings
	prov.Settings(&s)

	cached := cachedChunks(w)
	centerPos := world.ChunkPos{int32(s.Spawn.X() >> 4), int32(s.Spawn.Z() >> 4)}
	chunks := make(map[world.ChunkPos]*chunk.Chunk)

	propagateChunk(prov, cached, chunks, centerPos)

	return centerPos, chunks
}

// propagateChunk propagates a chunk in the chunks map, and then it's neighbours, until there are no chunks left.
func propagateChunk(prov *mcdb.Provider, cached map[world.ChunkPos]*chunk.Chunk, chunks map[world.ChunkPos]*chunk.Chunk, pos world.ChunkPos) {
	if _, ok := chunks[pos]; ok {
		return
	}

	c, ok := cached[pos]
	if !ok {
		l, exists, err := prov.LoadChunk(pos)
		if err != nil {
			return
		}
		if !exists {
			return
		}
		c = l
	}
	chunks[pos] = c

	propagateChunk(prov, cached, chunks, world.ChunkPos{pos.X(), pos.Z() + 1})
	propagateChunk(prov, cached, chunks, world.ChunkPos{pos.X(), pos.Z() - 1})
	propagateChunk(prov, cached, chunks, world.ChunkPos{pos.X() + 1, pos.Z()})
	propagateChunk(prov, cached, chunks, world.ChunkPos{pos.X() - 1, pos.Z()})
}

// renderWorld renders a world to *ebiten.Images.
func renderWorld(scale int, chunkMu *sync.Mutex, chunks map[world.ChunkPos]*chunk.Chunk) map[world.ChunkPos]*ebiten.Image {
	chunkMu.Lock()
	defer chunkMu.Unlock()

	rendered := make(map[world.ChunkPos]*ebiten.Image)
	for pos, ch := range chunks {
		rendered[pos] = renderChunk(scale, ch)
	}
	return rendered
}

// renderChunk renders a new chunk image from the given chunk.
func renderChunk(scale int, ch *chunk.Chunk) *ebiten.Image {
	img := image.NewRGBA(image.Rectangle{Max: image.Point{X: 16, Y: 16}})
	for x := byte(0); x < 16; x++ {
		for z := byte(0); z < 16; z++ {
			y := ch.HighestBlock(x, z)
			name, properties, _ := chunk.RuntimeIDToState(ch.Block(x, y, z, 0))
			rid, ok := chunk.StateToRuntimeID(name, properties)
			if ok {
				material := materials[rid]
				img.Set(int(x), int(z), materialColours[material])
			}
		}
	}
	return ebiten.NewImageFromImage(resize.Resize(uint(scale*16), uint(scale*16), img, resize.NearestNeighbor))
}

// refreshWorld refreshes the cached chunks.
func refreshCachedChunks(w *world.World, chunks map[world.ChunkPos]*chunk.Chunk) {
	for pos, ch := range cachedChunks(w) {
		chunks[pos] = ch
	}
}

// cachedChunks uses black magic to get the cached chunks from the world.
func cachedChunks(w *world.World) map[world.ChunkPos]*chunk.Chunk {
	r := make(map[world.ChunkPos]*chunk.Chunk)

	// forgive me, father, for i have sinned
	rf := reflect.ValueOf(w).Elem()
	chunks := rf.FieldByName("chunks")
	chunks = reflect.NewAt(chunks.Type(), unsafe.Pointer(chunks.UnsafeAddr())).Elem()

	m := chunks.MapRange()
	for {
		if !m.Next() {
			break
		}

		c := m.Value().Elem().FieldByName("Chunk")
		r[m.Key().Interface().(world.ChunkPos)] = c.Interface().(*chunk.Chunk)
	}
	return r
}

//go:linkname provider github.com/df-mc/dragonfly/server/world.(*World).provider
func provider(world *world.World) world.Provider
