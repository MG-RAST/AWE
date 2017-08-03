package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/user"
	"gopkg.in/mgo.v2/bson"
	"os"
	"strings"
	"time"
)

// this struct is embedded in ServerMgr
type CQMgr struct {
	clientMap    ClientMap
	workQueue    *WorkQueue
	suspendQueue bool
	coReq        chan CoReq  //workunit checkout request (WorkController -> qmgr.Handler)
	feedback     chan Notice //workunit execution feedback (WorkController -> qmgr.Handler)
	coSem        chan int    //semaphore for checkout (mutual exclusion between different clients)
}

//--------mgr methods-------

func (qm *CQMgr) ClientHandle() {
	// this code is not beeing used
}

// show functions used in debug
func (qm *CQMgr) ShowWorkQueue() {
	length, err := qm.workQueue.Len()
	if err != nil {
		logger.Error("error: %s", err.Error())
		return
	}
	logger.Debug(1, fmt.Sprintf("current queuing workunits (%d)", length))
	workunits, err := qm.workQueue.GetAll()
	if err != nil {
		return
	}
	for _, workunit := range workunits {
		id := workunit.Id
		logger.Debug(1, fmt.Sprintf("workid=%s", id))
	}
	return
}

//--------accessor methods-------

func (qm *CQMgr) GetClientMap() *ClientMap {
	return &qm.clientMap
}

func (qm *CQMgr) AddClient(client *Client, lock bool) (err error) {
	err = qm.clientMap.Add(client, lock)
	return
}

func (qm *CQMgr) GetClient(id string, lock_clientmap bool) (client *Client, ok bool, err error) {
	return qm.clientMap.Get(id, lock_clientmap)
}

// lock is for clientmap
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
	logger.Event(event.CLIENT_UNREGISTER, "clientid="+client.Id+";name="+client.Name)
	//requeue unfinished workunits associated with the failed client
	err = qm.ReQueueWorkunitByClient(client, true)
	if err != nil {
		logger.Error("(CheckClient) %s", err.Error())
	}

	err = qm.clientMap.Delete(id, lock)
	return
}

func (qm *CQMgr) DeleteClient(client *Client) (err error) {
	err = qm.ClientStatusChange(client, CLIENT_STAT_DELETED, true)
	return
}

func (qm *CQMgr) DeleteClientById(id string) (err error) {
	err = qm.ClientIdStatusChange(id, CLIENT_STAT_DELETED, true)
	return
}

func (qm *CQMgr) ClientIdStatusChange(id string, new_status string, client_write_lock bool) (err error) {
	client, ok, err := qm.clientMap.Get(id, true)
	if err != nil {
		return
	}
	if ok {
		err = client.Set_Status(new_status, client_write_lock)
		return
	}
	return errors.New(e.ClientNotFound)
}

func (qm *CQMgr) ClientStatusChange(client *Client, new_status string, client_write_lock bool) (err error) {
	client.Set_Status(new_status, client_write_lock)
	return

}

