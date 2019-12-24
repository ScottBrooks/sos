package sos

// #cgo LDFLAGS: -Wl,-rpath,\$ORIGIN -L${SRCDIR}/c_sdk -lworker
// #include "c_sdk/include/improbable/c_schema.h"
// #include "c_sdk/include/improbable/c_worker.h"
// #include <inttypes.h>
// #include <stdio.h>
// #include <stdlib.h>
// #include <string.h>
import "C"

import (
	"fmt"
	"reflect"
	"strconv"

	"log"
	"math/rand"
	"os"
	"os/signal"
	"unsafe"
)

const TIMEOUT_MS = 500

type EntityID int64
type RequestID uint32
type ComponentID uint32

type WorkerEntity struct {
	ID         EntityID
	Components []interface{}
}

type WorkerComponent struct {
	CID ComponentID
	//TODO: Rest of the data.
}

type DisconnectOp struct{}
type FlagUpdateOp struct {
	Key   string
	Value string
}
type LogMessageOp struct {
	Level   uint8
	Message string
}
type MetricsOp struct{}
type CriticalSectionOp struct{ In bool }
type AddEntityOp struct {
	ID EntityID
}
type RemoveEntityOp struct{ ID EntityID }
type ReserveEntityIdOp struct {
	RID        RequestID
	StatusCode uint8
	Message    string
	ID         EntityID
}
type ReserveEntityIdsOp struct {
	RID        RequestID
	StatusCode uint8
	Message    string
	FirstID    EntityID
	Num        int
}
type CreateEntityOp struct {
	RID        RequestID
	StatusCode uint8
	Message    string
	ID         EntityID
}
type DeleteEntityOp struct {
	RID        RequestID
	ID         EntityID
	StatusCode uint8
	Message    string
}
type EntityQueryOp struct {
	RID        RequestID
	StatusCode uint8
	Message    string
	Num        int
	Results    []WorkerEntity
}
type AddComponentOp struct {
	ID        EntityID
	CID       ComponentID
	Component interface{}
}
type RemoveComponentOp struct {
	ID  EntityID
	CID ComponentID
}
type AuthorityChangeOp struct {
	ID        EntityID
	CID       ComponentID
	Authority uint8
}
type ComponentUpdateOp struct {
	ID        EntityID
	CID       ComponentID
	Component interface{}
}
type CommandRequestOp struct {
	RID            RequestID
	ID             EntityID
	TimeoutMillis  uint32
	CallerWorkerID string
}
type CommandResponseOp struct {
	RID        RequestID
	ID         EntityID
	StatusCode uint8
	Message    string
}

type SphereConstraint struct {
	X, Y, Z, R float32
}
type AndConstraint struct {
	Constraints []BaseConstraint
}
type OrConstraint struct {
	Constraints []BaseConstraint
}
type NotConstraint struct {
	Constraint BaseConstraint
}

type BaseConstraint struct {
	EntityID    *EntityID
	ComponentID *ComponentID
	Sphere      *SphereConstraint
	And         *AndConstraint
	Or          *OrConstraint
	Not         *NotConstraint
}

func (bc *BaseConstraint) ToConstraint() C.Worker_Constraint {
	c := C.Worker_Constraint{}

	if bc.EntityID != nil {
		c.constraint_type = C.WORKER_CONSTRAINT_TYPE_ENTITY_ID
		constraint := C.Worker_EntityIdConstraint{entity_id: C.Worker_EntityId(*bc.EntityID)}
		C.memcpy(unsafe.Pointer(&c.anon0), unsafe.Pointer(&constraint), C.sizeof_Worker_EntityIdConstraint)

	}
	if bc.ComponentID != nil {
		c.constraint_type = C.WORKER_CONSTRAINT_TYPE_COMPONENT
		constraint := C.Worker_ComponentConstraint{component_id: C.Worker_ComponentId(*bc.ComponentID)}
		C.memcpy(unsafe.Pointer(&c.anon0), unsafe.Pointer(&constraint), C.sizeof_Worker_ComponentConstraint)

	}
	log.Printf("Constraint: %+v -> %+v", bc, c)

	return c
}

