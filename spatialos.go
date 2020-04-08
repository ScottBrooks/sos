package sos

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
