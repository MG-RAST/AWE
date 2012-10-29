package core

import ()

func CreateJobUpload(params map[string]string, files FormFiles) (job *Job, err error) {
	job = NewJob()

	err = job.Mkdir()
	if err != nil {
		return
	}
	err = job.Update(params, files)
	if err != nil {
		return
	}
	err = job.Save()
	return
}
