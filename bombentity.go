// Copyright 2017, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"math"
	"math/rand"

	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/tbogdala/glider"
)

const (
	bombMaxSpeed = 40.0 // 2 m/s
)

// BombEntity is a scene entity for bombs that fly at the player in the game.
type BombEntity struct {
	*VisibleEntity

	currentSpeed         mgl.Vec3 // m/s
	movementCurveYOffset float64
	movementCurveXOffset float64
}

// NewBombEntity returns a new bomb entity object.
func NewBombEntity() *BombEntity {
	b := new(BombEntity)
	b.VisibleEntity = NewVisibleEntity()
	b.movementCurveYOffset = rand.Float64() * 2.0
	b.movementCurveXOffset = rand.Float64() * 2.0
	return b
}

// SetMaxSpeed sets the bomb entity to it's maximum speed.
func (b *BombEntity) SetMaxSpeed() {
	// bombs travel down the negative Z axis
	b.currentSpeed = mgl.Vec3{0.0, 0.0, -bombMaxSpeed}
}

// ScrollPastPlayer should move the entity with relation to the inverse
// of the player speed, adjusted for frame delta.
func (b *BombEntity) ScrollPastPlayer(backwardSpeed mgl.Vec3, frameDelta float32) {
	// in addition to the normal backward speed we're going to add
	// the speed of the bomb.
	totalSpeed := backwardSpeed.Add(b.currentSpeed.Mul(frameDelta))

	// now we do a little wave adjustment
	totalSpeed[0] = totalSpeed[0] + float32(math.Cos(gameScene.currentGameTime+b.movementCurveXOffset))*frameDelta
	totalSpeed[1] = totalSpeed[1] + float32(math.Sin(gameScene.currentGameTime+b.movementCurveYOffset))*frameDelta

	// move everything else back the current speed of the ship
	loc := b.GetLocation().Add(totalSpeed)
	b.SetLocation(loc)
}

// GetColliders should return all of the coarse colliders for an entity.
func (b *BombEntity) GetColliders() []glider.Collider {
	return b.VisibleEntity.CoarseColliders
}
