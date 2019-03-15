package core

import (
	"errors"
	"fmt"

	"github.com/MG-RAST/AWE/lib/conf"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/user"

	//"github.com/davecgh/go-spew/spew"
	"encoding/json"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"
)

// CQMgr this struct is embedded in ServerMgr
type CQMgr struct {
	clientMap    ClientMap
	workQueue    *WorkQueue
	suspendQueue bool
	coReq        chan CheckoutRequest //workunit checkout request (WorkController -> qmgr.Handler)
	feedback     chan Notice          //workunit execution feedback (WorkController -> qmgr.Handler)
	coSem        chan int             //semaphore for checkout (mutual exclusion between different clients)
}

// FilterWorkStats _
type FilterWorkStats struct {
	Total            int
	SkipWork         int
	WrongClientgroup int
	WrongApp         int
}

//--------mgr methods-------

// ClientHandle _
func (qm *CQMgr) ClientHandle() {
	// this code is not beeing used
}

//--------accessor methods-------

// GetClientMap _
func (qm *CQMgr) GetClientMap() *ClientMap {
	return &qm.clientMap
}

// AddClient _
func (qm *CQMgr) AddClient(client *Client, lock bool) (err error) {
	err = qm.clientMap.Add(client, lock)
	return
}

// GetClient _
func (qm *CQMgr) GetClient(id string, lockClientMap bool) (client *Client, ok bool, err error) {
	return qm.clientMap.Get(id, lockClientMap)
}

// RemoveClient lock is for clientmap
func (qm *CQMgr) RemoveClient(id string, lock bool) (err error) {

	client, ok, err := qm.clientMap.Get(id, true)
	if err != nil {
		return
	}
	if !ok {
		err = fmt.Errorf("Client %s not found", id)
		return
	}

	//now client must be gone as tag set to false 30 seconds ago and no heartbeat received thereafter
	logger.Event(event.CLIENT_UNREGISTER, "clientid="+client.ID)
	//requeue unfinished workunits associated with the failed client
	err = qm.ReQueueWorkunitByClient(client, true)
	if err != nil {
		logger.Error("(CheckClient) %s", err.Error())
	}

	err = qm.clientMap.Delete(id, lock)
	return
}

//func (qm *CQMgr) DeleteClient(client *Client) (err error) {
//	err = qm.ClientStatusChange(client, CLIENT_STAT_DELETED, true)
//	return
//}

//func (qm *CQMgr) DeleteClientById(id string) (err error) {
//	err = qm.ClientIdStatusChange(id, CLIENT_STAT_DELETED, true)
//	return
//}

// func (qm *CQMgr) ClientIdStatusChange_deprecated(id string, new_status string, clientWriteLock bool) (err error) {
// 	client, ok, err := qm.clientMap.Get(id, true)
// 	if err != nil {
// 		return
// 	}
// 	if ok {
// 		//err = client.Set_Status(new_status, clientWriteLock)
// 		return
// 	}
// 	returnerrors.New(e.ClientNotFound)
//}

// ClientStatusChangeDeprecated _
func (qm *CQMgr) ClientStatusChangeDeprecated(client *Client, newStatus string, clientWriteLock bool) (err error) {
	//client.Set_Status(new_status, clientWriteLock)
	return

}

// HasClient _
func (qm *CQMgr) HasClient(id string, lockClientMap bool) (has bool, err error) {
	_, ok, err := qm.clientMap.Get(id, lockClientMap)
	if err != nil {
		return
	}
	if ok {
		has = true
	} else {
		has = false
	}
	return
}

// ListClients _
func (qm *CQMgr) ListClients() (ids []string, err error) {
	//qm.clientMap.RLock()
	//defer qm.clientMap.RUnlock()
	//for id, _ := range qm.clientMap {
	//	ids = append(ids, id)
	//}
	return qm.clientMap.GetClientIds()
}

//--------client methods-------

