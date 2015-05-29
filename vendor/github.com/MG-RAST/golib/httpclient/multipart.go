package httpclient

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path/filepath"
)

type Form struct {
	params      []param
	files       []file
	Length      int64
	Reader      io.Reader
	ContentType string
}

type param struct {
	k string
	v string
}

type file struct {
	n    string
	p    string
	r    io.Reader
	size int64
}

func NewForm() (f *Form) {
	return &Form{}
}

func (f *Form) AddFile(name, path string) {
	f.files = append(f.files, file{n: name, p: path})
	return
}

func (f *Form) AddFileReader(name string, r io.Reader, size int64) {
	f.files = append(f.files, file{n: name, r: r, size: size})
	return
}

func (f *Form) AddParam(key, val string) {
	f.params = append(f.params, param{k: key, v: val})
	return
}

func (f *Form) Dump() (buf []byte) {
	buf, _ = ioutil.ReadAll(f.Reader)
	return
}

func (f *Form) Create() (err error) {
	readers := []io.Reader{}
	buf := bytes.NewBufferString("")
	writer := multipart.NewWriter(buf)

	for _, p := range f.params {
		writer.WriteField(p.k, p.v)
	}
	readers = append(readers, bytes.NewBufferString(buf.String()))
	f.Length += int64(buf.Len())
	buf.Reset()

	for _, file := range f.files {
		if file.p != "" {
			writer.CreateFormFile(file.n, filepath.Base(file.p))
		} else {
			writer.CreateFormFile(file.n, file.n)
		}
		readers = append(readers, bytes.NewBufferString(buf.String()))
		f.Length += int64(buf.Len())
		buf.Reset()
		if file.p == "" {
			f.Length += file.size
			readers = append(readers, file.r)
		} else {
			if fh, err := os.Open(file.p); err == nil {
				if fi, err := fh.Stat(); err == nil {
					f.Length += fi.Size()
					readers = append(readers, fh)
				} else {
					return err
				}
			} else {
				return err
			}
		}
	}

	writer.Close()
	readers = append(readers, bytes.NewBufferString(buf.String()))
	f.Length += int64(buf.Len())
	f.Reader = io.MultiReader(readers...)
	f.ContentType = writer.FormDataContentType()
	return
}
