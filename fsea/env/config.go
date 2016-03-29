package env

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"strconv"
)

type File struct {
	Id   string
	Name string
}

type Bucket struct {
	Id   string
	Path string
	File []*File
}

type Large struct {
	Path string
	Max  string
	Fold string
}

type Config struct {
	// id
	Id string
	// 端口
	Port int16
	// 管理端口
	AdminPort int16
	// 服务ip
	AdminIP []string
	// 注册服务器列表
	Regto []string
	// bucket列表
	Bucket []*Bucket
	// large对象
	Large Large
}

var config *Config
var fileName string

func CreateConfig(name string) (*Config, error) {
	config = &Config{}
	if _, err := toml.DecodeFile(name, config); err != nil {
		config = nil
		return nil, err
	} else {
		fileName = name
		return config, nil
	}
}

// 从文件中读取配置
func GetConfig() *Config {
	return config
}

func (c *Config) Save() error {
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := toml.NewEncoder(f)
	return encoder.Encode(c)
}

// 根据桶id和seed分配一个文件对象。并不真实增加一个文件。
// bid 指定桶对象目录
// seed 文件类型。系统自动根据 fid_name.bkt 格式生成文件名
// 返回值：
//		File对象，描述分配的文件信息
//		文件的路径
//		可能返回的错误信息
func (c *Config) AssignFile(bid string, seed string) (*Bucket, *File, error) {
	for _, bucket := range c.Bucket {
		if bucket.Id == bid {
			var maxId int64 = -1
			for _, file := range bucket.File {
				fid, err := strconv.ParseInt(file.Id, 16, 64)
				if err != nil {
					return nil, nil, err
				}
				if fid > maxId {
					maxId = fid
				}
			}
			fid := strconv.FormatInt(maxId+1, 16)
			f := File{
				Id:   fid,
				Name: fmt.Sprintf("%s_%s.bkt", fid, seed),
			}
			return bucket, &f, nil
		}
	}
	return nil, nil, errors.New("bid not found.")
}

// 将文件对象加入到配置中
func (c *Config) AddFile(bid string, f *File) error {
	for _, bucket := range c.Bucket {
		if bucket.Id == bid {
			for _, file := range bucket.File {
				if file.Id == f.Id || file.Name == f.Name {
					return errors.New("File id is used.")
				}
			}
			bucket.File = append(bucket.File, f)
			return nil
		}
	}
	return errors.New("bid not found")
}

// 将文件对象加入到配置中
func (c *Config) AddFileAndSave(bid string, f *File) error {
	if err := c.AddFile(bid, f); err != nil {
		return err
	}
	return c.Save()
}
