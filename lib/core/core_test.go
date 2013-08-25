package core

import (
	"encoding/json"
	"labix.org/v2/mgo/bson"
	"testing"
)

func TestIOmap(t *testing.T) {
	print("\nTestIOmap\n")
	i := NewIOmap()
	i.Add("qc.passed.fna", "http://shock.mcs.anl.gov:8000/node/fad5eabc7602b1fcebcaff518266805f", "ff3f41a91bbf135b38d0b35b1df3a42e", false)
	i.Add("qc.failed.fna", "http://shock.mcs.anl.gov:8000/node/f361be3e7a0914f82147bc1ba68df41e", "dff5aa75f124db423cda694c16254f69", false)
	m, _ := json.Marshal(i)
	print(string(m) + "\n")

	if !i.Has("qc.passed.fna") || !i.Has("qc.failed.fna") {
		t.Fail()
	}

	if v := i.Find("qc.passed.fna"); v == nil {
		t.Fail()
	}

	if v := i.Find("qc.failed.fna"); v == nil {
		t.Fail()
	}

	if v := i.Find("doesnotexist"); v != nil {
		t.Fail()
	}
}

func TestCommand(t *testing.T) {
	print("\nTestCommand\n")
	c := Command{Name: "superblat", Description: "", Options: "-p8 -o8", Template: "inputs::i1 inputs::i2 > outputs::o1"}
	c.Substitute(nil, nil)
	m, _ := json.Marshal(c)
	print(string(m) + "\n")
}

func TestTask(t *testing.T) {
	print("\nTestTask\n")
	nt := NewTask()
	m, _ := json.Marshal(nt)
	print(string(m) + "\n")
}

func BenchmarkTask(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nt := NewTask()
		json.Marshal(nt)
	}
}

func TestPipeline(t *testing.T) {
	print("\nTestPipeline\n")
	p := NewPipeline()
	m, _ := json.Marshal(p)
	print(string(m) + "\n")
}

func BenchmarkPipeline(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p := NewPipeline()
		json.Marshal(p)
	}
}

func BenchmarkPipelineB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p := NewPipeline()
		bson.Marshal(p)
	}
}
