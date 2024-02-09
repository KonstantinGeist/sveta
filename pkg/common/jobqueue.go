package common

import "sync"

type Job func() error

type JobQueue struct {
	jobsChannel chan Job
	stopChannel chan struct{}
	waitGroup   sync.WaitGroup
	logger      Logger
}

func NewJobQueue(logger Logger) *JobQueue {
	worker := &JobQueue{
		jobsChannel: make(chan Job, 128),
		stopChannel: make(chan struct{}),
		logger:      logger,
	}
	worker.waitGroup.Add(1)
	go worker.run()
	return worker
}

func (j *JobQueue) Enqueue(job Job) {
	j.jobsChannel <- job
}

func (j *JobQueue) Stop() {
	j.stopChannel <- struct{}{}
	j.waitGroup.Wait()
}

func (j *JobQueue) run() {
	for {
		select {
		case job := <-j.jobsChannel:
			err := job()
			if err != nil {
				j.logger.Log("failed to process a job: " + err.Error())
			}
		case <-j.stopChannel:
			j.waitGroup.Done()
			return
		}
	}
}
