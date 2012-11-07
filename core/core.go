package core

import ()

func CreateJobUpload(params map[string]string, files FormFiles) (job *Job, err error) {
	job = NewJob()

	err = job.Mkdir()
	if err != nil {
		return
	}
	err = job.UpdateFile(params, files)
	if err != nil {
		return
	}
	err = job.ParseTasks()
	if err != nil {
		return
	}

	err = job.Save()
	return
}

func LoadJob(id string) (job *Job, err error) {
	if db, err := DBConnect(); err == nil {
		defer db.Close()
		job = new(Job)
		if err = db.FindById(id, job); err == nil {
			return job, nil
		} else {
			return nil, err
		}
	}
	return nil, err
}

func LoadJobs(ids []string) (jobs []*Job, err error) {
	if db, err := DBConnect(); err == nil {
		defer db.Close()
		if err = db.FindJobs(ids, &jobs); err == nil {
			return jobs, err
		} else {
			return nil, err
		}
	}
	return nil, err
}