func (qm *CQMgr) HasClient(id string, lock_clientmap bool) (has bool, err error) {
	_, ok, err := qm.clientMap.Get(id, lock_clientmap)
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

func (qm *CQMgr) ListClients() (ids []string, err error) {
	//qm.clientMap.RLock()
	//defer qm.clientMap.RUnlock()
	//for id, _ := range qm.clientMap {
	//	ids = append(ids, id)
	//}
	return qm.clientMap.GetClientIds()
}

//--------client methods-------

func (qm *CQMgr) CheckClient(client *Client) (ok bool, err error) {
	ok = true
	err = client.LockNamed("ClientChecker")
	if err != nil {
		return
	}
	defer client.Unlock()

	if client.Tag == true {
		// *** Client is active

		client.Tag = false
		total_minutes := int(time.Now().Sub(client.RegTime).Minutes())
		hours := total_minutes / 60
		minutes := total_minutes % 60
		client.Serve_time = fmt.Sprintf("%dh%dm", hours, minutes)

		current_work, xerr := client.Get_current_work(false)
		if xerr != nil {
			logger.Error("(CheckClient) Get_current_work: %s", xerr.Error())
			return
		}
		for _, work_id := range current_work {
			work, ok, zerr := qm.workQueue.all.Get(work_id)
			if zerr != nil {
				continue
			}
			if !ok {
				continue
			}
			if work.State == WORK_STAT_RESERVED {
				qm.workQueue.StatusChange(work_id, work, WORK_STAT_CHECKOUT)
			}
		}

	} else {
		// *** Client is NOT active

		//delete the client from client map
		//qm.RemoveClient(client.Id)
		ok = false
	}
	return
}

func (qm *CQMgr) ClientChecker() {
	logger.Info("ClientChecker starting")
	for {
		time.Sleep(30 * time.Second)
		logger.Debug(3, "(ClientChecker) time to update client list....")

		delete_clients := []string{}

		client_list, xerr := qm.clientMap.GetClients() // this uses a list of pointers to prevent long locking of the CLientMap
		if xerr != nil {
			logger.Error("ClientChecker/GetClients: %s", xerr.Error())
			continue
		}
		logger.Debug(3, "(ClientChecker) check %d clients", len(client_list))
		for _, client := range client_list {
			ok, xerr := qm.CheckClient(client)
			if xerr != nil {
				logger.Error("ClientChecker/CheckClient: %s", xerr.Error())
				continue
			}
			if !ok {
				delete_clients = append(delete_clients, client.Id)
			}
		}

		// Now delete clients
		if len(delete_clients) > 0 {
			qm.DeleteClients(delete_clients)

		}
	}
}

func (qm *CQMgr) DeleteClients(delete_clients []string) {

	for _, client_id := range delete_clients {
		qm.RemoveClient(client_id, true)
	}

}

func (qm *CQMgr) ClientHeartBeat(id string, cg *ClientGroup) (hbmsg HBmsg, err error) {
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

	logger.Debug(3, "HeartBeatFrom:"+"clientid="+id+",name="+client.Name)

	//get suspended workunit that need the client to discard
	current_work, xerr := client.Get_current_work(false)
	suspended := []string{}

	for _, work_id := range current_work {
		work, ok, zerr := qm.workQueue.all.Get(work_id)
		if err != nil {
			err = zerr
			return
		}
		if !ok {
			continue
		}

		if work.State == WORK_STAT_SUSPEND {
			suspended = append(suspended, work.Id)
		}

	}
	if len(suspended) > 0 {
		hbmsg["discard"] = strings.Join(suspended, ",")
	}
	if client.Status == CLIENT_STAT_DELETED {
		hbmsg["stop"] = id
	}

	hbmsg["server-uuid"] = Server_UUID

	return

}

// This can be a new client or an old client that re-registers
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
	client_group := client.Group
	client_id, _ := client.Get_Id(false)
	client.Unlock()

	// If clientgroup is nil at this point, create a publicly owned clientgroup, with the provided group name (if one doesn't exist with the same name)
	if cg == nil {
		// See if clientgroup already exists with this name
		// If it does and it does not have "public" execution rights, throw error
		// If it doesn't, create one owned by public, and continue with client registration
		// Otherwise proceed with client registration.
		cg, _ = LoadClientGroupByName(client_group)

		if cg != nil {
			rights := cg.Acl.Check("public")
			if rights["execute"] == false {
				err = errors.New("Clientgroup with the group specified by your client exists, but you cannot register with it, without clientgroup token.")
				return nil, err
			}
		} else {
			u := &user.User{Uuid: "public"}
			cg, err = CreateClientGroup(client_group, u)
			if err != nil {
				err = fmt.Errorf("CreateClientGroup returned: %s", err.Error())
				return nil, err
			}
		}
	}
	// If the name of the clientgroup (from auth token) does not match the name in the client profile, throw an error
	if cg != nil && client_group != cg.Name {
		return nil, errors.New(e.ClientGroupBadName)
	}

	all_workunits_map := make(map[string]bool)

	// collect current_work from new client object
	current_work, err := client.Get_current_work(true)
	if err != nil {
		return
	}

	if len(current_work) > 1 {
		logger.Error("Client %s reports %d elements in Current_work", client_id, len(current_work)) // TODO this is a temprorary check. Remove this if you want to support multiple workunits per client.
		err = fmt.Errorf("Client reports too many current workunits")
		return
	}

	for _, workid := range current_work {
		all_workunits_map[workid] = true
	}

	// check if client is already known
	old_client, old_client_exists, err := qm.GetClient(client_id, true)
	if err != nil {
		return
	}
	if old_client_exists {

		// client is already known. Check current_work, then remove old client object

		old_client_current_work, xerr := old_client.Get_current_work(true)
		if xerr != nil {
			err = xerr
			return
		}

		for _, work_id := range old_client_current_work {

			_, ok := all_workunits_map[work_id]
			if ok {
				// all good, new client is still working on the same workunit the old client did work on.
				continue
			}

			// the client does not know about this workunit anymore, make sure the workunit also knows that
			work, ok, xerr := qm.workQueue.Get(work_id)
			if xerr != nil {
				logger.Error("(RegisterNewClient) %s", xerr.Error())
				continue
			}
			if !ok {
				// workunit not found, that is ok...
				continue
			}

			if work.Client == client_id {
				work.Client = ""
				work_state := work.State
				if (work_state != WORK_STAT_SUSPEND) && (work_state != WORK_STAT_QUEUED) {
					qm.workQueue.StatusChange(work_id, work, WORK_STAT_QUEUED)
				}
			}

			continue

		}

		// keep old client, copy relevant values
		//err = qm.RemoveClient(client_id, true)
		//if err != nil {
		//	return
		//}
	}

	keep_work := []string{}

	for workid, _ := range all_workunits_map {
		work, ok, xerr := qm.workQueue.Get(workid)
		if xerr != nil {
			logger.Error("(RegisterNewClient) %s", xerr.Error())
			continue
		}

		if !ok {
			// client has workunit the server does not know about !? (e.g. after reboot)
			continue
		}

		if work.Client == client_id {
			// ok: workunits exists and knows where it belongs to
			work_state := work.State

			// fix state if needed
			if (work_state != WORK_STAT_SUSPEND) && (work_state != WORK_STAT_CHECKOUT) {
				qm.workQueue.StatusChange(workid, work, WORK_STAT_CHECKOUT)
			}

			keep_work = append(keep_work, workid)
			continue
		}

		if work.Client == "" {
			// reclaim work unit
			work.Client = client_id
			qm.workQueue.StatusChange(workid, work, WORK_STAT_CHECKOUT)
			keep_work = append(keep_work, workid)
			continue
		}

		// work.Client != client.Id (workunit seems to be owned by another client, leave it alone)

		continue
	}

	if old_client_exists {
		// copy values from new client to old client

		old_client.Set_current_work(keep_work, true)
		old_client.Tag = true
		// new client struct will be deleted afterwards
	} else {
		client.Set_current_work(keep_work, true)
		client.Tag = true
		err = qm.AddClient(client, true) // locks clientMap
		if err != nil {
			return
		}
	}
	return
}