//CheckClient _
func (qm *CQMgr) CheckClient(client *Client) (ok bool, err error) {
	ok = true
	err = client.LockNamed("ClientChecker")
	if err != nil {
		return
	}
	defer client.Unlock()

	logger.Debug(3, "(CheckClient) client: %s", client.ID)

	if client.Tag == true {
		// *** Client is active
		client.Online = true
		logger.Debug(3, "(CheckClient) client %s active", client.ID)

		client.Tag = false
		totalMinutes := int(time.Now().Sub(client.RegTime).Minutes())
		hours := totalMinutes / 60
		minutes := totalMinutes % 60
		client.ServeTime = fmt.Sprintf("%dh%dm", hours, minutes)

		//spew.Dump(client)

		currentWork, xerr := client.WorkerState.CurrentWork.Get_list(false)
		if xerr != nil {
			logger.Error("(CheckClient) Get_current_work: %s", xerr.Error())
			return
		}

		logger.Debug(3, "(CheckClient) client %s has %d workunits", client.ID, len(currentWork))

		for _, workID := range currentWork {
			var workidStr string
			workidStr, err = workID.String()
			if err != nil {
				return
			}
			logger.Debug(3, "(CheckClient) client %s has work %s", client.ID, workidStr)
			work, ok, zerr := qm.workQueue.all.Get(workID)
			if zerr != nil {
				logger.Warning("(CheckClient) failed getting work %s from workQueue: %s", workidStr, zerr.Error())
				continue
			}
			if !ok {
				logger.Error("(CheckClient) work %s not in workQueue", workidStr) // this could happen wehen server was restarted but worker does not know yet
				continue
			}
			logger.Debug(3, "(CheckClient) work.State: %s", work.State)
			if work.State == WORK_STAT_RESERVED {
				qm.workQueue.StatusChange(workID, work, WORK_STAT_CHECKOUT, "")
			}
		}

	} else {
		client.Online = false

		ok = false
	}
	return
}