type Adapter interface {
	OnDisconnect(DisconnectOp)
	OnFlagUpdate(FlagUpdateOp)
	OnLogMessage(LogMessageOp)
	OnMetrics(MetricsOp)
	OnCriticalSection(CriticalSectionOp)
	OnAddEntity(AddEntityOp)
	OnRemoveEntity(RemoveEntityOp)
	OnReserveEntityId(ReserveEntityIdOp)
	OnReserveEntityIds(ReserveEntityIdsOp)
	OnCreateEntity(CreateEntityOp)
	OnDeleteEntity(DeleteEntityOp)
	OnEntityQuery(EntityQueryOp)
	OnAddComponent(AddComponentOp)
	OnRemoveComponent(RemoveComponentOp)
	OnAuthorityChange(AuthorityChangeOp)
	OnComponentUpdate(ComponentUpdateOp)
	OnCommandRequest(CommandRequestOp)
	OnCommandResponse(CommandResponseOp)
	AllocComponent(ID EntityID, CID ComponentID) (interface{}, error)

	WorkerType() string
}

type QueryResponseFunc func(count int, results *C.Worker_Entity)

type SpatialSystem struct {
	connection *C.Worker_Connection
	LogMetrics bool

	Entities          []spatialEntity
	InCriticalSection bool
	TickCount         int
	WorkerID          string

	handler Adapter
}

func (ss *SpatialSystem) sendComponentUpdate(id int64, u *C.Worker_ComponentUpdate) {

	C.Worker_Connection_SendComponentUpdate(ss.connection, C.int64_t(id), u)
}
func (ss *SpatialSystem) AddDisturbance(x, y float32, amount float32) {
}

func (ss *SpatialSystem) CreateEntity(ent interface{}) RequestID {
	v := reflect.ValueOf(ent)
	t := reflect.TypeOf(ent)
	var cd []C.Worker_ComponentData
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		st := t.Field(i)
		if st.Tag.Get("sos") == "-" {
			continue
		}
		cid, err := strconv.Atoi(st.Tag.Get("sos"))
		if err != nil {
			log.Printf("Error in struct tag for %+v: %d", ent, i)
		}

		cd = append(cd, structToSchema(ComponentID(cid), f.Interface()))
	}

	timeout := C.uint(TIMEOUT_MS)
	reqID := C.Worker_Connection_SendCreateEntityRequest(ss.connection, C.uint(len(cd)), &cd[0], nil, &timeout)
	return RequestID(reqID)

}

func (ss *SpatialSystem) Delete(id EntityID) {
	timeout := C.uint(TIMEOUT_MS)
	C.Worker_Connection_SendDeleteEntityRequest(ss.connection, C.int64_t(id), &timeout)
	// log.Printf("Delete req: %+v", req)

}

func (ss *SpatialSystem) Shutdown() {
	C.Worker_Connection_Destroy(ss.connection)
}

func NewSpatialSystem(handler Adapter, host string, port int, workerID string) *SpatialSystem {
	wt := handler.WorkerType()
	if workerID == "" {
		workerID = fmt.Sprintf("%s_%d", wt, rand.Intn(1024))
	}
	ss := SpatialSystem{
		WorkerID: workerID,
		handler:  handler,
	}
	params := C.Worker_DefaultConnectionParameters()
	worker_type := C.CString(wt)

	default_vtable := (*C.Worker_ComponentVtable)(C.calloc(C.sizeof_Worker_ComponentVtable, 1))
	params.worker_type = worker_type
	params.network.tcp.multiplex_level = 4
	params.default_component_vtable = default_vtable
	addr := C.CString(host)
	connection_future := C.Worker_ConnectAsync(addr, C.ushort(port), C.CString(ss.WorkerID), &params)

	ss.connection = C.Worker_ConnectionFuture_Get(connection_future, nil)
	//TODO: call on shutdown -> C.Worker_ConnectionFuture_Destroy(connection_future)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		sig := <-c
		log.Printf("Signal: %+v", sig)
		C.Worker_ConnectionFuture_Destroy(connection_future)
		os.Exit(1)
	}()

	logger_name := C.CString("sos")
	message := C.CString("Connected to spatialos")

	var msg C.Worker_LogMessage
	msg.level = C.WORKER_LOG_LEVEL_WARN
	msg.logger_name = logger_name
	msg.message = message

	C.Worker_Connection_SendLogMessage(ss.connection, &msg)

	attr := C.Worker_Connection_GetWorkerAttributes(ss.connection)
	log.Printf("Attr: %+v", attr)

	return &ss
}

