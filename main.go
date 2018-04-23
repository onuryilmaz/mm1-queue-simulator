package main

import (
	"log"
	"github.com/agoussia/godes"
	"runtime"
	"flag"
)

type Job struct {
	ID                  int
	creationTime        int
	processStartTime    int
	processCompleteTime int

	processDuration int

	finished bool
}

func NewJob(time int) *Job {
	log.Printf("Time: %d | Job created with ID %d", time, time)
	j := &Job{creationTime: time, ID: time}
	return j
}

func (j *Job) startJob(time int, processDuration int) {

	j.processStartTime = time
	j.processDuration = processDuration

	log.Printf("Time: %d | Job (ID: %d) is exiting queue and starting processing | Process Time: %d", time, j.ID, j.processDuration)

}

func (j *Job) isDone(time int) bool {
	if (j.processStartTime +j.processDuration) == time {
		j.processCompleteTime = time
		j.finished = true
		log.Printf("Time: %d | Job (ID: %d) completed processing", time, j.ID)
		return true
	}
	return false
}

type MM1 struct {
	queue     []*Job
	server    bool
	inProcess *Job

	meanArrivalTime int
	meanServiceTime int
	maximumTime     int

	exponentialGenerator *godes.ExpDistr

	Statistics
}

type Statistics struct {
	JobsGenerated int
	JobsServiced  int

	QueueLength      int
	QueueWaitingTime int

	TasksInSystem       int
	ReponseTimeInSystem int

	ServerUtilization int

	ArrivalTime int
	ServiceTime int

	Time int
}

func (s *MM1) Summary() {

	mu := float64(1) / float64(s.meanServiceTime)
	lambda := float64(1) / float64(s.meanArrivalTime)
	ro := lambda / mu

	averageNumberOfJobsInQueue := (ro * ro) / (1 - ro)
	averageNumberOfJobsInSystem := lambda / (mu - lambda)

	averageResponseTime := 1.0 / (mu * (1 - ro))
	averageWaitingTimeInQueue := ro / (mu * (1.0 - ro))

	log.Print("Total number of jobs generated: ", s.JobsGenerated, " | Theoretical: ", s.Time/s.meanArrivalTime)
	log.Print("Total number of jobs serviced: ", s.JobsServiced, " | Theoretical: ", s.Time/s.meanServiceTime)
	log.Print("Average queue length: ", s.QueueLength/s.Time, " | Theoretical: ", averageNumberOfJobsInQueue)
	log.Print("Average waiting time in queue: ", s.QueueWaitingTime/(s.JobsServiced), " | Theoretical: ", averageWaitingTimeInQueue)
	log.Print("Average number of tasks in the system: ", s.TasksInSystem/s.Time, " | Theoretical: ", averageNumberOfJobsInSystem)
	log.Print("Response time (Average waiting time in the system): ", s.ReponseTimeInSystem/s.JobsGenerated, " | Theoretical: ", averageResponseTime)
	log.Printf("Server utilization: %f | Theoretical: %f", float64(s.ServerUtilization)/float64(s.Time), ro)
	log.Print("Idle time for the server: ", s.Time-s.ServerUtilization)

	log.Print("Average arrival time: ", float64(s.ArrivalTime)/float64(s.JobsGenerated))
	log.Print("Average service time: ", float64(s.ServiceTime)/float64(s.JobsServiced))
}

func NewMM1(meanArrivalTime int, meanServiceTime int, maximumTime int) *MM1 {
	m := &MM1{}
	m.meanArrivalTime = meanArrivalTime
	m.meanServiceTime = meanServiceTime
	m.maximumTime = maximumTime
	m.queue = make([]*Job, 0)
	m.exponentialGenerator = godes.NewExpDistr(false)

	return m
}

func (mm1 *MM1) Enqueue(j *Job, time int) {
	log.Printf("Time: %d | Job (ID: %d) entering queue", time, j.ID)
	mm1.queue = append(mm1.queue, j)
}