func (qm *CQMgr) GetClientByUser(id string, u *user.User) (client *Client, err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	client, ok, err := qm.GetClient(id, true)
	if err != nil {
		return
	}
	if ok {
		if val, exists := filtered_clientgroups[client.Group]; exists == true || val == true {
			return client, nil
		}
	}
	return nil, errors.New(e.ClientNotFound)
}

func (qm *CQMgr) GetAllClientsByUser(u *user.User) (clients []*Client, err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		rights := cg.Acl.Check(u.Uuid)
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || rights["read"] == true || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	all_clients, err := qm.clientMap.GetClients()
	if err != nil {
		return
	}

	for _, client := range all_clients {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
			clients = append(clients, client)
		}
	}

	return
}

func (qm *CQMgr) DeleteClientByUser(id string, u *user.User) (err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	client, ok, err := qm.GetClient(id, true)
	if err != nil {
		return
	}
	if ok {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
			err = qm.DeleteClient(client)
			return
		}
		return errors.New(e.UnAuth)
	}
	return errors.New(e.ClientNotFound)
}

// use id OR client
func (qm *CQMgr) SuspendClient(id string, client *Client, client_write_lock bool) (err error) {

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

	if client_write_lock {
		err = client.LockNamed("SuspendClient")
		if err != nil {
			return
		}
		defer client.Unlock()
	}

	status, err := client.Get_Status(false)
	if err != nil {
		return
	}

	if status == CLIENT_STAT_ACTIVE_IDLE || status == CLIENT_STAT_ACTIVE_BUSY {
		client.Set_Status(CLIENT_STAT_SUSPEND, false)
		//if err = qm.ClientStatusChange(id, CLIENT_STAT_SUSPEND); err != nil {
		//	return
		//}
		qm.ReQueueWorkunitByClient(client, false)
		return
	}
	return errors.New(e.ClientNotActive)

}