func (ss *SpatialSystem) Update(dt float32) {

	for {
		op_list := C.Worker_Connection_GetOpList(ss.connection, 0)
		op_count := int(op_list.op_count)
		for i := 0; i < op_count; i++ {
			// Maybe bad math when multiple ops
			op := (*C.Worker_Op)(unsafe.Pointer(uintptr(unsafe.Pointer(op_list.ops)) + uintptr(i)*C.sizeof_Worker_Op))
			switch op.op_type {
			case C.WORKER_OP_TYPE_DISCONNECT:
				disconnect := (*C.Worker_DisconnectOp)(unsafe.Pointer(&op.anon0))
				ss.onDisconnect(disconnect)
			case C.WORKER_OP_TYPE_FLAG_UPDATE:
				flagUpdate := (*C.Worker_FlagUpdateOp)(unsafe.Pointer(&op.anon0))
				ss.onFlagUpdate(flagUpdate)
			case C.WORKER_OP_TYPE_LOG_MESSAGE:
				logMessage := (*C.Worker_LogMessageOp)(unsafe.Pointer(&op.anon0))
				ss.onLogMessage(logMessage)
			case C.WORKER_OP_TYPE_METRICS:
				metrics := (*C.Worker_Metrics)(unsafe.Pointer(&op.anon0))
				ss.onMetrics(metrics)
			case C.WORKER_OP_TYPE_CRITICAL_SECTION:
				cs := (*C.Worker_CriticalSectionOp)(unsafe.Pointer(&op.anon0))
				ss.onCriticalSection(cs)
			case C.WORKER_OP_TYPE_ADD_ENTITY:
				addEntity := (*C.Worker_AddEntityOp)(unsafe.Pointer(&op.anon0))
				ss.onAddEntity(addEntity)
			case C.WORKER_OP_TYPE_REMOVE_ENTITY:
				removeEntity := (*C.Worker_RemoveEntityOp)(unsafe.Pointer(&op.anon0))
				ss.onRemoveEntity(removeEntity)
			case C.WORKER_OP_TYPE_RESERVE_ENTITY_ID_RESPONSE:
				reserveEntityId := (*C.Worker_ReserveEntityIdResponseOp)(unsafe.Pointer(&op.anon0))
				ss.onReserveEntityId(reserveEntityId)
			case C.WORKER_OP_TYPE_RESERVE_ENTITY_IDS_RESPONSE:
				reserveEntityIds := (*C.Worker_ReserveEntityIdsResponseOp)(unsafe.Pointer(&op.anon0))
				ss.onReserveEntityIds(reserveEntityIds)
			case C.WORKER_OP_TYPE_CREATE_ENTITY_RESPONSE:
				createEntity := (*C.Worker_CreateEntityResponseOp)(unsafe.Pointer(&op.anon0))
				ss.onCreateEntity(createEntity)
			case C.WORKER_OP_TYPE_DELETE_ENTITY_RESPONSE:
				deleteEntity := (*C.Worker_DeleteEntityResponseOp)(unsafe.Pointer(&op.anon0))
				ss.onDeleteEntity(deleteEntity)
			case C.WORKER_OP_TYPE_ENTITY_QUERY_RESPONSE:
				entityQueryResponse := (*C.Worker_EntityQueryResponseOp)(unsafe.Pointer(&op.anon0))
				ss.onEntityQueryResponse(entityQueryResponse)
			case C.WORKER_OP_TYPE_ADD_COMPONENT:
				addComponent := (*C.Worker_AddComponentOp)(unsafe.Pointer(&op.anon0))
				ss.onAddComponent(addComponent)
			case C.WORKER_OP_TYPE_REMOVE_COMPONENT:
				removeComponent := (*C.Worker_RemoveComponentOp)(unsafe.Pointer(&op.anon0))
				ss.onRemoveComponent(removeComponent)
			case C.WORKER_OP_TYPE_AUTHORITY_CHANGE:
				authorityChange := (*C.Worker_AuthorityChangeOp)(unsafe.Pointer(&op.anon0))
				ss.onAuthorityChange(authorityChange)
			case C.WORKER_OP_TYPE_COMPONENT_UPDATE:
				componentUpdate := (*C.Worker_ComponentUpdateOp)(unsafe.Pointer(&op.anon0))
				ss.onComponentUpdate(componentUpdate)
			case C.WORKER_OP_TYPE_COMMAND_REQUEST:
				commandRequest := (*C.Worker_CommandRequestOp)(unsafe.Pointer(&op.anon0))
				ss.onCommandRequest(commandRequest)
			case C.WORKER_OP_TYPE_COMMAND_RESPONSE:
				commandResponse := (*C.Worker_CommandResponseOp)(unsafe.Pointer(&op.anon0))
				ss.onCommandResponse(commandResponse)

			default:
				log.Printf("Op: %+v", op)
			}

		}
		C.Worker_OpList_Destroy(op_list)
		if op_count == 0 {
			break
		}

	}

}

