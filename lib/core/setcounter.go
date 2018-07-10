package core

import "github.com/MG-RAST/AWE/lib/core/cwl"

type SetCounter struct {
	Counter      []int
	Max          []int
	NumberOfSets int
	Scatter_type string
	//position_in_counter int
}

func NewSetCounter(numberOfSets int, array []*cwl.Array, scatter_type string) (sc *SetCounter) {

	sc = &SetCounter{}

	sc.NumberOfSets = numberOfSets

	//sc.position_in_counter = sc.NumberOfSets

	sc.Counter = make([]int, sc.NumberOfSets)
	sc.Max = make([]int, sc.NumberOfSets)
	for i := 0; i < sc.NumberOfSets; i++ {
		sc.Counter[i] = 0
		sc.Max[i] = array[i].Len() - 1
	}

	sc.Scatter_type = scatter_type
	return
}

func (sc *SetCounter) Increment() (ok bool) {

	if sc.Scatter_type == "cross" {
		for position_in_counter := sc.NumberOfSets - 1; position_in_counter >= 0; position_in_counter-- {
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
		// "dot" dotproduct
		// this is not very efficient but keeps the code simpler, as only one counter is used
		if sc.Counter[0] >= sc.Max[0] {
			ok = false
			return
		}

		for position_in_counter := sc.NumberOfSets - 1; position_in_counter >= 0; position_in_counter-- {
			sc.Counter[position_in_counter] += 1
		}
	}

	// carry over not possible, done.
	ok = false
	return

}
