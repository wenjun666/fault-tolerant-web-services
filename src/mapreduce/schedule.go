package mapreduce

import (
	"fmt"
	"sync"
)

// schedule starts and waits for all tasks in the given phase (Map or Reduce).
func (mr *Master) schedule(phase jobPhase) {
	var ntasks int
	var nios int // number of inputs (for reduce) or outputs (for map)
	switch phase {
	case mapPhase:
		ntasks = len(mr.files)
		nios = mr.nReduce
	case reducePhase:
		ntasks = mr.nReduce
		nios = len(mr.files)
	}

	fmt.Printf("Schedule: %v %v tasks (%d I/Os)\n", ntasks, phase, nios)

	// All ntasks tasks have to be scheduled on workers, and only once all of
	// them have been completed successfully should the function return.
	// Remember that workers may fail, and that any given worker may finish
	// multiple tasks.
	//
	// TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO

	var wg sync.WaitGroup

	// concurrently doing n tasks
	for i:= 0 ; i < ntasks; i++ {

		wg.Add(1)

		go func(i int) {

			defer wg.Done()

			//create the particular task for a  worker
			args := &DoTaskArgs{
				JobName: mr.jobName,
				File: mr.files[i],
				Phase: phase,
				TaskNumber: i,
				NumOtherPhase: nios,
			}

			ok := false
			// continuing doing i-th task until it is completed
			for !ok {

				// get idle worker from registerChannel
				worker := <- mr.registerChannel

				// get the status of i-th task
				ok = call(worker, "Worker.DoTask", args, new(struct{}))

				// set this worker as idle after finishing the task
				// need to create go routine to do this
				// if the worker fail, I should disgard the worker, because a fail worker will keep failing, which will cost lots of time
				go func(){
					mr.registerChannel <- worker
				}()
			}

		}(i)
	}

	// Wait for all goroutines to complete.
	wg.Wait()


	fmt.Printf("Schedule: %v phase done\n", phase)
}