func (qm *CQMgr) SuspendAllClients() (count int, err error) {
	clients, err := qm.ListClients()
	if err != nil {
		return
	}
	for _, id := range clients {
		if err := qm.SuspendClient(id, nil, true); err == nil {
			count += 1
		}
	}
	return
}

func (qm *CQMgr) SuspendClientByUser(id string, u *user.User) (err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	client, ok, err := qm.GetClient(id, true)
	if err != nil {
		return
	}
	if ok {
		return errors.New(e.ClientNotFound)
	}

	if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
		err = client.LockNamed("SuspendClientByUser")
		if err != nil {
			return
		}
		defer client.Unlock()
		status, xerr := client.Get_Status(false)
		if xerr != nil {
			err = xerr
			return
		}
		if status == CLIENT_STAT_ACTIVE_IDLE || status == CLIENT_STAT_ACTIVE_BUSY {
			client.Set_Status(CLIENT_STAT_SUSPEND, false)
			//if err = qm.ClientStatusChange(id, CLIENT_STAT_SUSPEND); err != nil {
			//	return
			//}
			qm.ReQueueWorkunitByClient(client, false)
			return
		}
		return errors.New(e.ClientNotActive)
	}
	return errors.New(e.UnAuth)

}

func (qm *CQMgr) SuspendAllClientsByUser(u *user.User) (count int) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	clients, err := qm.clientMap.GetClients()
	if err != nil {
		return
	}
	for _, client := range clients {
		err = client.LockNamed("SuspendAllClientsByUser")
		if err != nil {
			continue
		}
		status, xerr := client.Get_Status(false)
		if xerr != nil {
			continue
		}
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true && (status == CLIENT_STAT_ACTIVE_IDLE || status == CLIENT_STAT_ACTIVE_BUSY) {
			qm.SuspendClient("", client, false)
			count += 1
		}
		client.Unlock()
	}

	return count
}

func (qm *CQMgr) ResumeClient(id string) (err error) {
	client, ok, err := qm.GetClient(id, true)
	if err != nil {
		return
	}
	if !ok {
		return errors.New(e.ClientNotFound)
	}

	err = client.LockNamed("ResumeClient")
	if err != nil {
		return
	}
	defer client.Unlock()
	if client.Status == CLIENT_STAT_SUSPEND {
		//err = qm.ClientStatusChange(id, CLIENT_STAT_ACTIVE_IDLE)
		client.Status = CLIENT_STAT_ACTIVE_IDLE
		return
	}
	return errors.New(e.ClientNotSuspended)

}

func (qm *CQMgr) ResumeClientByUser(id string, u *user.User) (err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	client, ok, err := qm.GetClient(id, true)
	if err != nil {
		return
	}
	if !ok {
		return errors.New(e.ClientNotFound)
	}

	err = client.LockNamed("ResumeClientByUser")
	if err != nil {
		return
	}
	defer client.Unlock()

	if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
		if client.Status == CLIENT_STAT_SUSPEND {
			//err = qm.ClientStatusChange(id, CLIENT_STAT_ACTIVE_IDLE)
			client.Status = CLIENT_STAT_ACTIVE_IDLE
			return
		}
		return errors.New(e.ClientNotSuspended)
	}
	return errors.New(e.UnAuth)

}