func (ss *SpatialSystem) UpdateComponent(ID EntityID, CID ComponentID, comp interface{}) {
	var update C.Worker_ComponentUpdate
	update.component_id = C.uint(CID)
	update.schema_type = C.Schema_CreateComponentUpdate(update.component_id)
	obj := C.Schema_GetComponentUpdateFields(update.schema_type)
	structToObj(obj, comp)

	C.Worker_Connection_SendComponentUpdate(ss.connection, C.int64_t(ID), &update)
}

func (ss *SpatialSystem) EntityQuery(bc BaseConstraint, fullQuery bool, components []ComponentID) RequestID {
	timeout := C.uint(TIMEOUT_MS * 10)
	var query C.Worker_EntityQuery
	if fullQuery {
		query.result_type = C.WORKER_RESULT_TYPE_SNAPSHOT
	} else {
		query.result_type = C.WORKER_RESULT_TYPE_COUNT
	}
	if len(components) != 0 {
		query.snapshot_result_type_component_id_count = C.uint(len(components))
		query.snapshot_result_type_component_ids = (*C.Worker_ComponentId)(C.calloc(C.sizeof_Worker_ComponentId, C.uint64_t(query.snapshot_result_type_component_id_count)))
		for i, cid := range components {
			val := (*C.Worker_ComponentId)(unsafe.Pointer(uintptr(unsafe.Pointer(query.snapshot_result_type_component_ids)) + uintptr(i)*C.sizeof_Worker_ComponentId))
			*val = C.uint(cid)
		}
	}
	query.constraint = bc.ToConstraint()

	return RequestID(C.Worker_Connection_SendEntityQueryRequest(ss.connection, &query, &timeout))
}

func (ss *SpatialSystem) onDisconnect(op *C.Worker_DisconnectOp) {
	log.Printf("Disconnected[%d]: %s", op.connection_status_code, C.GoString(op.reason))
	ss.handler.OnDisconnect(DisconnectOp{})
}