// ClientChecker _
func (qm *CQMgr) ClientChecker() {
	logger.Info("(ClientChecker) starting")
	if conf.CPUPROFILE != "" {
		f, err := os.Create(conf.CPUPROFILE)
		if err != nil {
			logger.Error(err.Error())
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	for {
		time.Sleep(30 * time.Second)

		if conf.MEMPROFILE != "" {
			f, err := os.Create(conf.MEMPROFILE)
			if err != nil {
				logger.Error("could not create memory profile: %s", err.Error())
			}
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				logger.Error("could not write memory profile: %s", err.Error())
			}
			f.Close()
		}

		logger.Debug(3, "(ClientChecker) time to update client list....")

		deleteClients := []string{}

		clientList, xerr := qm.clientMap.GetClients() // this uses a list of pointers to prevent long locking of the CLientMap
		if xerr != nil {
			logger.Error("(ClientChecker) GetClients: %s", xerr.Error())
			continue
		}
		logger.Debug(3, "(ClientChecker) check %d clients", len(clientList))
		for _, client := range clientList {
			ok, xerr := qm.CheckClient(client)
			if xerr != nil {
				logger.Error("(ClientChecker) CheckClient: %s", xerr.Error())
				continue
			}
			if !ok {
				deleteClients = append(deleteClients, client.ID)
			}
		}

		// Now delete clients
		if len(deleteClients) > 0 {
			qm.DeleteClients(deleteClients)

		}
	}
}

// DeleteClients _
func (qm *CQMgr) DeleteClients(deleteClients []string) {

	for _, clientID := range deleteClients {
		qm.RemoveClient(clientID, true)
	}

}

// ClientHeartBeat this is invoked on server side when clients sends heartbeat
func (qm *CQMgr) ClientHeartBeat(id string, cg *ClientGroup, workerstate WorkerState) (hbmsg HeartbeatInstructions, err error) {
	hbmsg = make(map[string]string, 1)
	client, ok, xerr := qm.GetClient(id, true)
	if xerr != nil {
		err = xerr
		return
	}

	if !ok {
		err = errors.New(e.ClientNotFound)
		return
	}

	err = client.LockNamed("ClientHeartBeat")
	if err != nil {
		return
	}
	defer client.Unlock()

	// If the name of the clientgroup (from auth token) does not match the name in the client retrieved, throw an error
	if cg != nil && client.Group != cg.Name {
		return nil, errors.New(e.ClientGroupBadName)
	}
	client.Tag = true
	err = client.SetOnline(true, false)
	if err != nil {
		return
	}

	workerstate.CurrentWork.FillMap() // fix struct by moving values from Data array into internal map (was not exported)

	client.WorkerState = workerstate // TODO could do a comparsion with assigned state here

	_ = client.UpdateStatus(false)

	logger.Debug(3, "HeartBeatFrom: client %s", id)

	//get suspended workunit that need the client to discard
	currentWork, xerr := client.CurrentWork.Get_list(false)
	discard := []string{}

	for _, workID := range currentWork {
		var work *Workunit

		work, ok, err = qm.workQueue.all.Get(workID)
		if err != nil {
			return
		}

		if !ok {
			workIDSstr, _ := workID.String()
			// server does not know about the work the client id working on
			logger.Error("(ClientHeartBeat) Client was working on unknown workunit. Told him to discard.")
			discard = append(discard, workIDSstr)
			continue
		}

		if work.State == WORK_STAT_SUSPEND {
			discard = append(discard, work.Id)
		}

	}
	if len(discard) > 0 {
		hbmsg["discard"] = strings.Join(discard, ",")
	}
	//if client.Status == CLIENT_STAT_DELETED {
	//	hbmsg["stop"] = id
	//}

	hbmsg["server-uuid"] = ServerUUID

	return

}

// RegisterNewClient This can be a new client or an old client that re-registers
func (qm *CQMgr) RegisterNewClient(files FormFiles, cg *ClientGroup) (client *Client, err error) {
	logger.Debug(3, "RegisterNewClient called")
	if _, ok := files["profile"]; ok {
		client, err = NewProfileClient(files["profile"].Path)
		os.Remove(files["profile"].Path)
		if err != nil {
			err = fmt.Errorf("NewProfileClient returned: %s", err.Error())
			return
		}

	} else {

		err = fmt.Errorf("Profile file not provided")
		return
		//client = NewClient()
	}

	err = client.LockNamed("RegisterNewClient")
	if err != nil {
		return
	}
	clientGroup := client.Group
	clientID, _ := client.GetID(false)
	client.Unlock()

	// If clientgroup is nil at this point, create a publicly owned clientgroup, with the provided group name (if one doesn't exist with the same name)
	if cg == nil {
		// See if clientgroup already exists with this name
		// If it does and it does not have "public" execution rights, throw error
		// If it doesn't, create one owned by public, and continue with client registration
		// Otherwise proceed with client registration.
		cg, _ = LoadClientGroupByName(clientGroup)

		if cg != nil {
			rights := cg.ACL.Check("public")
			if rights["execute"] == false {
				err = errors.New("Clientgroup with the group specified by your client exists, but you cannot register with it, without clientgroup token")
				return
			}
		} else {
			u := &user.User{Uuid: "public"}
			cg, err = CreateClientGroup(clientGroup, u)
			if err != nil {
				err = fmt.Errorf("CreateClientGroup returned: %s", err.Error())
				return nil, err
			}
		}
	}
	// If the name of the clientgroup (from auth token) does not match the name in the client profile, throw an error
	if cg != nil && clientGroup != cg.Name {
		return nil, errors.New(e.ClientGroupBadName)
	}

	// check if client is already known
	var oldClient *Client
	var oldClientExists bool
	oldClient, oldClientExists, err = qm.GetClient(clientID, true)
	if err != nil {
		return
	}

	if oldClientExists {
		// copy values from new client to old client
		oldClient.CurrentWork = client.CurrentWork

		oldClient.Tag = true
		// new client struct will be deleted afterwards
	} else {

		client.Tag = true

		err = qm.AddClient(client, true)
		if err != nil {
			return
		}
	}
	return
}

// GetClientByUser _
func (qm *CQMgr) GetClientByUser(id string, u *user.User) (client *Client, err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filteredClientGroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.ACL.Owner == u.Uuid || u.Admin == true || cg.ACL.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.ACL.Owner == "public") {
			filteredClientGroups[cg.Name] = true
		}
	}

	client, ok, err := qm.GetClient(id, true)
	if err != nil {
		return
	}
	if ok {
		if val, exists := filteredClientGroups[client.Group]; exists == true || val == true {
			return client, nil
		}
	}
	return nil, errors.New(e.ClientNotFound)
}

