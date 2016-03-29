package pool

import (
	"bktfile"
	"errors"
	"fmt"
	"fsea/env"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

func TransId(bid string, fid string) string {
	return fmt.Sprintf("%s:%s", bid, fid)
}

type Pool struct {
	// 针对buckets的lock
	lock    sync.RWMutex
	buckets map[string]*File
	files   FileSet
}

var pool *Pool

func GetPool() *Pool {
	if pool == nil {
		pool = &Pool{}
		pool.Init()
	}
	return pool
}

// 初始化，不需要加锁
func (p *Pool) Init() {
	p.buckets = make(map[string]*File)
	config := env.GetConfig()
	for _, bucket := range config.Bucket {
		for _, file := range bucket.File {
			name := bucket.Path + string(os.PathSeparator) + file.Name
			id := TransId(bucket.Id, file.Id)
			if err := p.loadFile(id, name); err == nil {
				log.Printf("(%s)loaded: %s\n", id, name)
			} else {
				log.Println(err)
			}
		}
	}
}

// id规则：[bid:fid]
func (p *Pool) GetFile(id string) *File {
	defer p.lock.RUnlock()
	p.lock.RLock()

	if f, ok := p.buckets[id]; ok {
		return f
	}
	return nil
}

func (p *Pool) loadFile(id string, name string) error {
	f, err := bktfile.OpenFile(name, bktfile.OF_RDWR)
	if err != nil {
		log.Printf("failed to load file %s.[%s]", name, err.Error())
		return err
	}
	file := &File{id, f}
	p.buckets[id] = file
	p.files.AddFile(file)
	return nil
}

func (p *Pool) ReloadFile(bid string, fid string) error {
	id := TransId(bid, fid)
	f := p.GetFile(id)
	if f != nil {
		return f.file.Reopen(bktfile.OF_RDWR)
	} else {
		return errors.New("no such file to reload")
	}
}

func (p *Pool) MountFile(bid string, fid string, name string, bucketSize int32, numberOfBuckets int32) error {
	id := TransId(bid, fid)
	if p.GetFile(id) != nil {
		return errors.New("file id already exist.")
	}

	f, err := bktfile.CreateFile(name, 0666, bucketSize, numberOfBuckets)
	if err != nil {
		return err
	}

	file := &File{id, f}
	p.lock.Lock()
	p.buckets[id] = file
	p.lock.Unlock()

	p.files.AddFile(file)
	return nil
}

func (p *Pool) AddFile(bid string, fid string, name string) error {
	id := TransId(bid, fid)
	if p.GetFile(id) != nil {
		return errors.New("file id already exist.")
	}

	f, err := bktfile.OpenFile(name, bktfile.OF_RDWR)
	if err != nil {
		return err
	}
	file := &File{id, f}

	p.lock.Lock()
	p.buckets[id] = file
	p.lock.Unlock()

	p.files.AddFile(file)
	return nil
}

func (p *Pool) Write(data []byte) (string, error) {
	return p.files.Write(data)
}

func (p *Pool) getFileEnv(dataId string) (*File, int32, *env.Error) {
	sep := strings.LastIndex(dataId, ":")
	if sep == -1 {
		return nil, -1, env.NewError(env.InvalidDataId, dataId)
	}
	id := dataId[:sep]
	index, err := strconv.ParseInt(dataId[sep+1:], 16, 64)
	if err != nil {
		return nil, -1, env.NewError(env.InvalidDataId, err.Error())
	}
	f := p.GetFile(id)
	if f == nil {
		return nil, -1, env.NewError(env.InvalidFileId, id)
	}
	return f, int32(index), nil
}

func (p *Pool) Read(dataId string) ([]byte, int64, *env.Error) {
	f, index, err := p.getFileEnv(dataId)
	if err != nil {
		return nil, -1, err
	}
	d, t, e := f.file.Read(index)
	if e != nil {
		return nil, -1, env.NewError(env.UnspecificError, e.Error())
	}
	return d, t, nil
}

func (p *Pool) Delete(dataId string) *env.Error {
	f, index, err := p.getFileEnv(dataId)
	if err != nil {
		return err
	}
	full := f.file.IsFull()
	defer func() {
		if full && !f.file.IsFull() {
			p.files.AddFile(f)
		}
	}()
	if e := f.file.Empty(index); e != nil {
		return env.NewError(env.UnspecificError, e.Error())
	}
	return nil
}
