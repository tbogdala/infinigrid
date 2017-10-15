// Copyright 2017, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"math"

	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/tbogdala/glider"
)

// max roll and pitch speeds for input
const (
	shipMoveSpeed = 20.0          // 1 m/s
	maxRollRads   = math.Pi / 4.0 // 45 deg
	maxPitchRads  = math.Pi / 8.0 // 22.5 deg
)

// ShipEntity is a scene entity for ships that fly in the game.
type ShipEntity struct {
	*VisibleEntity

	// currentShipRoll is the current Roll rotation for the ship in radians.
	currentShipRoll float32

	// currentShipPitch is the current Pitch rotation for the ship in radians.
	currentShipPitch float32

	currentShipSpeed mgl.Vec3 // m/s
}

// NewShipEntity returns a new ship entity object.
func NewShipEntity() *ShipEntity {
	se := new(ShipEntity)
	se.VisibleEntity = NewVisibleEntity()
	return se
}

// GetColliders should return all of the coarse colliders for an entity.
func (s *ShipEntity) GetColliders() []glider.Collider {
	return s.VisibleEntity.CoarseColliders
}
