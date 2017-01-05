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
	//clientMapLock sync.RWMutex
	//clientMap     map[string]*Client
	clientMap    ClientMap
	workQueue    *WorkQueue
	suspendQueue bool
	coReq        chan CoReq //workunit checkout request (WorkController -> qmgr.Handler)
	//coAck        chan CoAck  //workunit checkout item including data and err (qmgr.Handler -> WorkController)
	feedback chan Notice //workunit execution feedback (WorkController -> qmgr.Handler)
	coSem    chan int    //semaphore for checkout (mutual exclusion between different clients)
}

func NewCQMgr() *CQMgr {
	return &CQMgr{
		//clientMap:    map[string]*Client{},
		clientMap:    ClientMap{_map: map[string]*Client{}},
		workQueue:    NewWorkQueue(),
		suspendQueue: false,
		coReq:        make(chan CoReq, 1024),
		//coAck:        make(chan CoAck),
		feedback: make(chan Notice),
		coSem:    make(chan int, 1), //non-blocking buffered channel
	}
}

//--------mgr methods-------

func (qm *CQMgr) ClientHandle() {
	// this code is not beeing used

}

// show functions used in debug
func (qm *CQMgr) ShowWorkQueue() {
	logger.Debug(1, fmt.Sprintf("current queuing workunits (%d)", qm.workQueue.Len()))
	for _, workunit := range qm.workQueue.GetAll() {
		id := workunit.Id
		logger.Debug(1, fmt.Sprintf("workid=%s", id))
	}
	return
}

//--------accessor methods-------

func (qm *CQMgr) GetClientMap() *ClientMap {
	return &qm.clientMap
}

func (qm *CQMgr) AddClient(client *Client, lock bool) {
	qm.clientMap.Add(client, lock)
}

func (qm *CQMgr) GetClient(id string, lock_clientmap bool) (client *Client, ok bool) {
	return qm.clientMap.Get(id, lock_clientmap)
}

// lock is for clientmap
func (qm *CQMgr) RemoveClient(id string, lock bool) {
	qm.clientMap.Delete(id, lock)
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
	if client, ok := qm.clientMap.Get(id, true); ok {
		client.Set_Status(new_status, client_write_lock)
		return
	}
	return errors.New(e.ClientNotFound)
}

func (qm *CQMgr) ClientStatusChange(client *Client, new_status string, client_write_lock bool) (err error) {
	client.Set_Status(new_status, client_write_lock)
	return

}

func (qm *CQMgr) HasClient(id string, lock_clientmap bool) (has bool) {
	if _, ok := qm.clientMap.Get(id, lock_clientmap); ok {
		has = true
	} else {
		has = false
	}
	return
}

func (qm *CQMgr) ListClients() (ids []string) {
	//qm.clientMap.RLock()
	//defer qm.clientMap.RUnlock()
	//for id, _ := range qm.clientMap {
	//	ids = append(ids, id)
	//}
	return qm.clientMap.GetClientIds()
}

//--------client methods-------

func (qm *CQMgr) CheckClient(client *Client) (ok bool) {
	ok = true
	client.LockNamed("ClientChecker")
	defer client.Unlock()

	if client.Tag == true {
		client.Tag = false
		total_minutes := int(time.Now().Sub(client.RegTime).Minutes())
		hours := total_minutes / 60
		minutes := total_minutes % 60
		client.Serve_time = fmt.Sprintf("%dh%dm", hours, minutes)
		if client.Current_work_length(false) > 0 {
			client.Idle_time = 0
		} else {
			client.Idle_time += 30
		}

	} else {
		if !qm.HasClient(client.Id, false) { // no need to lock clientmap again
			return
		}
		//now client must be gone as tag set to false 30 seconds ago and no heartbeat received thereafter
		logger.Event(event.CLIENT_UNREGISTER, "clientid="+client.Id+";name="+client.Name)
		//requeue unfinished workunits associated with the failed client
		qm.ReQueueWorkunitByClient(client, false)
		//delete the client from client map
		//qm.RemoveClient(client.Id)
		ok = false
	}
	return
}