func (qm *CQMgr) ResumeSuspendedClients() (count int, err error) {

	clients, err := qm.clientMap.GetClients()
	if err != nil {
		return
	}
	for _, client := range clients {
		err = client.LockNamed("ResumeSuspendedClients")
		if err != nil {
			continue
		}
		if client.Status == CLIENT_STAT_SUSPEND {
			//qm.ClientStatusChange(client.Id, CLIENT_STAT_ACTIVE_IDLE)
			client.Status = CLIENT_STAT_ACTIVE_IDLE
			count += 1
		}
		client.Unlock()
	}

	return
}

func (qm *CQMgr) ResumeSuspendedClientsByUser(u *user.User) (count int) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	clients, err := qm.clientMap.GetClients()
	if err != nil {
		return
	}
	for _, client := range clients {
		err = client.LockNamed("ResumeSuspendedClientsByUser")
		if err != nil {
			continue
		}
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true && client.Status == CLIENT_STAT_SUSPEND {
			//qm.ClientStatusChange(client.Id, CLIENT_STAT_ACTIVE_IDLE)
			client.Status = CLIENT_STAT_ACTIVE_IDLE
			count += 1
		}
		client.Unlock()
	}

	return count
}

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

func (qm *CQMgr) UpdateSubClientsByUser(id string, count int, u *user.User) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	client, ok, err := qm.GetClient(id, true)
	if err != nil {
		return
	}
	if ok {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
			client.SubClients = count

		}
	}
}

//--------end client methods-------

//-------start of workunit methods---

func (qm *CQMgr) CheckoutWorkunits(req_policy string, client_id string, client *Client, available_bytes int64, num int) (workunits []*Workunit, err error) {

	logger.Debug(3, "run CheckoutWorkunits for client %s", client_id)

	//precheck if the client is registered
	//client, hasClient, err := qm.GetClient(client_id, true)
	//if err != nil {
	//	return
	//}
	//if !hasClient {
	//	return nil, errors.New(e.ClientNotFound)
	//}

	err = client.LockNamed("CheckoutWorkunits serving " + client_id)
	if err != nil {
		return
	}
	status := client.Status
	response_channel := client.coAckChannel

	work_length, _ := client.Current_work_length(false)
	client.Unlock()

	if work_length > 0 {
		logger.Error("Client %s wants to checkout work, but still has work: work_length=%d", client_id, work_length)
		return nil, errors.New(e.ClientBusy)
	}

	if status == CLIENT_STAT_SUSPEND {
		return nil, errors.New(e.ClientSuspended)
	}
	if status == CLIENT_STAT_DELETED {
		qm.RemoveClient(client_id, false)
		return nil, errors.New(e.ClientDeleted)
	}

	//req := CoReq{policy: req_policy, fromclient: client_id, available: available_bytes, count: num, response: client.coAckChannel}
	req := CoReq{policy: req_policy, fromclient: client_id, available: available_bytes, count: num, response: response_channel}

	logger.Debug(3, "(CheckoutWorkunits) %s qm.coReq <- req", client_id)
	// request workunit

	select {
	case qm.coReq <- req:
	default:
		logger.Error("Work request by client %s rejected, request queue is full", client_id)
		err = fmt.Errorf("Too many work requests - Please try again later")
		return

	}

	logger.Debug(3, "(CheckoutWorkunits) %s client.Get_Ack()", client_id)
	//ack := <-qm.coAck

	var ack CoAck
	// get workunit
	lock, err := client.RLockNamed("CheckoutWorkunits waiting for ack, client_id: " + client_id)
	if err != nil {
		return
	}
	ack, err = client.Get_Ack()
	client.RUnlockNamed(lock)

	logger.Debug(3, "(CheckoutWorkunits) %s got ack", client_id)
	if err != nil {
		return
	}

	logger.Debug(3, "(CheckoutWorkunits) %s got ack", client_id)

	if ack.err != nil {
		logger.Debug(3, "(CheckoutWorkunits) %s ack.err: %s", client_id, ack.err.Error())
		return ack.workunits, ack.err
	}

	added_work := 0
	for _, work := range ack.workunits {
		work_id := work.Id
		err = client.Add_work(work_id)
		if err != nil {
			return
		}
		added_work += 1
	}
	status, xerr := client.Get_Status(true)
	if xerr != nil {
		err = xerr
		return
	}
	if added_work > 0 && status == CLIENT_STAT_ACTIVE_IDLE {
		client.Set_Status(CLIENT_STAT_ACTIVE_BUSY, true)
	}

	logger.Debug(3, "(CheckoutWorkunits) %s finished", client_id)
	return ack.workunits, ack.err
}