// GetAllClientsByUser _
func (qm *CQMgr) GetAllClientsByUser(u *user.User) (clients []*Client, err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filteredClientGroups := map[string]bool{}
	for _, cg := range *clientgroups {
		rights := cg.ACL.Check(u.Uuid)
		if (u.Uuid != "public" && (cg.ACL.Owner == u.Uuid || rights["read"] == true || u.Admin == true || cg.ACL.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.ACL.Owner == "public") {
			filteredClientGroups[cg.Name] = true
		}
	}

	allClients, err := qm.clientMap.GetClients()
	if err != nil {
		return
	}

	for _, client := range allClients {
		if val, exists := filteredClientGroups[client.Group]; exists == true && val == true {
			clients = append(clients, client)
		}
	}

	return
}

// func (qm *CQMgr) DeleteClientByUser(id string, u *user.User) (err error) {
// 	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
// 	q := bson.M{}
// 	clientgroups := new(ClientGroups)
// 	dbFindClientGroups(q, clientgroups)
// 	filteredClientGroups := map[string]bool{}
// 	for _, cg := range *clientgroups {
// 		if (u.Uuid != "public" && (cg.ACL.Owner == u.Uuid || u.Admin == true || cg.ACL.Owner == "public")) ||
// 			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.ACL.Owner == "public") {
// 			filteredClientGroups[cg.Name] = true
// 		}
// 	}
//
// 	client, ok, err := qm.GetClient(id, true)
// 	if err != nil {
// 		return
// 	}
// 	if ok {
// 		if val, exists := filteredClientGroups[client.Group]; exists == true && val == true {
// 			err = qm.DeleteClient(client)
// 			return
// 		}
// 		return errors.New(e.UnAuth)
// 	}
// 	return errors.New(e.ClientNotFound)
// }

// SuspendClient use id OR client
func (qm *CQMgr) SuspendClient(id string, client *Client, reason string, clientWriteLock bool) (err error) {

	if client == nil {
		var ok bool
		client, ok, err = qm.GetClient(id, true)
		if err != nil {
			return
		}
		if !ok {
			err = errors.New(e.ClientNotFound)
			return
		}
	}

	if clientWriteLock {
		err = client.LockNamed("SuspendClient")
		if err != nil {
			return
		}
		defer client.Unlock()
	}

	isSuspended, err := client.GetSuspended(false)
	if err != nil {
		return
	}

	if isSuspended {
		err = errors.New(e.ClientNotActive)
		return
	}

	client.Suspend(reason, false)

	err = qm.ReQueueWorkunitByClient(client, false)
	if err != nil {
		return
	}

	return

}

// SuspendAllClients _
func (qm *CQMgr) SuspendAllClients(reason string) (count int, err error) {
	clients, err := qm.ListClients()
	if err != nil {
		return
	}
	for _, id := range clients {
		if err := qm.SuspendClient(id, nil, reason, true); err == nil {
			count++
		}
	}
	return
}

// SuspendClientByUser _
func (qm *CQMgr) SuspendClientByUser(id string, u *user.User, reason string) (err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filteredClientGroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.ACL.Owner == u.Uuid || u.Admin == true || cg.ACL.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.ACL.Owner == "public") {
			filteredClientGroups[cg.Name] = true
		}
	}

	client, ok, err := qm.GetClient(id, true)
	if err != nil {
		return
	}
	if ok {
		err = errors.New(e.ClientNotFound)
		return
	}

	val, exists := filteredClientGroups[client.Group]
	if exists == true && val == true {

		err = qm.SuspendClient("", client, reason, true)
		if err != nil {
			return
		}

	}
	return errors.New(e.UnAuth)

}

// SuspendAllClientsByUser _
func (qm *CQMgr) SuspendAllClientsByUser(u *user.User, reason string) (count int, err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filteredClientGroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.ACL.Owner == u.Uuid || u.Admin == true || cg.ACL.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.ACL.Owner == "public") {
			filteredClientGroups[cg.Name] = true
		}
	}

	clients, err := qm.clientMap.GetClients()
	if err != nil {
		return
	}
	for _, client := range clients {

		isSuspended, xerr := client.GetSuspended(true)
		if xerr != nil {
			err = xerr
			return
		}

		group, xerr := client.GetGroup(true)
		if xerr != nil {
			return
		}

		if val, exists := filteredClientGroups[group]; exists == true && val == true && isSuspended {
			err = qm.SuspendClient("", client, reason, true)
			if err != nil {
				return
			}
			count++
		}

	}

	return
}

