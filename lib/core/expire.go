package core

import (
	"regexp"
	"time"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"gopkg.in/mgo.v2/bson"
)

var (
	// Ttl _
	Ttl *JobReaper
	// ExpireRegex _
	ExpireRegex = regexp.MustCompile(`^(\d+)(M|H|D)$`)
)

// InitReaper _
func InitReaper() {
	Ttl = NewJobReaper()
}

// JobReaper _
type JobReaper struct{}

// NewJobReaper _
func NewJobReaper() *JobReaper {
	return &JobReaper{}
}

// Handle _
func (jr *JobReaper) Handle() {
	waitDuration := time.Duration(conf.EXPIRE_WAIT) * time.Minute
	for {
		// sleep
		time.Sleep(waitDuration)
		// query to get expired jobs
		jobs := Jobs{}
		query := jr.getQuery()
		jobs.GetAllUnsorted(query)
		// delete expired jobs
		for _, j := range jobs {
			logger.Event(event.JOB_EXPIRED, "jobid="+j.ID)
			if err := j.Delete(); err != nil {
				logger.Error("Err@job_delete: " + err.Error())
			}
		}
	}
}

func (jr *JobReaper) getQuery() (query bson.M) {
	isCompOrDel := bson.M{"$or": []bson.M{bson.M{"state": JOB_STAT_COMPLETED}, bson.M{"state": JOB_STAT_DELETED}}} // job in completed or deleted state
	hasExpire := bson.M{"expiration": bson.M{"$exists": true}}                                                     // has the field
	toExpire := bson.M{"expiration": bson.M{"$ne": time.Time{}}}                                                   // value has been set, not default
	isExpired := bson.M{"expiration": bson.M{"$lt": time.Now()}}                                                   // value is too old
	query = bson.M{"$and": []bson.M{isCompOrDel, hasExpire, toExpire, isExpired}}
	return
}