func (qm *CQMgr) GetWorkById(id string) (workunit *Workunit, err error) {
	workunit, ok, err := qm.workQueue.Get(id)
	if err != nil {
		return
	}
	if !ok {
		err = errors.New(fmt.Sprintf("no workunit found with id %s", id))
	}
	return
}

func (qm *CQMgr) NotifyWorkStatus(notice Notice) {
	qm.feedback <- notice
	return
}

// when popWorks is called, the client should already be locked
func (qm *CQMgr) popWorks(req CoReq) (client_specific_workunits []*Workunit, err error) {

	client_id := req.fromclient

	client, ok, err := qm.clientMap.Get(client_id, true) // locks the clientmap
	if err != nil {
		return
	}
	if !ok {
		err = fmt.Errorf("(popWorks) Client %s not found", client_id)
		return
	}

	logger.Debug(3, "(popWorks) starting for client: %s", client_id)

	filtered, err := qm.filterWorkByClient(client)
	if err != nil {
		return
	}
	logger.Debug(3, "(popWorks) filterWorkByClient returned: %d (0 meansNoEligibleWorkunitFound)", len(filtered))
	if len(filtered) == 0 {
		err = errors.New(e.NoEligibleWorkunitFound)
		return
	}
	client_specific_workunits, err = qm.workQueue.selectWorkunits(filtered, req.policy, req.available, req.count)
	if err != nil {
		return
	}
	//get workunits successfully, put them into coWorkMap
	for _, work := range client_specific_workunits {
		work.Client = client_id
		work.CheckoutTime = time.Now()
		//qm.workQueue.Put(work) TODO isn't that already in the queue ?
		qm.workQueue.StatusChange(work.Id, work, WORK_STAT_CHECKOUT)
	}

	logger.Debug(3, "done with popWorks() for client: %s ", client_id)
	return
}