// ResumeClient _
func (qm *CQMgr) ResumeClient(id string) (err error) {
	client, ok, err := qm.GetClient(id, true)
	if err != nil {
		return
	}
	if !ok {
		return errors.New(e.ClientNotFound)
	}

	err = client.Resume(true)

	if err != nil {
		return
	}

	return

}

// ResumeClientByUser _
func (qm *CQMgr) ResumeClientByUser(id string, u *user.User) (err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filteredClientGroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.ACL.Owner == u.Uuid || u.Admin == true || cg.ACL.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.ACL.Owner == "public") {
			filteredClientGroups[cg.Name] = true
		}
	}

	client, ok, err := qm.GetClient(id, true)
	if err != nil {
		return
	}
	if !ok {
		return errors.New(e.ClientNotFound)
	}

	if val, exists := filteredClientGroups[client.Group]; exists == true && val == true {
		err = client.Resume(true)
		if err != nil {
			return
		}

		return
	}
	return errors.New(e.UnAuth)

}

// ResumeSuspendedClients _
func (qm *CQMgr) ResumeSuspendedClients() (count int, err error) {

	clients, err := qm.clientMap.GetClients()
	if err != nil {
		return
	}
	for _, client := range clients {

		isSuspended, xerr := client.GetSuspended(true)
		if xerr != nil {
			err = xerr
			return
		}

		if isSuspended {
			//qm.ClientStatusChange(client.ID, CLIENT_STAT_ACTIVE_IDLE)
			err = client.Resume(true)
			if err != nil {
				return
			}
			count++
		}

	}

	return
}

// ResumeSuspendedClientsByUser _
func (qm *CQMgr) ResumeSuspendedClientsByUser(u *user.User) (count int) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filteredClientGroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.ACL.Owner == u.Uuid || u.Admin == true || cg.ACL.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.ACL.Owner == "public") {
			filteredClientGroups[cg.Name] = true
		}
	}

	clients, err := qm.clientMap.GetClients()
	if err != nil {
		return
	}
	for _, client := range clients {

		isSuspended, xerr := client.GetSuspended(true)
		if xerr != nil {
			err = xerr
			return
		}

		if !isSuspended {
			continue
		}

		group, err := client.GetGroup(true)

		val, exists := filteredClientGroups[group]
		if exists == true && val == true {

			err = client.Resume(true)
			if err != nil {
				return
			}
			count++
		}

	}

	return count
}

// UpdateSubClients _
func (qm *CQMgr) UpdateSubClients(id string, count int) (err error) {
	client, ok, err := qm.GetClient(id, true)
	if err != nil {
		return
	}
	if ok {
		client.SubClients = count
	}
	return
}

// UpdateSubClientsByUser _
func (qm *CQMgr) UpdateSubClientsByUser(id string, count int, u *user.User) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filteredClientGroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.ACL.Owner == u.Uuid || u.Admin == true || cg.ACL.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.ACL.Owner == "public") {
			filteredClientGroups[cg.Name] = true
		}
	}

	client, ok, err := qm.GetClient(id, true)
	if err != nil {
		return
	}
	if ok {
		if val, exists := filteredClientGroups[client.Group]; exists == true && val == true {
			client.SubClients = count

		}
	}
}

//--------end client methods-------

//-------start of workunit methods---

