package sos

import "time"

type spatialEntity struct {
	ID                 int64
	HasAuthority       bool
	LastPositionUpdate time.Time
}

func (se *spatialEntity) HasStalePosition() bool {
	delta := time.Since(se.LastPositionUpdate)
	return delta > time.Second
}

func (se *spatialEntity) PendingAuthorityLoss() {
}

func (se *spatialEntity) Tick() {

}
