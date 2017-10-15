// Copyright 2017, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/tbogdala/glider"
)

// WallSetEntity is a scene entity for a wall segment that is placed in the game.
type WallSetEntity struct {
	*VisibleEntity
}

// NewWallSetEntity returns a new wall set entity object.
func NewWallSetEntity() *WallSetEntity {
	wse := new(WallSetEntity)
	wse.VisibleEntity = NewVisibleEntity()
	return wse
}

// ScrollPastPlayer should move the entity with relation to the inverse
// of the player speed, adjusted for frame delta.
func (wse *WallSetEntity) ScrollPastPlayer(backwardSpeed mgl.Vec3, frameDelta float32) {
	// move everything else back the current speed of the ship
	loc := wse.GetLocation().Add(backwardSpeed)
	wse.SetLocation(loc)
}

// GetColliders should return all of the coarse colliders for an entity.
func (wse *WallSetEntity) GetColliders() []glider.Collider {
	return wse.VisibleEntity.CoarseColliders
}
