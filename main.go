package main

import (
	"flag"
	"log"
	"runtime"

	"github.com/agoussia/godes"
)

// Job holds the properties of a job instance
type Job struct {
	ID                  int
	creationTime        int
	processStartTime    int
	processCompleteTime int
	processDuration     int
}

// NewJob creates a new job at the provided time
func NewJob(time int) *Job {
	log.Printf("Time: %d | Job created with ID %d", time, time)
	j := &Job{creationTime: time, ID: time}
	return j
}

// StartJob starts the job at the provided time with the duration
func (j *Job) StartJob(time int, processDuration int) {

	j.processStartTime = time
	j.processDuration = processDuration

	log.Printf("Time: %d | Job (ID: %d) is exiting queue and starting processing | Process Time: %d", time, j.ID, j.processDuration)
}

// IsDone checks whether the job is completed at the provided time
func (j *Job) IsDone(time int) bool {
	if (j.processStartTime + j.processDuration) == time {
		j.processCompleteTime = time
		log.Printf("Time: %d | Job (ID: %d) completed processing", time, j.ID)
		return true
	}
	return false
}

// MM1 holds the properties for MM1 simulator
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

// Statistics holds the properties of collected statistics
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

// Summary calculates the theoretical and simulation values and prints
func (s *MM1) Summary() {

	mu := float64(1) / float64(s.meanServiceTime)
	lambda := float64(1) / float64(s.meanArrivalTime)
	ro := lambda / mu

	averageNumberOfJobsInQueue := (ro * ro) / (1 - ro)
	averageNumberOfJobsInSystem := lambda / (mu - lambda)

	averageResponseTime := 1.0 / (mu * (1 - ro))
	averageWaitingTimeInQueue := ro / (mu * (1.0 - ro))

	log.Println("SUMMARY------------")
	log.Printf("Total number of jobs generated: %d | Theoretical Maximum: %.2f", s.JobsGenerated, float64(s.Time)/float64(s.meanArrivalTime))
	log.Printf("Total number of jobs serviced: %d | Theoretical Maximum: %.2f", s.JobsServiced, float64(s.Time)/float64(s.meanServiceTime))
	log.Printf("Average queue length: %.2f | Theoretical: %.2f", float64(s.QueueLength)/float64(s.Time), averageNumberOfJobsInQueue)
	log.Printf("Average waiting time in queue: %.2f | Theoretical: %.2f", float64(s.QueueWaitingTime)/float64(s.JobsServiced), averageWaitingTimeInQueue)
	log.Printf("Average number of tasks in the system: %.2f | Theoretical: %.2f", float64(s.TasksInSystem)/float64(s.Time), averageNumberOfJobsInSystem)
	log.Printf("Response time (Average waiting time in the system): %.2f | Theoretical: %.2f", float64(s.ReponseTimeInSystem)/float64(s.JobsGenerated), averageResponseTime)
	log.Printf("Server utilization: %.2f | Theoretical: %.2f", float64(s.ServerUtilization)/float64(s.Time), ro)
	log.Printf("Idle time for the server: %d | Theoretical: %.2f", s.Time-s.ServerUtilization, float64(s.Time)*(1-ro))

	log.Printf("Average arrival time: %2.f | Expected: %d", float64(s.ArrivalTime)/float64(s.JobsGenerated), s.meanArrivalTime)
	log.Printf("Average service time: %2.f | Expected: %d", float64(s.ServiceTime)/float64(s.JobsServiced), s.meanServiceTime)
}

// NewMM1 creates a new simulator instance with the provided parameters
func NewMM1(meanArrivalTime int, meanServiceTime int, maximumTime int) *MM1 {
	m := &MM1{}
	m.meanArrivalTime = meanArrivalTime
	m.meanServiceTime = meanServiceTime
	m.maximumTime = maximumTime
	m.queue = make([]*Job, 0)
	m.exponentialGenerator = godes.NewExpDistr(false)

	return m
}

// Enqueue adds the provided job to the queue at the provided time
func (mm1 *MM1) Enqueue(j *Job, time int) {
	log.Printf("Time: %d | Job (ID: %d) entering queue", time, j.ID)
	mm1.queue = append(mm1.queue, j)
}

// Dequeue removed a job from the queue at the provided time
func (mm1 *MM1) Dequeue(time int) (*Job, bool) {

	if len(mm1.queue) == 0 {
		log.Printf("Time: %d | Queue is empty", time)
		return nil, false
	}

	j := mm1.queue[0]
	mm1.queue = mm1.queue[1:]

	return j, true
}

// GenerateServiceTime generates a random service time
func (mm1 *MM1) GenerateServiceTime() int {
	lambda := float64(1.0) / float64(mm1.meanServiceTime)
	data := mm1.exponentialGenerator.Get(lambda)
	integerData := int(data + 0.5)
	if integerData == 0 {
		integerData = 1
	}
	return integerData
}

// GenerateArrivalTime generates a random arrival time
func (mm1 *MM1) GenerateArrivalTime() int {
	lambda := float64(1.0) / float64(mm1.meanArrivalTime)
	data := mm1.exponentialGenerator.Get(lambda)
	integerData := int(data + 0.5)
	if integerData == 0 {
		integerData = 1
	}
	return integerData
}

// Start starts the simulator
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
			randomIAT := mm1.GenerateArrivalTime()
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
				mm1.inProcess.StartJob(time, mm1.GenerateServiceTime())
				mm1.QueueWaitingTime += mm1.inProcess.processStartTime - mm1.inProcess.creationTime
			}

		} else {
			mm1.ServerUtilization++
		}

		// if the server has finished serving the task
		if mm1.inProcess != nil && mm1.inProcess.IsDone(time) {
			log.Printf("Time: %d | Server is done!", time)
			mm1.JobsServiced++
			mm1.ServiceTime += mm1.inProcess.processCompleteTime - mm1.inProcess.processStartTime
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
				mm1.inProcess.StartJob(time, mm1.GenerateServiceTime())
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

	meanArrivalTime := flag.Int("mean-arrival-time", 100, "Mean arrival time")
	meanServiceTime := flag.Int("mean-service-time", 40, "Mean service time")
	maxTime := flag.Int("max-time", 10000, "Max time")
	flag.Parse()

	runtime.GOMAXPROCS(1)
	mm1 := NewMM1(*meanArrivalTime, *meanServiceTime, *maxTime)

	mm1.Start()

	mm1.Summary()
}