// client has to be read-locked
func (qm *CQMgr) filterWorkByClient(client *Client) (workunits WorkList, err error) {

	if client == nil {
		err = fmt.Errorf("(filterWorkByClient) client == nil")
		return
	}

	clientid := client.Id

	if clientid == "" {
		err = fmt.Errorf("(filterWorkByClient) clientid empty")
		return
	}

	logger.Debug(3, "(filterWorkByClient) starting for client: %s", clientid)

	workunit_list, err := qm.workQueue.Queue.GetWorkunits()
	if err != nil {
		return
	}

	if len(workunit_list) == 0 {
		err = errors.New(e.QueueEmpty)
		return
	}

	logger.Debug(3, "(filterWorkByClient) GetWorkunits() returned: %d", len(workunit_list))
	for _, workunit := range workunit_list {
		id := workunit.Id
		logger.Debug(3, "check if job %s would fit client %s", id, clientid)

		//skip works that are in the client's skip-list
		if client.Contains_Skip_work_nolock(workunit.Id) {
			logger.Debug(3, "2) workunit %s is in Skip_work list of the client %s)", id, clientid)
			continue
		}
		//skip works that have dedicate client groups which this client doesn't belong to
		if len(workunit.Info.ClientGroups) > 0 {
			eligible_groups := strings.Split(workunit.Info.ClientGroups, ",")
			if !contains(eligible_groups, client.Group) {
				logger.Debug(3, fmt.Sprintf("3) !contains(eligible_groups, client.Group) %s", id))
				continue
			}
		}
		//append works whos apps are supported by the client
		if contains(client.Apps, workunit.Cmd.Name) || contains(client.Apps, conf.ALL_APP) {
			logger.Debug(3, "append job %s to list of client %s", id, clientid)
			workunits = append(workunits, workunit)
		} else {
			logger.Debug(2, fmt.Sprintf("3) contains(client.Apps, work.Cmd.Name) || contains(client.Apps, conf.ALL_APP) %s", id))
		}
	}
	logger.Debug(3, fmt.Sprintf("done with filterWorkByClient() for client: %s", clientid))

	if len(workunits) == 0 {
		err = errors.New(e.NoEligibleWorkunitFound)
		return
	}

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
func (qm *CQMgr) handleWorkStatusChange(notice Notice) (err error) {
	//to be implemented for proxy or server
	return
}

func (qm *CQMgr) FetchDataToken(workid string, clientid string) (token string, err error) {
	//to be implemented for proxy or server
	return
}

func (qm *CQMgr) ShowWorkunits(status string) (workunits []*Workunit, err error) {
	workunit_list, err := qm.workQueue.GetAll()
	if err != nil {
		return
	}
	for _, work := range workunit_list {
		if work.State == status || status == "" {
			workunits = append(workunits, work)
		}
	}
	return
}

func (qm *CQMgr) ShowWorkunitsByUser(status string, u *user.User) (workunits []*Workunit) {
	// Only returns workunits of jobs that the user has read access to or is the owner of.  If user is admin, return all.
	workunit_list, err := qm.workQueue.GetAll()
	if err != nil {
		return
	}
	for _, work := range workunit_list {
		// skip loading jobs from db if user is admin
		if u.Admin == true {
			if work.State == status || status == "" {
				workunits = append(workunits, work)
			}
		} else {
			if jobid, err := GetJobIdByWorkId(work.Id); err == nil {
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
	}
	return workunits
}

func (qm *CQMgr) EnqueueWorkunit(work *Workunit) (err error) {
	err = qm.workQueue.Add(work)
	return
}

func (qm *CQMgr) ReQueueWorkunitByClient(client *Client, client_write_lock bool) (err error) {

	worklist, err := client.Get_current_work(client_write_lock)
	if err != nil {
		return
	}
	for _, workid := range worklist {
		logger.Debug(3, "(ReQueueWorkunitByClient) try to requeue work %s", workid)
		work, has_work, xerr := qm.workQueue.Get(workid)
		if xerr != nil {
			logger.Error("(ReQueueWorkunitByClient) error checking workunit %s", workid)
			continue
		}

		if !has_work {
			logger.Error("(ReQueueWorkunitByClient) Workunit %s not found", workid)
			continue
		}

		jobid, err := GetJobIdByWorkId(workid)
		if err != nil {
			logger.Error("(ReQueueWorkunitByClient) GetJobIdByWorkId: %s", err.Error())
			continue
		}
		job_state, err := dbGetJobFieldString(jobid, "state")
		if err != nil {
			logger.Error("(ReQueueWorkunitByClient) dbGetJobField: %s", err.Error())
			continue
		}

		if contains(JOB_STATS_ACTIVE, job_state) { //only requeue workunits belonging to active jobs (rule out suspended jobs)
			if work.Client == client.Id {
				qm.workQueue.StatusChange(workid, work, WORK_STAT_QUEUED)
				logger.Event(event.WORK_REQUEUE, "workid="+workid)
			} else {

				other_client_id := work.Client

				other_client, ok, xerr := qm.GetClient(other_client_id, true)
				if xerr != nil {
					logger.Error("(ReQueueWorkunitByClient) other_client: %s ", xerr)
					continue
				}
				if ok {
					// other_client exists (if otherclient does not exist, that is ok....)
					oc_has_work, err := other_client.Current_work_has(workid)
					if err != nil {
						logger.Error("(ReQueueWorkunitByClient) Current_work_has: %s ", err)
						continue
					}
					if !oc_has_work {
						// other client has not this workunit,
						qm.workQueue.StatusChange(workid, work, WORK_STAT_SUSPEND)
						continue
					}
				}
				// client does not exists of has different workunit
				// no status change

			}
		} else {
			qm.workQueue.StatusChange(workid, work, WORK_STAT_SUSPEND)
		}

	}
	return
}

//---end of workunit methods