// CheckoutWorkunits _
func (qm *CQMgr) CheckoutWorkunits(reqPolicy string, clientID string, client *Client, availableBytes int64, num int) (workunits []*Workunit, err error) {

	logger.Debug(3, "run CheckoutWorkunits for client %s", clientID)

	//precheck if the client is registered
	//client, hasClient, err := qm.GetClient(client_id, true)
	//if err != nil {
	//	return
	//}
	//if !hasClient {
	//	return nil, errors.New(e.ClientNotFound)
	//}

	err = client.LockNamed("CheckoutWorkunits serving " + clientID)
	if err != nil {
		return
	}
	//status := client.Status
	responseChannel := client.coAckChannel

	workLength, _ := client.CurrentWork.Length(false)
	client.Unlock()

	if workLength > 0 {
		logger.Error("Client %s wants to checkout work, but still has work: workLength=%d", clientID, workLength)
		return nil, errors.New(e.ClientBusy)
	}

	isSuspended, err := client.GetSuspended(true)
	if err != nil {
		return
	}

	if isSuspended {
		err = errors.New(e.ClientSuspended)
		return
	}

	//if status == CLIENT_STAT_DELETED {
	//	qm.RemoveClient(client_id, false)
	//	return nil, errors.New(e.ClientDeleted)
	//}

	//req := CoReq{policy: req_policy, fromclient: client_id, available: available_bytes, count: num, response: client.coAckChannel}
	req := CheckoutRequest{policy: reqPolicy, fromclient: clientID, available: availableBytes, count: num, response: responseChannel}

	logger.Debug(3, "(CheckoutWorkunits) %s qm.coReq <- req", clientID)
	// request workunit

	//err = qm.requestQueue.Push(&req)
	//if err != nil {
	//	logger.Error("Work request by client %s rejected, request queue is full", client_id)
	//	err = fmt.Errorf("Too many work requests - Please try again later")
	//	return
	//}
	select {
	case qm.coReq <- req:
	default:
		logger.Error("Work request by client %s rejected, request queue is full", clientID)
		err = fmt.Errorf("Too many work requests - Please try again later")
		return

	}

	logger.Debug(3, "(CheckoutWorkunits) %s client.Get_Ack()", clientID)
	//ack := <-qm.coAck

	var ack CoAck
	// get workunit
	lock, err := client.RLockNamed("CheckoutWorkunits waiting for ack, clientID: " + clientID)
	if err != nil {
		return
	}
	ack, err = client.GetAck()
	client.RUnlockNamed(lock)

	logger.Debug(3, "(CheckoutWorkunits) %s got ack", clientID)
	if err != nil {
		return
	}

	logger.Debug(3, "(CheckoutWorkunits) %s got ack", clientID)

	if ack.err != nil {
		logger.Debug(3, "(CheckoutWorkunits) %s ack.err: %s", clientID, ack.err.Error())
		return ack.workunits, ack.err
	}

	addedWork := 0
	for _, work := range ack.workunits {
		workID := work.Workunit_Unique_Identifier
		//work_id, xerr := work.Workunit_Unique_Identifier// New_Workunit_Unique_Identifier_FromString(work.Id)
		//if xerr != nil {
		//	return
		//}
		err = client.AssignedWork.Add(workID)
		if err != nil {
			return
		}
		addedWork++
	}

	//if added_work > 0 && status == CLIENT_STAT_ACTIVE_IDLE {
	//	client.Set_Status(CLIENT_STAT_ACTIVE_BUSY, true)
	//}

	logger.Debug(3, "(CheckoutWorkunits) %s finished", clientID)
	return ack.workunits, ack.err
}

// GetWorkByID _
func (qm *CQMgr) GetWorkByID(id Workunit_Unique_Identifier) (workunit *Workunit, err error) {
	workunit, ok, err := qm.workQueue.Get(id)
	if err != nil {
		return
	}
	if !ok {
		var workStr string
		workStr, err = id.String()
		if err != nil {
			err = fmt.Errorf("() workid.String() returned: %s", err.Error())
			return
		}
		err = fmt.Errorf("no workunit found with id %s", workStr)
	}
	return
}

// NotifyWorkStatus _
func (qm *CQMgr) NotifyWorkStatus(notice Notice) {
	qm.feedback <- notice
	return
}

