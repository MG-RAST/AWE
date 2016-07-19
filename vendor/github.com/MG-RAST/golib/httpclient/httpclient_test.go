package httpclient_test

import (
	"fmt"
	. "github.com/MG-RAST/Shock/shock-client/lib/httpclient"
	"io/ioutil"
	"testing"
)

func TestForm(t *testing.T) {
	f := NewForm()
	f.AddParam("upload_type", "true")
	f.AddParam("parts", "25")
	f.AddFile("upload", "/Users/jared/go/kbno.cfg")
	f.AddFile("upload", "/Users/jared/go/shock.cfg")
	if err := f.Create(); err != nil {
		println(err.Error())
	}
	println("Content-Type: " + f.ContentType)
	fmt.Printf("Content-Length: %d\n", f.Length)
	if form, err := ioutil.ReadAll(f.Reader); err != nil {
		println(err.Error())
	} else {
		fmt.Printf("%s", form)
		fmt.Printf("len: %d\n", len(form))
	}
	f.Close()
}