func (ss *SpatialSystem) onFlagUpdate(op *C.Worker_FlagUpdateOp) {
	key := C.GoString(op.name)
	value := C.GoString(op.value)
	ss.handler.OnFlagUpdate(FlagUpdateOp{Key: key, Value: value})
}
func (ss *SpatialSystem) onLogMessage(op *C.Worker_LogMessageOp) {
	//TODO entity_id for the log message
	ss.handler.OnLogMessage(LogMessageOp{Level: uint8(op.level), Message: C.GoString(op.message)})
}

func (ss *SpatialSystem) onMetrics(metrics *C.Worker_Metrics) {
	// TODO: Metrics
	//log.Printf("OnMetrics: %+v", metrics)
	//ss.handler.OnMetrics(MetricsOp{})
}

func (ss *SpatialSystem) onCriticalSection(op *C.Worker_CriticalSectionOp) {
	ss.handler.OnCriticalSection(CriticalSectionOp{In: op.in_critical_section == 1})

	ss.InCriticalSection = op.in_critical_section == 1
}
func (ss *SpatialSystem) onAddEntity(op *C.Worker_AddEntityOp) {
	ss.handler.OnAddEntity(AddEntityOp{ID: EntityID(op.entity_id)})

	e := spatialEntity{}
	e.ID = int64(op.entity_id)
	ss.Entities = append(ss.Entities, e)
}
func (ss *SpatialSystem) onRemoveEntity(op *C.Worker_RemoveEntityOp) {
	ss.handler.OnRemoveEntity(RemoveEntityOp{ID: EntityID(op.entity_id)})

	index := -1
	for i, e := range ss.Entities {
		if e.ID == int64(op.entity_id) {
			index = i
			break

		}
	}
	if index >= 0 {
		log.Printf("Removing entity from our view: %d", op.entity_id)
		ss.Entities = append(ss.Entities[:index], ss.Entities[index+1:]...)
	}

}
func (ss *SpatialSystem) onReserveEntityId(op *C.Worker_ReserveEntityIdResponseOp) {
	ss.handler.OnReserveEntityId(ReserveEntityIdOp{RID: RequestID(op.request_id), StatusCode: uint8(op.status_code), Message: C.GoString(op.message), ID: EntityID(op.entity_id)})
}
func (ss *SpatialSystem) onReserveEntityIds(op *C.Worker_ReserveEntityIdsResponseOp) {
	ss.handler.OnReserveEntityIds(ReserveEntityIdsOp{RID: RequestID(op.request_id), StatusCode: uint8(op.status_code), Message: C.GoString(op.message), FirstID: EntityID(op.first_entity_id), Num: int(op.number_of_entity_ids)})
}
func (ss *SpatialSystem) onCreateEntity(op *C.Worker_CreateEntityResponseOp) {
	ss.handler.OnCreateEntity(CreateEntityOp{RID: RequestID(op.request_id), StatusCode: uint8(op.status_code), Message: C.GoString(op.message), ID: EntityID(op.entity_id)})
}
func (ss *SpatialSystem) onDeleteEntity(op *C.Worker_DeleteEntityResponseOp) {
	ss.handler.OnDeleteEntity(DeleteEntityOp{RID: RequestID(op.request_id), StatusCode: uint8(op.status_code), Message: C.GoString(op.message)})
	if op.status_code == 1 {
		//log.Printf("DeleteEntity: %+v %s", op, C.GoString(op.message))
		ss.Remove(int64(op.entity_id))
	}
}
func (ss *SpatialSystem) onEntityQueryResponse(op *C.Worker_EntityQueryResponseOp) {
	log.Printf("Got a response: %+v: %s", op, C.GoString(op.message))

	var results []WorkerEntity
	if op.results != nil {
		results = make([]WorkerEntity, op.result_count)
		for i := 0; i < len(results); i++ {
			ent := (*C.Worker_Entity)(unsafe.Pointer(uintptr(unsafe.Pointer(op.results)) + uintptr(i)*C.sizeof_Worker_Entity))
			results[i].ID = EntityID(ent.entity_id)

			for j := 0; j < int(ent.component_count); j++ {
				comp := (*C.Worker_ComponentData)(unsafe.Pointer(uintptr(unsafe.Pointer(ent.components)) + uintptr(j)*C.sizeof_Worker_ComponentData))
				obj := C.Schema_GetComponentDataFields(comp.schema_type)

				c, err := ss.handler.AllocComponent(EntityID(ent.entity_id), ComponentID(comp.component_id))
				if err != nil {
					log.Printf("unable to alloc component on query")
					return
				}
				schemaToStruct(c, obj)

				results[i].Components = append(results[i].Components, c)
			}
		}
	}

	ss.handler.OnEntityQuery(EntityQueryOp{RID: RequestID(op.request_id), StatusCode: uint8(op.status_code), Message: C.GoString(op.message), Num: int(op.result_count), Results: results})
}
func (ss *SpatialSystem) onAddComponent(op *C.Worker_AddComponentOp) {
	log.Printf("Adding Component: %d to Entity: %d", op.data.component_id, op.entity_id)
	c, err := ss.handler.AllocComponent(EntityID(op.entity_id), ComponentID(op.data.component_id))
	if err != nil {
		log.Printf("Unable to alloc component")
		return
	}
	obj := C.Schema_GetComponentDataFields(op.data.schema_type)
	schemaToStruct(c, obj)
	ss.handler.OnAddComponent(AddComponentOp{ID: EntityID(op.entity_id), CID: ComponentID(op.data.component_id), Component: c})

}
func (ss *SpatialSystem) onRemoveComponent(op *C.Worker_RemoveComponentOp) {
	ss.handler.OnRemoveComponent(RemoveComponentOp{ID: EntityID(op.entity_id), CID: ComponentID(op.component_id)})
}