func (mm1 *MM1) Dequeue(time int) (*Job, bool) {

	if len(mm1.queue) == 0 {
		log.Printf("Time: %d | Queue is empty", time)
		return nil, false
	}

	j := mm1.queue[0]
	mm1.queue = mm1.queue[1:]

	return j, true
}

func (mm1 *MM1) GenerateServiceTime() int {
	lambda := float64(1.0) / float64(mm1.meanServiceTime)
	log.Println("Generating service time with lambda :", lambda)
	data := mm1.exponentialGenerator.Get(lambda)
	integerData := int(data + 0.5)
	log.Println("Generated service time:", integerData)
	return integerData
}

func (mm1 *MM1) GenerateArrivalTime() int {
	lambda := float64(1.0) / float64(mm1.meanArrivalTime)
	log.Println("Generating arrival time with lambda :", lambda)
	data := mm1.exponentialGenerator.Get(lambda)
	integerData := int(data + 0.5)
	if integerData == 0 {
		integerData = 1
	}
	log.Println("Generated arrival time:", integerData)
	return integerData
}

func (mm1 *MM1) Start() {

	nextArrivalTime := mm1.GenerateArrivalTime()
	mm1.ArrivalTime = nextArrivalTime

	// 	timer = 0
	// while ( timer < max_time )
	for time := 0; time <= mm1.maximumTime; time++ {

		// if it is the time for a new arrival
		if time == nextArrivalTime {
			log.Printf("Time: %d | New arrival time!", time)
			// generate a new task and insert it to the queue
			j := NewJob(time)
			mm1.JobsGenerated++

			mm1.Enqueue(j, time)

			// generate random IAT for the next task
			randomIAT :=  mm1.GenerateArrivalTime()
			nextArrivalTime += randomIAT
			mm1.ArrivalTime += randomIAT

		}

		// if the server is idle
		if mm1.server == false {

			log.Printf("Time: %d | Server is idle!", time)

			// delete a task from the beginning of the queue and assign it to the server
			found := false
			mm1.inProcess, found = mm1.Dequeue(time)

			if found {
				mm1.server = true
				// generate random service time
				mm1.inProcess.startJob(time, mm1.GenerateServiceTime())
				mm1.QueueWaitingTime += mm1.inProcess.processStartTime - mm1.inProcess.creationTime
			}

		} else {
			mm1.ServerUtilization++
		}

		// if the server has finished serving the task
		if mm1.inProcess != nil && mm1.inProcess.isDone(time) {
			log.Printf("Time: %d | Server is done!", time)
			mm1.JobsServiced++
			mm1.ServiceTime +=  mm1.inProcess.processCompleteTime - mm1.inProcess.processStartTime
			mm1.ReponseTimeInSystem += mm1.inProcess.processCompleteTime - mm1.inProcess.creationTime

			// mark the task as finished
			mm1.inProcess = nil
			mm1.server = false

			// delete a task from the beginning of the queue and assign it to the server
			found := false
			mm1.inProcess, found = mm1.Dequeue(time)

			if found {
				mm1.server = true
				// generate random service time
				mm1.inProcess.startJob(time, mm1.GenerateServiceTime())
				mm1.QueueWaitingTime += mm1.inProcess.processStartTime - mm1.inProcess.creationTime
			}

		}

		// update statistical information
		log.Printf("Time: %d | Number in queue: %d | Service busy: %v", time, len(mm1.queue), mm1.server)
		mm1.QueueLength += len(mm1.queue)
		mm1.TasksInSystem += len(mm1.queue)
		if mm1.server {
			mm1.TasksInSystem++
		}
		mm1.Time++
	}

}
func main() {

	meanArrivalTime := flag.Int("mean-arrival-time", 0, "Mean arrival time")
	meanServiceTime := flag.Int("mean-service-time", 0, "Mean service time")
	maxTime := flag.Int("max-time", 0, "Max time")
	flag.Parse()

	runtime.GOMAXPROCS(1)
	mm1 := NewMM1(*meanArrivalTime, *meanServiceTime, *maxTime)

	mm1.Start()

	mm1.Summary()
}