// when popWorks is called, the client should already be locked
func (qm *CQMgr) popWorks(req CheckoutRequest) (clientSpecificWorkunits []*Workunit, err error) {

	clientID := req.fromclient

	client, ok, err := qm.clientMap.Get(clientID, true) // locks the clientmap
	if err != nil {
		return
	}
	if !ok {
		err = fmt.Errorf("(popWorks) Client %s not found", clientID)
		return
	}

	logger.Debug(3, "(popWorks) starting for client: %s", clientID)

	filtered, stats, err := qm.filterWorkByClient(client)
	if err != nil {
		err = fmt.Errorf("(popWorks) filterWorkByClient returned: %s", err.Error())
		return
	}

	if len(filtered) == 0 {
		var statJSONByte []byte
		statJSONByte, err = json.Marshal(stats)
		if err != nil {
			statJSONByte = []byte("failed")
		}
		clientGroup := ""
		clientGroup, err = client.GetGroup(false)
		if err != nil {
			clientGroup = ""
		}
		logger.Error("(popWorks) filterWorkByClient returned no workunits for client %s (%s), stats: %s", clientID, clientGroup, statJSONByte)
		err = errors.New(e.NoEligibleWorkunitFound)

		return
	}
	clientSpecificWorkunits, err = qm.workQueue.selectWorkunits(filtered, req.policy, req.available, req.count)
	if err != nil {
		err = fmt.Errorf("(popWorks) selectWorkunits returned: %s", err.Error())
		return
	}
	//get workunits successfully, put them into coWorkMap
	for _, work := range clientSpecificWorkunits {
		work.Client = clientID
		work.CheckoutTime = time.Now()
		//qm.workQueue.Put(work) TODO isn't that already in the queue ?
		qm.workQueue.StatusChange(work.Workunit_Unique_Identifier, work, WORK_STAT_CHECKOUT, "")
	}

	logger.Debug(3, "(popWorks) done with client: %s ", clientID)
	return
}

// client has to be read-locked
func (qm *CQMgr) filterWorkByClient(client *Client) (workunits WorkList, s FilterWorkStats, err error) {

	s = FilterWorkStats{0, 0, 0, 0}

	if client == nil {
		err = fmt.Errorf("(filterWorkByClient) client == nil")
		return
	}

	clientid := client.ID

	if clientid == "" {
		err = fmt.Errorf("(filterWorkByClient) clientid empty")
		return
	}

	logger.Debug(3, "(filterWorkByClient) starting for client: %s", clientid)

	workunitList, err := qm.workQueue.Queue.GetWorkunits()
	if err != nil {
		err = fmt.Errorf("(filterWorkByClient) qm.workQueue.Queue.GetWorkunits retruns: %s", err.Error())
		return
	}

	if len(workunitList) == 0 {
		err = errors.New(e.QueueEmpty)
		return
	}

	logger.Debug(3, "(filterWorkByClient) GetWorkunits() returned: %d", len(workunitList))
	for _, workunit := range workunitList {
		s.Total++
		id := workunit.Id
		logger.Debug(3, "check if job %s would fit client %s", id, clientid)

		//skip works that are in the client's skip-list
		if client.ContainsSkipWorkNolock(workunit.Id) {
			logger.Debug(3, "2) workunit %s is in Skip_work list of the client %s)", id, clientid)
			s.SkipWork++
			continue
		}
		//skip works that have dedicate client groups which this client doesn't belong to
		if len(workunit.Info.ClientGroups) > 0 {
			eligibleGroups := strings.Split(workunit.Info.ClientGroups, ",")
			if !contains(eligibleGroups, client.Group) {
				logger.Debug(3, fmt.Sprintf("3) !contains(eligibleGroups, client.Group) %s", id))
				s.WrongClientgroup++
				continue
			}
		}
		//append works whos apps are supported by the client
		if contains(client.Apps, workunit.Cmd.Name) || contains(client.Apps, conf.ALL_APP) {
			logger.Debug(3, "append job %s to list of client %s", id, clientid)
			workunits = append(workunits, workunit)
		} else {
			s.WrongApp++
			logger.Debug(2, "3) contains(client.Apps, work.Cmd.Name) || contains(client.Apps, conf.ALL_APP) %s", id)
		}
	}
	logger.Debug(3, "done with filterWorkByClient() for client: %s", clientid)

	// error should be created by the caller, not in this function
	//if len(workunits) == 0 {
	//	err = errors.New(e.NoEligibleWorkunitFound)
	//	return
	//}

	return
}

// lock: read-lock for client
//func (qm *CQMgr) getWorkByClient(clientid string, lock bool) (ids []string) {
//	client, ok := qm.GetClient(clientid, true)
//	if ok {
//		ids = Get_current_work(lock)
//	}
//	return
//}

//handle feedback from a client about the execution of a workunit
func (qm *CQMgr) handleNoticeWorkDelivered(notice *Notice) (err error) {
	//to be implemented for proxy or server
	return
}

