package sos

import ()

func (ss *SpatialSystem) Remove(ID int64) {
	delete := -1
	for index, e := range ss.Entities {
		if e.ID == ID {
			delete = index
			break
		}

	}
	if delete >= 0 {
		ss.Entities = append(ss.Entities[:delete], ss.Entities[delete+1:]...)
	}
}

func (ss *SpatialSystem) GetEntityByID(id int64) *spatialEntity {
	var e *spatialEntity
	for i := range ss.Entities {
		e = &ss.Entities[i]
		if e.ID == id {
			return e
		}
	}
	return nil
}

func (ss *SpatialSystem) tick() {
	ss.TickCount++
	var updatedEnts []int64
	for i := range ss.Entities {
		e := &ss.Entities[i]
		//if e.HasAuthority {
		updatedEnts = append(updatedEnts, e.ID)
		//e.Tick()
		//}
	}

	if len(updatedEnts) > 0 {
		log.Printf("Owns: %v", updatedEnts)
	}
	for _, oe := range updatedEnts {
		e := ss.GetEntityByID(oe)
		if e.HasAuthority {
		}
	}

}
