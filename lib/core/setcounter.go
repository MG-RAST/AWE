package core

import (
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
)

type SetCounter struct {
	Counter      []int
	Max          []int
	NumberOfSets int
	Scatter_type string
	//position_in_counter int
}

func NewSetCounter(numberOfSets int, array []cwl.Array, scatter_type string) (sc *SetCounter) {

	logger.Debug(3, "(NewSetCounter) numberOfSets: %d", numberOfSets)
	logger.Debug(3, "(NewSetCounter) array: %d", len(array))
	logger.Debug(3, "(NewSetCounter) scatter_type: %s", scatter_type)

	sc = &SetCounter{}

	sc.NumberOfSets = numberOfSets

	//sc.position_in_counter = sc.NumberOfSets

	sc.Counter = make([]int, sc.NumberOfSets)
	sc.Max = make([]int, sc.NumberOfSets)
	for i := 0; i < sc.NumberOfSets; i++ {
		sc.Counter[i] = 0
		sc.Max[i] = array[i].Len() - 1 // indicates last valid position. Needed for carry-over, e.g. 9+1 = 0
	}

	sc.Scatter_type = scatter_type
	return
}

func (sc *SetCounter) Increment() (ok bool) {

	if sc.Scatter_type == "cross" {
		//fmt.Printf("(SetCounter/Increment) cross\n")
		for position_in_counter := sc.NumberOfSets - 1; position_in_counter >= 0; position_in_counter-- {
			//fmt.Printf("(SetCounter/Increment) position_in_counter: %d\n", position_in_counter)
			//fmt.Printf("(SetCounter/Increment) sc.Counter[position_in_counter]: %d\n", sc.Counter[position_in_counter])
			if sc.Counter[position_in_counter] < sc.Max[position_in_counter] {
				sc.Counter[position_in_counter] += 1
				ok = true
				return
			}
			// sc.Counter[position_in_counter] == sc.Max[position_in_counter]

			sc.Counter[position_in_counter] = 0
			// carry over - continue

		}
	} else {
		//fmt.Printf("(SetCounter/Increment) dot\n")

		// "dot" dotproduct
		// this is not very efficient but keeps the code simpler, as only one counter is used

		if sc.Counter[0] >= sc.Max[0] {
			//end of counter
			ok = false
			return
		}

		// increment position in each array
		for position_in_counter := sc.NumberOfSets - 1; position_in_counter >= 0; position_in_counter-- {
			sc.Counter[position_in_counter] += 1
			//fmt.Printf("(SetCounter/Increment) sc.Counter[position_in_counter]: %d \n", sc.Counter[position_in_counter])
		}
		ok = true
		return
	}

	// carry over not possible, done.
	ok = false
	return

}