func (ss *SpatialSystem) onAuthorityChange(op *C.Worker_AuthorityChangeOp) {
	ss.handler.OnAuthorityChange(AuthorityChangeOp{ID: EntityID(op.entity_id), CID: ComponentID(op.component_id), Authority: uint8(op.authority)})

	e := ss.GetEntityByID(int64(op.entity_id))
	if e == nil {
		log.Printf("Could not find entity matchining id: %d", op.entity_id)
		return
	}
	if op.authority == 2 {
		log.Printf("Pending loss")
		e.PendingAuthorityLoss()
	} else if op.authority == 1 {
		e.HasAuthority = true
	} else {
		e.HasAuthority = false
	}

}
func (ss *SpatialSystem) onComponentUpdate(op *C.Worker_ComponentUpdateOp) {
	c, err := ss.handler.AllocComponent(EntityID(op.entity_id), ComponentID(op.update.component_id))
	if err != nil {
		log.Printf("Unable to alloc component")
		return
	}
	obj := C.Schema_GetComponentUpdateFields(op.update.schema_type)
	schemaToStruct(c, obj)
	ss.handler.OnComponentUpdate(ComponentUpdateOp{ID: EntityID(op.entity_id), CID: ComponentID(op.update.component_id), Component: c})
}
func (ss *SpatialSystem) onCommandRequest(op *C.Worker_CommandRequestOp) {
	log.Printf("CommandRequest: %+v", op)
	//TODO deserialize command request + attributes
	ss.handler.OnCommandRequest(CommandRequestOp{RID: RequestID(op.request_id), ID: EntityID(op.entity_id), TimeoutMillis: uint32(op.timeout_millis), CallerWorkerID: C.GoString(op.caller_worker_id)})
}
func (ss *SpatialSystem) onCommandResponse(op *C.Worker_CommandResponseOp) {
	log.Printf("CommandResponse: %+v", op)
	//TODO deserialize command response
	ss.handler.OnCommandResponse(CommandResponseOp{RID: RequestID(op.request_id), ID: EntityID(op.entity_id), StatusCode: uint8(op.status_code), Message: C.GoString(op.message)})
}