package core

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
)

type WorkLog struct {
	Id   string            `bson:"wuid" json:"wuid"` // TODO change !
	Rank int               `bson:"rank" json:"rank"`
	Logs map[string]string `bson:"logs" json:"logs"`
}

func NewWorkLog(id Workunit_Unique_Identifier) (wlog *WorkLog) {
	work_id := fmt.Sprintf("%s_%d", id.TaskId, id.Rank)
	wlog = new(WorkLog)
	wlog.Id = work_id
	wlog.Rank = id.Rank
	wlog.Logs = map[string]string{}
	for _, log := range conf.WORKUNIT_LOGS {

		wlog.Logs[log], _ = QMgr.GetReportMsg(id, log)
	}
	return
}
