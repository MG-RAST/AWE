package core

// create workunit slice type to use for sorting

type WorkunitsSortby struct {
	Order     string
	Direction string
	Workunits []*Workunit
}

func (w WorkunitsSortby) Len() int {
	return len(w.Workunits)
}

func (w WorkunitsSortby) Swap(i, j int) {
	w.Workunits[i], w.Workunits[j] = w.Workunits[j], w.Workunits[i]
}

func (w WorkunitsSortby) Less(i, j int) bool {
	// default is ascending
	if w.Direction == "desc" {
		i, j = j, i
	}
	switch w.Order {
	// default is info.submittime
	default:
		return w.Workunits[i].Info.SubmitTime.Before(w.Workunits[j].Info.SubmitTime)
	case "wuid":
		return w.Workunits[i].Id < w.Workunits[j].Id
	case "client":
		return w.Workunits[i].Client < w.Workunits[j].Client
	case "info.submittime":
		return w.Workunits[i].Info.SubmitTime.Before(w.Workunits[j].Info.SubmitTime)
	case "checkout_time":
		return w.Workunits[i].CheckoutTime.Before(w.Workunits[j].CheckoutTime)
	case "info.name":
		return w.Workunits[i].Info.Name < w.Workunits[j].Info.Name
	case "cmd.name":
		return w.Workunits[i].Cmd.Name < w.Workunits[j].Cmd.Name
	case "rank":
		return w.Workunits[i].Rank < w.Workunits[j].Rank
	case "totalwork":
		return w.Workunits[i].TotalWork < w.Workunits[j].TotalWork
	case "state":
		return w.Workunits[i].State < w.Workunits[j].State
	case "failed":
		return w.Workunits[i].Failed < w.Workunits[j].Failed
	case "info.priority":
		return w.Workunits[i].Info.Priority < w.Workunits[j].Info.Priority
	}
}