func (qm *CQMgr) ClientChecker() {
	for {
		time.Sleep(30 * time.Second)
		logger.Debug(3, "time to update client list....")

		delete_clients := []string{}

		client_list := qm.clientMap.GetClients() // this uses a list of pointers to prevent long locking of the CLientMap

		for _, client := range client_list {
			ok := qm.CheckClient(client)
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
	if client, ok := qm.GetClient(id, true); ok {
		client.LockNamed("ClientHeartBeat")
		defer client.Unlock()

		// If the name of the clientgroup (from auth token) does not match the name in the client retrieved, throw an error
		if cg != nil && client.Group != cg.Name {
			return nil, errors.New(e.ClientGroupBadName)
		}
		client.Tag = true

		logger.Debug(3, "HeartBeatFrom:"+"clientid="+id+",name="+client.Name)

		//get suspended workunit that need the client to discard
		current_work := client.Get_current_work(false)
		suspended := []string{}

		for _, work_id := range current_work {
			work, ok := qm.workQueue.workMap.Get(work_id)
			if ok {
				if work.State == WORK_STAT_SUSPEND {
					suspended = append(suspended, work.Id)
				}
			}
		}
		if len(suspended) > 0 {
			hbmsg["discard"] = strings.Join(suspended, ",")
		}
		if client.Status == CLIENT_STAT_DELETED {
			hbmsg["stop"] = id
		}

		//hbmsg["discard"] = strings.Join(workids, ",")
		return hbmsg, nil
	}
	return hbmsg, errors.New(e.ClientNotFound)
}

func (qm *CQMgr) RegisterNewClient(files FormFiles, cg *ClientGroup) (client *Client, err error) {
	if _, ok := files["profile"]; ok {
		client, err = NewProfileClient(files["profile"].Path)
		os.Remove(files["profile"].Path)
		if err != nil {
			err = fmt.Errorf("NewProfileClient returned: %s", err.Error())
			return
		}
	} else {
		client = NewClient()
	}

	client.LockNamed("RegisterNewClient")
	defer client.Unlock()

	// If clientgroup is nil at this point, create a publicly owned clientgroup, with the provided group name (if one doesn't exist with the same name)
	if cg == nil {
		// See if clientgroup already exists with this name
		// If it does and it does not have "public" execution rights, throw error
		// If it doesn't, create one owned by public, and continue with client registration
		// Otherwise proceed with client registration.
		cg, _ = LoadClientGroupByName(client.Group)

		if cg != nil {
			rights := cg.Acl.Check("public")
			if rights["execute"] == false {
				err = errors.New("Clientgroup with the group specified by your client exists, but you cannot register with it, without clientgroup token.")
				return nil, err
			}
		} else {
			u := &user.User{Uuid: "public"}
			cg, err = CreateClientGroup(client.Group, u)
			if err != nil {
				err = fmt.Errorf("CreateClientGroup returned: %s", err.Error())
				return nil, err
			}
		}
	}
	// If the name of the clientgroup (from auth token) does not match the name in the client profile, throw an error
	if cg != nil && client.Group != cg.Name {
		return nil, errors.New(e.ClientGroupBadName)
	}

	qm.AddClient(client, true) // locks clientMap

	if client.Current_work_length(false) > 0 { //re-registered client
		// move already checked-out workunit from waiting queue (workMap) to checked-out list (coWorkMap)

		for workid, _ := range client.Current_work {
			if qm.workQueue.Has(workid) {
				qm.workQueue.StatusChange(workid, WORK_STAT_CHECKOUT)
			}
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

	if client, ok := qm.GetClient(id, true); ok {
		if val, exists := filtered_clientgroups[client.Group]; exists == true || val == true {
			return client, nil
		}
	}
	return nil, errors.New(e.ClientNotFound)
}

func (qm *CQMgr) GetAllClientsByUser(u *user.User) (clients []*Client) {
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

	for _, client := range qm.clientMap.GetClients() {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
			clients = append(clients, client)
		}
	}

	return clients
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

	if client, ok := qm.GetClient(id, true); ok {
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
		client, ok = qm.GetClient(id, true)
		if !ok {
			return errors.New(e.ClientNotFound)
		}
	}

	if client_write_lock {
		client.LockNamed("SuspendClient")
		defer client.Unlock()
	}

	status := client.Get_Status(false)
	if status == CLIENT_STAT_ACTIVE_IDLE || status == CLIENT_STAT_ACTIVE_BUSY {
		client.Set_Status(CLIENT_STAT_SUSPEND, false)
		//if err = qm.ClientStatusChange(id, CLIENT_STAT_SUSPEND); err != nil {
		//	return
		//}
		qm.ReQueueWorkunitByClient(client, client_write_lock)
		return
	}
	return errors.New(e.ClientNotActive)

}

func (qm *CQMgr) SuspendAllClients() (count int) {
	for _, id := range qm.ListClients() {
		if err := qm.SuspendClient(id, nil, true); err == nil {
			count += 1
		}
	}
	return count
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

	if client, ok := qm.GetClient(id, true); ok {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
			client.LockNamed("SuspendClientByUser")
			defer client.Unlock()
			status := client.Get_Status(false)
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
	return errors.New(e.ClientNotFound)
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

	for _, client := range qm.clientMap.GetClients() {
		client.LockNamed("SuspendAllClientsByUser")
		status := client.Get_Status(false)
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true && (status == CLIENT_STAT_ACTIVE_IDLE || status == CLIENT_STAT_ACTIVE_BUSY) {
			qm.SuspendClient("", client, false)
			count += 1
		}
		client.Unlock()
	}

	return count
}

func (qm *CQMgr) ResumeClient(id string) (err error) {
	if client, ok := qm.GetClient(id, true); ok {
		client.LockNamed("ResumeClient")
		defer client.Unlock()
		if client.Status == CLIENT_STAT_SUSPEND {
			//err = qm.ClientStatusChange(id, CLIENT_STAT_ACTIVE_IDLE)
			client.Status = CLIENT_STAT_ACTIVE_IDLE
			return
		}
		return errors.New(e.ClientNotSuspended)
	}
	return errors.New(e.ClientNotFound)
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

	if client, ok := qm.GetClient(id, true); ok {
		client.LockNamed("ResumeClientByUser")
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
	return errors.New(e.ClientNotFound)
}

func (qm *CQMgr) ResumeSuspendedClients() (count int) {

	for _, client := range qm.clientMap.GetClients() {
		client.LockNamed("ResumeSuspendedClients")
		if client.Status == CLIENT_STAT_SUSPEND {
			//qm.ClientStatusChange(client.Id, CLIENT_STAT_ACTIVE_IDLE)
			client.Status = CLIENT_STAT_ACTIVE_IDLE
			count += 1
		}
		client.Unlock()
	}

	return count
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

	for _, client := range qm.clientMap.GetClients() {
		client.LockNamed("ResumeSuspendedClientsByUser")
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true && client.Status == CLIENT_STAT_SUSPEND {
			//qm.ClientStatusChange(client.Id, CLIENT_STAT_ACTIVE_IDLE)
			client.Status = CLIENT_STAT_ACTIVE_IDLE
			count += 1
		}
		client.Unlock()
	}

	return count
}

func (qm *CQMgr) UpdateSubClients(id string, count int) {
	if client, ok := qm.GetClient(id, true); ok {
		client.SubClients = count

	}
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

	if client, ok := qm.GetClient(id, true); ok {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
			client.SubClients = count

		}
	}
}

//--------end client methods-------

//-------start of workunit methods---

func (qm *CQMgr) CheckoutWorkunits(req_policy string, client_id string, available_bytes int64, num int) (workunits []*Workunit, err error) {

	logger.Debug(3, "run CheckoutWorkunits for client %s", client_id)

	//precheck if the client is registered
	client, hasClient := qm.GetClient(client_id, true)
	if !hasClient {
		return nil, errors.New(e.ClientNotFound)
	}

	client.LockNamed("CheckoutWorkunits serving " + client_id)
	defer client.Unlock()

	status := client.Status
	response_channel := client.coAckChannel

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
	qm.coReq <- req
	logger.Debug(3, "(CheckoutWorkunits) %s client.Get_Ack()", client_id)
	//ack := <-qm.coAck

	var ack CoAck
	// get workunit
	ack, err = client.Get_Ack()

	logger.Debug(3, "(CheckoutWorkunits) %s got ack", client_id)
	if err != nil {
		return
	}

	logger.Debug(3, "(CheckoutWorkunits) %s got ack", client_id)
	if ack.err == nil {
		for _, work := range ack.workunits {
			work_id := work.Id
			client.Add_work_nolock(work_id)
		}
		if client.Get_Status(false) == CLIENT_STAT_ACTIVE_IDLE {
			client.Set_Status(CLIENT_STAT_ACTIVE_BUSY, false)
		}
	} else {

		logger.Debug(3, "(CheckoutWorkunits) %s ack.err: %s", client_id, ack.err.Error())
	}

	logger.Debug(3, "(CheckoutWorkunits) %s finished", client_id)
	return ack.workunits, ack.err
}

//func (qm *CQMgr) LockSemaphore() {
//	qm.coSem <- 1
//}

//func (qm *CQMgr) UnlockSemaphore() {
//	<-qm.coSem
//}

func (qm *CQMgr) GetWorkById(id string) (workunit *Workunit, err error) {
	workunit, ok := qm.workQueue.Get(id)
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
func (qm *CQMgr) popWorks(req CoReq) (works []*Workunit, err error) {

	client_id := req.fromclient

	client, ok := qm.clientMap.Get(client_id, true) // locks the clientmap

	if !ok {
		err = fmt.Errorf("Client %s not found", client_id)
		return
	}

	logger.Debug(3, fmt.Sprintf("starting popWorks() for client: %s", client_id))

	filtered, err := qm.filterWorkByClient(client)
	if err != nil {
		return
	}
	logger.Debug(2, fmt.Sprintf("popWorks filtered: %d (0 meansNoEligibleWorkunitFound)", len(filtered)))
	if len(filtered) == 0 {
		return nil, errors.New(e.NoEligibleWorkunitFound)
	}
	works, err = qm.workQueue.selectWorkunits(filtered, req.policy, req.available, req.count)
	if err == nil { //get workunits successfully, put them into coWorkMap
		for _, work := range works {
			work.Client = client_id
			work.CheckoutTime = time.Now()
			qm.workQueue.Put(work)
			qm.workQueue.StatusChange(work.Id, WORK_STAT_CHECKOUT)
		}
	}
	logger.Debug(3, fmt.Sprintf("done with popWorks() for client: %s", client_id))
	return
}

// client has to be read-locked
func (qm *CQMgr) filterWorkByClient(client *Client) (workunits WorkList, err error) {

	clientid := client.Id
	logger.Debug(3, fmt.Sprintf("starting filterWorkByClient() for client: %s", clientid))

	for _, workunit := range qm.workQueue.Wait.GetWorkunits() {
		id := workunit.Id
		logger.Debug(3, "check if job %s would fit client %s", id, clientid)

		//skip works that are in the client's skip-list
		if client.Contains_Skip_work_nolock(workunit.Id) {
			logger.Debug(3, "2) workunit %s is in Skip_work list of the client %s) %s", id, clientid)
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

func (qm *CQMgr) ShowWorkunits(status string) (workunits []*Workunit) {
	for _, work := range qm.workQueue.GetAll() {
		if work.State == status || status == "" {
			workunits = append(workunits, work)
		}
	}
	return workunits
}

func (qm *CQMgr) ShowWorkunitsByUser(status string, u *user.User) (workunits []*Workunit) {
	// Only returns workunits of jobs that the user has read access to or is the owner of.  If user is admin, return all.
	for _, work := range qm.workQueue.GetAll() {
		// skip loading jobs from db if user is admin
		if u.Admin == true {
			if work.State == status || status == "" {
				workunits = append(workunits, work)
			}
		} else {
			if jobid, err := GetJobIdByWorkId(work.Id); err == nil {
				if job, err := LoadJob(jobid); err == nil {
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

func (qm *CQMgr) ReQueueWorkunitByClient(client *Client, lock bool) (err error) {

	worklist := client.Get_current_work(lock)
	for _, workid := range worklist {
		if qm.workQueue.Has(workid) {
			jobid, _ := GetJobIdByWorkId(workid)
			if job, err := LoadJob(jobid); err == nil {
				if contains(JOB_STATS_ACTIVE, job.State) { //only requeue workunits belonging to active jobs (rule out suspended jobs)
					qm.workQueue.StatusChange(workid, WORK_STAT_QUEUED)
					logger.Event(event.WORK_REQUEUE, "workid="+workid)
				}
			}
		}
	}
	return
}

//---end of workunit methods
