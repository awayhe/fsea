package module

import (
	"encoding/json"
	"log"
)

const (
	UnspecificError   = -1
	InvalidBucketId   = 101
	InvalidFileId     = 102
	InvalidBucketSize = 103
	InvalidFileSize   = 104
	FileNotFound      = 105
	InvalidDataId     = 106
)

var statusText = map[int]string{
	UnspecificError:   "Unspecific Error",
	InvalidBucketId:   "BucketId is invalid",
	InvalidFileId:     "FileId is invalid",
	InvalidBucketSize: "Bucket size is too large, valid range is [1, 2048]",
	InvalidFileSize:   "File size is too large. the file size should smaller then 4GB",
	FileNotFound:      "File is not found",
	InvalidDataId:     "DataId is invalid",
}

type Error struct {
	Err     int
	Message string
	Detail  string
}

func NewError(errCode int, detail string) *Error {
	message, _ := statusText[errCode]
	return &Error{
		errCode,
		message,
		detail,
	}
}

func (e *Error) Marshal() []byte {
	if v, err := json.Marshal(e); err != nil {
		log.Println(err)
		return []byte(``)
	} else {
		return v
	}
}

func (e Error) Error() string {
	return string(e.Marshal())
}
