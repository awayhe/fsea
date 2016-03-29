package pool

import (
	"bktfile"
	"errors"
	"fmt"
	"sort"
	"sync"
)

type File struct {
	id   string
	file *bktfile.File
}

func (f *File) Weight() float64 {
	fd := f.file.FileHeader()
	return float64(fd.NumberOfEmptyBuckets) / float64(fd.NumberOfBuckets)
}

type Files struct {
	size  int32
	files []*File
}

type FileSet struct {
	lock    sync.RWMutex
	fileset []*Files
}

type ByWeight []*File

func (w ByWeight) Len() int           { return len(w) }
func (w ByWeight) Swap(i, j int)      { w[i], w[j] = w[j], w[i] }
func (w ByWeight) Less(i, j int) bool { return w[i].Weight() > w[j].Weight() } // 倒序排列

func (f *File) genId(index int32) string {
	return fmt.Sprintf("%s:%x", f.id, index)
}

func (fs *Files) Sort() {
	sort.Sort(ByWeight(fs.files))
}

func (fs *Files) Write(data []byte) (string, error) {
	f := fs.files[0]
	index, err := f.file.Write(data)

	// err的情况下也可能引起文件满
	if f.file.IsFull() {
		fs.files = fs.files[1:]
	} else {
		fs.Sort()
	}

	if err != nil {
		return "", err
	}
	return f.genId(index), nil
}

func (fs *Files) AppendFile(f *File) {
	fs.files = append(fs.files, f)
	fs.Sort()
}

func (fs *Files) IsFull() bool {
	return len(fs.files) == 0
}

func (s *FileSet) AddFile(f *File) {
	if f.file.IsFull() {
		return
	}

	bucketSize := f.file.FileHeader().BucketSize

	defer s.lock.Unlock()
	s.lock.Lock()

	count := len(s.fileset)
	i := sort.Search(count, func(i int) bool { return s.fileset[i].size >= bucketSize })

	if i < count {
		fs := s.fileset[i]
		if fs.size == bucketSize {
			fs.AppendFile(f)
		} else {
			fs = &Files{bucketSize, []*File{f}}
			s.insert(fs, i)
		}
	} else {
		fs := &Files{bucketSize, []*File{f}}
		s.fileset = append(s.fileset, fs)
	}
}

func (s *FileSet) insert(fs *Files, i int) {
	ss := make([]*Files, len(s.fileset)+1)
	at := copy(ss, s.fileset[:i])
	ss[i] = fs
	copy(ss[at+1:], s.fileset[i:])
	s.fileset = ss
}

func (s *FileSet) Write(data []byte) (string, error) {
	count := len(s.fileset)
	if count == 0 {
		return "", errors.New("No valid bucket files.")
	}

	defer s.lock.Unlock()
	s.lock.Lock()
	size := int32(len(data))
	i := sort.Search(count, func(i int) bool { return s.fileset[i].size > size })
	if i < count {
		id, err := s.fileset[i].Write(data)
		if s.fileset[i].IsFull() { // remove full item from list
			s.fileset = append(s.fileset[:i], s.fileset[i+1:]...)
		}
		return id, err
	}
	return "", errors.New("Data too large")
}