// FetchDataToken _
func (qm *CQMgr) FetchDataToken(workid string, clientid string) (token string, err error) {
	//to be implemented for proxy or server
	return
}

// ShowWorkunits _
func (qm *CQMgr) ShowWorkunits(status string) (workunits []*Workunit, err error) {
	workunitList, err := qm.workQueue.GetAll()
	if err != nil {
		return
	}
	for _, work := range workunitList {
		if work.State == status || status == "" {
			workunits = append(workunits, work)
		}
	}
	return
}

// ShowWorkunitsByUser _
func (qm *CQMgr) ShowWorkunitsByUser(status string, u *user.User) (workunits []*Workunit) {
	// Only returns workunits of jobs that the user has read access to or is the owner of.  If user is admin, return all.
	workunitList, err := qm.workQueue.GetAll()
	if err != nil {
		return
	}
	for _, work := range workunitList {
		// skip loading jobs from db if user is admin
		if u.Admin == true {
			if work.State == status || status == "" {
				workunits = append(workunits, work)
			}
		} else {
			jobid := work.JobId

			if job, err := GetJob(jobid); err == nil {
				rights := job.Acl.Check(u.Uuid)
				if job.Acl.Owner == u.Uuid || rights["read"] == true {
					if work.State == status || status == "" {
						workunits = append(workunits, work)
					}
				}
			}

		}
	}
	return workunits
}

// EnqueueWorkunit _
func (qm *CQMgr) EnqueueWorkunit(work *Workunit) (err error) {
	err = qm.workQueue.Add(work)
	return
}

// ReQueueWorkunitByClient _
func (qm *CQMgr) ReQueueWorkunitByClient(client *Client, clientWriteLock bool) (err error) {

	worklist, err := client.CurrentWork.Get_list(clientWriteLock)
	if err != nil {
		return
	}
	for _, workid := range worklist {
		workidStr, _ := workid.String()
		logger.Debug(3, "(ReQueueWorkunitByClient) try to requeue work %s", workidStr)
		work, hasWork, xerr := qm.workQueue.Get(workid)
		if xerr != nil {
			logger.Error("(ReQueueWorkunitByClient) error checking workunit %s", workidStr)
			continue
		}

		if !hasWork {
			logger.Error("(ReQueueWorkunitByClient) Workunit %s not found", workidStr)
			continue
		}

		jobid := work.JobId

		job, xerr := GetJob(jobid)
		if xerr != nil {
			err = xerr
			return
		}
		jobState, err := job.GetState(true)
		if err != nil {
			logger.Error("(ReQueueWorkunitByClient) dbGetJobField: %s", err.Error())
			continue
		}

		if contains(JOB_STATS_ACTIVE, jobState) { //only requeue workunits belonging to active jobs (rule out suspended jobs)
			if work.Client == client.ID {
				qm.workQueue.StatusChange(workid, work, WORK_STAT_QUEUED, "")
				var workStr string
				workStr, err = workid.String()
				if err != nil {
					err = fmt.Errorf("(ReQueueWorkunitByClient) workid.String() returned: %s", err.Error())
					return err
				}
				logger.Event(event.WORK_REQUEUE, "workid="+workStr)
			} else {

				otherClientID := work.Client

				otherClient, ok, xerr := qm.GetClient(otherClientID, true)
				if xerr != nil {
					logger.Error("(ReQueueWorkunitByClient) otherClient: %s ", xerr.Error())
					continue
				}
				if ok {
					// otherClient exists (if otherclient does not exist, that is ok....)
					otherClientHasWork, err := otherClient.CurrentWork.Has(workid)
					if err != nil {
						logger.Error("(ReQueueWorkunitByClient) Current_work_has: %s ", err.Error())
						continue
					}
					if !otherClientHasWork {
						// other client has not this workunit,
						qm.workQueue.StatusChange(workid, work, WORK_STAT_SUSPEND, "workunit has wrong client info")
						continue
					}
				}
				// client does not exists of has different workunit
				// no status change

			}
		} else {
			qm.workQueue.StatusChange(workid, work, WORK_STAT_SUSPEND, "workunit does not belong to an actove job")
		}

	}
	return
}

//---end of workunit methods
