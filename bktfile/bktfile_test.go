package bktfile

import (
	//	"files"
	"fmt"
	//"log"
	"os"
	"testing"
)

var testPath string
var samplePath string
var mtrl1, mtrlFmt string

func init() {
	testPath = "/tmp/bktfile/test/"
	samplePath = "/tmp/bktfile/material/"

	mtrl1 = "This is first bucket writed for CreateFile"
	mtrlFmt = "Meterial infomation item %d, routine %d"

}

/*
func TestCreateFile(t *testing.T) {
	name := testPath + "testCreateFile.bkt"
	f, err := CreateFile(name, 0666, 512, 1024)
	if err != nil {
		t.Error(err)
		return
	}

	defer f.Close()

	i, err := f.Write([]byte(mtrl1))
	if err != nil {
		t.Error(err)
		return
	}

	if i != 0 {
		fmt.Println(i)
		t.Fail()
	}
}

func TestOpenFile(t *testing.T) {
	name := testPath + "testCreateFile.bkt"
	f, err := OpenFile(name, OF_RDONLY)
	if err != nil {
		t.Error(err)
		return
	}

	d, _, err := f.Read(0)
	if err != nil {
		t.Error(err)
		return
	}

	if string(d) != mtrl1 {
		t.Error("bucket 0 infomation is not matched")
		return
	}
}
*/
func TestRoutineWrite(t *testing.T) {
	name := testPath + "testRoutineFile.bkt"

	os.Remove(name)
	f, err := CreateFile(name, 0666, 512, 4096)
	if err != nil {
		t.Error(err)
		return
	}

	defer f.Close()

	c := make(chan int, 32)
	for i := 0; i < 32; i++ {
		go writeDataRoutine(t, f, i, c)
	}

	for i := 0; i < 32; i++ {
		<-c
	}

	f.Close()
	t.Logf("file Closed\n")
	f, err = OpenFile(name, OF_RDWR)

	for i := 0; i < 32; i++ {
		go emptyDataRoutine(t, f, i, c)
	}

	for i := 0; i < 32; i++ {
		<-c
	}
	f.Close()

	t.Logf("file Closed\n")

	f, err = OpenFile(name, OF_RDWR)
	writeDataRoutine(t, f, 0, c)
}

func emptyDataRoutine(t *testing.T, f *File, index int, c chan int) {
	i := int32(index*32 + 3)
	f.Empty(i)
	t.Logf("bucket %d is emptyed\n", i)
	c <- index
}

func writeDataRoutine(t *testing.T, f *File, index int, c chan int) {
	var bkIndex int32
	for i := 1; i < 256; i++ {
		bkIndex, _ = f.Write([]byte(fmt.Sprintf(mtrlFmt, i, index)))
		t.Logf("routine(%d) write bucket index: %d\n", index, bkIndex)
	}
	t.Logf("routine %d finished\n", index)
	c <- index
}
