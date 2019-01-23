package core

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

type MultipartWriter struct {
	b bytes.Buffer
	w *multipart.Writer
}

func NewMultipartWriter() *MultipartWriter {
	m := &MultipartWriter{}
	m.w = multipart.NewWriter(&m.b)
	return m
}

func (m *MultipartWriter) Send(method string, url string, header map[string][]string) (response *http.Response, err error) {
	m.w.Close()
	//fmt.Println("------------")
	//spew.Dump(m.w)
	//fmt.Println("------------")

	req, err := http.NewRequest(method, url, &m.b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", m.w.FormDataContentType())

	for key := range header {
		header_array := header[key]
		for _, value := range header_array {
			req.Header.Add(key, value)
		}

	}

	// Submit the request
	client := &http.Client{}
	//fmt.Printf("%s %s\n\n", method, url)
	response, err = client.Do(req)
	if err != nil {
		return
	}

	// Check the response
	//if response.StatusCode != http.StatusOK {
	//	err = fmt.Errorf("bad status: %s", response.Status)
	//}
	return

}

func (m *MultipartWriter) AddDataAsFile(fieldname string, filepath string, data *[]byte) (err error) {

	fw, err := m.w.CreateFormFile(fieldname, filepath)
	if err != nil {
		return
	}
	_, err = fw.Write(*data)
	if err != nil {
		return
	}
	return
}

func (m *MultipartWriter) AddFile(fieldname string, filepath string) (err error) {

	f, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer f.Close()
	fw, err := m.w.CreateFormFile(fieldname, filepath)
	if err != nil {
		return
	}
	if _, err = io.Copy(fw, f); err != nil {
		return
	}

	return
}

func (m *MultipartWriter) AddForm(fieldname string, value string) (err error) {

	var form_field io.Writer
	form_field, err = m.w.CreateFormField(fieldname)
	if err != nil {
		err = fmt.Errorf("(AddForm) CreateFormField returned: %s", err.Error())
		return
	}

	_, err = form_field.Write([]byte(value))
	if err != nil {
		err = fmt.Errorf("(AddForm) form_field.Write returned: %s", err.Error())
		return
	}

	return
}
