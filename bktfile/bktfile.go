// bktfile是Bucket File的缩写。
// bktfile由一系列固定大小的桶数组组成。
package bktfile

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"
	"time"
)

type FileHeader struct {
	Magic           uint16 // "BF"
	MajorVersion    uint8  // 主版本号
	MinorVersion    uint8  // 次版本号
	HeaderSize      int16  // 文件头部大小
	BucketSize      int32  // Bucket大小
	NumberOfBuckets int32  // 文件中Bucket个数
	// 以下两个参数运行过程中可变
	NumberOfEmptyBuckets int32 // 文件中可用Bucket个数
	IndexOfEmptyBucket   int32 // 第一个可用空桶位置
}

type Bucket struct {
	DataLength int32 // 数据部分长度，当BucketStatus是'\0'时
	Status     int8  // 'u' 已用 '\0' 未用 'd' 回收站 'e' 错误状态
	TimeStamp  int64 // 从1970年1月1日开始的秒数
	HeaderSize uint8 // 桶头的头大小

}

var defaultFileHeader FileHeader
var defaultBucket Bucket

var sizeOfFileHeader, sizeOfBucketHeader int

var majorVersion, minorVersion uint8

func init() {
	sizeOfFileHeader = binary.Size(defaultFileHeader)
	sizeOfBucketHeader = binary.Size(defaultBucket)
	majorVersion = 0
	minorVersion = 1
}

const (
	BUCKETFILE_MAGIC      uint16 = 0x4642
	INVALID_INDEX         int32  = -1
	INVALID_LENGTH32      int32  = -1
	INVALID_LENGTH64      int64  = -1
	INVALID_POINTER       int64  = -1
	BUCKET_STATUS_EMPTY   int8   = 0
	BUCKET_STATUS_USED    int8   = 'u'
	BUCKET_STATUS_DELETED int8   = 'd'
	BUCKET_STATUS_ERROR   int8   = 'e'
)

type File struct {
	fh     FileHeader
	closer io.Closer
	reader io.ReaderAt
	writer io.WriteSeeker
	locker sync.Mutex

	name string
}

func (h *FileHeader) isValid() bool {
	if h.Magic != BUCKETFILE_MAGIC {
		return false
	}

	if h.MajorVersion > majorVersion || h.MajorVersion > minorVersion {
		return false
	}

	if h.NumberOfEmptyBuckets > h.NumberOfBuckets || h.IndexOfEmptyBucket > h.NumberOfBuckets {
		return false
	}

	return true
}

func (h *FileHeader) indexToPointer(index int32) int64 {
	return int64(h.HeaderSize) + int64(index)*int64(h.BucketSize)
}

func (h *FileHeader) isFull() bool {
	return h.NumberOfEmptyBuckets == 0 || h.IndexOfEmptyBucket == h.NumberOfBuckets
}

func (b *Bucket) isEmpty() bool {
	return b.Status == BUCKET_STATUS_EMPTY
}

func (b *Bucket) isUsed() bool {
	return b.Status == BUCKET_STATUS_USED
}

func (b *Bucket) isDeleted() bool {
	return b.Status == BUCKET_STATUS_DELETED
}

func (b *Bucket) isError() bool {
	return b.Status == BUCKET_STATUS_ERROR
}

func (b *Bucket) setStatus(status int8) {
	b.Status = status
}

func (b *Bucket) dataLength() int32 {
	if b.isUsed() || b.isDeleted() {
		return b.DataLength
	} else {
		return INVALID_LENGTH32
	}
}

func (b *Bucket) indexOfNextEmptyBucket() int32 {
	if b.isEmpty() {
		return b.DataLength
	} else {
		return INVALID_INDEX
	}
}

func (b *Bucket) setIndexOfNextEmptyBucket(index int32) {
	b.DataLength = index
}

const (
	OF_RDONLY = os.O_RDONLY
	OF_RDWR   = os.O_RDWR
)

// 创建一个指定桶大小，桶个数的桶文件
func CreateFile(name string, perm os.FileMode, bucketSize int32, numberOfBuckets int32) (*File, error) {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return nil, err
	}
	bf := new(File)
	bf.name = name
	bf.fh = FileHeader{
		BUCKETFILE_MAGIC,
		majorVersion,
		minorVersion,
		int16(sizeOfFileHeader),
		bucketSize,
		numberOfBuckets,
		numberOfBuckets,
		0,
	}

	fileSize := int64(bucketSize)*int64(numberOfBuckets) + int64(bf.fh.HeaderSize)
	if err := f.Truncate(fileSize); err != nil {
		f.Close()
		return nil, err
	}

	if err = binary.Write(f, binary.LittleEndian, bf.fh); err != nil {
		f.Close()
		return nil, err
	}

	bf.writer, bf.reader, bf.closer = f, f, f

	return bf, nil
}

// 打开一个桶文件进行读或者写
func OpenFile(name string, flag int) (*File, error) {
	f, err := os.OpenFile(name, flag, 0000)
	if err != nil {
		return nil, err
	}

	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	bf := new(File)
	if fi.Size() < int64(sizeOfFileHeader) {
		return nil, errors.New("Invalid file length.")
	}

	if err := binary.Read(f, binary.LittleEndian, &bf.fh); err != nil {
		return nil, err
	}

	if !bf.fh.isValid() {
		return nil, errors.New("Not a valid bucket file")
	}

	bf.reader = f
	bf.closer = f
	if (flag & OF_RDWR) == OF_RDWR {
		bf.writer = f
	}
	bf.name = name

	f = nil
	return bf, nil
}

func (f *File) Reopen(flag int) error {
	name := f.Name()

	defer f.locker.Unlock()
	f.locker.Lock()
	f.Close()

	file, err := OpenFile(name, flag)
	if err != nil {
		return err
	}
	f.fh = file.fh
	f.reader = file.reader
	f.writer = file.writer
	f.closer = file.closer
	f.name = name

	return nil
}

func (f *File) readBucket(pointerToBucket int64) (*Bucket, error) {
	bucket := new(Bucket)
	sr := bufio.NewReader(io.NewSectionReader(f.reader, pointerToBucket, int64(f.fh.BucketSize)))
	if err := binary.Read(sr, binary.LittleEndian, bucket); err != nil {
		return nil, err
	}
	return bucket, nil
}

// 返回数据，写入时间，或者错误
func (f *File) readData(pointerToBucket int64) ([]byte, int64, error) {
	if f.reader == nil {
		return nil, 0, errors.New("File is not readable")
	}
	var bucket Bucket
	sr := bufio.NewReader(io.NewSectionReader(f.reader, pointerToBucket, int64(f.fh.BucketSize)))
	if err := binary.Read(sr, binary.LittleEndian, &bucket); err != nil {
		return nil, 0, err
	}
	// it's a empty bucket
	if bucket.isEmpty() {
		return nil, 0, nil
	}

	if bucket.isUsed() {
		if bucket.DataLength > f.fh.BucketSize-int32(bucket.HeaderSize) {
			return nil, 0, errors.New("Invalid bucket data size.")
		}
		data := make([]byte, bucket.DataLength)
		if _, err := sr.Read(data); err != nil {
			return nil, 0, err
		} else {
			return data, bucket.TimeStamp, nil
		}
	} else {
		return nil, 0, nil
	}
}

func (f *File) FileHeader() FileHeader {
	return f.fh
}

func (f *File) Name() string {
	return f.name
}

func (f *File) IsFull() bool {
	return f.fh.NumberOfEmptyBuckets == 0
}

func (f *File) flushHead() error {
	if _, err := f.writer.Seek(0, 0); err != nil {
		return err
	}
	if err := binary.Write(f.writer, binary.LittleEndian, f.fh); err != nil {
		return err
	}
	return nil
}

// 从指定桶读取数据并返回。如果是空桶，则返回空。
func (f *File) Read(index int32) ([]byte, int64, error) {
	if index >= f.fh.NumberOfBuckets {
		return nil, 0, errors.New("Index overflows")
	}
	return f.readData(f.fh.indexToPointer(index))
}

// 将len(data)的数据写入一个空桶。如果数据大小大于桶可以容纳的数据大小则失败
func (f *File) Write(data []byte) (int32, error) {
	dataLength := len(data)
	if dataLength > int(f.fh.BucketSize)-int(sizeOfBucketHeader) {
		return -1, errors.New("Data is too long.")
	}

	if f.writer == nil {
		return -1, errors.New("File not writealbe.")
	}

	if f.fh.isFull() {
		return -1, errors.New("Bucket file is full.")
	}

	defer f.locker.Unlock()
	f.locker.Lock()

	pointToEmptyBucket := f.fh.indexToPointer(f.fh.IndexOfEmptyBucket)
	bucket, err := f.readBucket(pointToEmptyBucket)
	if err != nil {
		return -1, err
	}
	if !bucket.isEmpty() {
		return -1, errors.New("Empty bucket wanted, but nonempty bucket found.")
	}

	indexOfNextEmptyBucket := bucket.indexOfNextEmptyBucket()
	if indexOfNextEmptyBucket == 0 {
		indexOfNextEmptyBucket = f.fh.IndexOfEmptyBucket + 1
	}

	bucket.setStatus(BUCKET_STATUS_USED)
	bucket.DataLength = int32(dataLength)
	bucket.TimeStamp = time.Now().Unix()
	bucket.HeaderSize = uint8(sizeOfBucketHeader)

	indexOfThisBucket := f.fh.IndexOfEmptyBucket
	pointToThisBucket := f.fh.indexToPointer(indexOfThisBucket)

	// 写文件头，如果桶写失败，最多这个桶就废了，不至于文件坏掉
	if _, err = f.writer.Seek(0, 0); err != nil {
		return -1, err
	}

	f.fh.IndexOfEmptyBucket = indexOfNextEmptyBucket
	f.fh.NumberOfEmptyBuckets--

	defer func() {
		// 如果头或者桶写失败，头部索引先回滚。至少保证文件关闭前，当前写失败的桶还能继续使用。
		if err != nil {
			f.fh.IndexOfEmptyBucket = indexOfThisBucket
			f.fh.NumberOfEmptyBuckets++
		}
	}()

	if err = binary.Write(f.writer, binary.LittleEndian, f.fh); err != nil {
		return -1, err
	}

	// 先写桶和数据
	if _, err = f.writer.Seek(pointToThisBucket, 0); err != nil {
		return -1, err
	}
	bufwriter := bufio.NewWriter(f.writer)
	if err = binary.Write(bufwriter, binary.LittleEndian, *bucket); err != nil {
		return -1, err
	}
	if _, err = bufwriter.Write(data); err != nil {
		return -1, err
	}
	if err = bufwriter.Flush(); err != nil {
		return -1, err
	}

	return indexOfThisBucket, nil
}

// 清空回收指定索引的桶
func (f *File) Empty(index int32) error {
	if index >= f.fh.NumberOfBuckets {
		return errors.New("Index overflows.")
	}

	defer f.locker.Unlock()
	f.locker.Lock()

	pointerToBucket := f.fh.indexToPointer(index)
	bucket, err := f.readBucket(pointerToBucket)
	if err != nil {
		return err
	}

	if !bucket.isEmpty() {
		bucket.setIndexOfNextEmptyBucket(f.fh.IndexOfEmptyBucket)
		bucket.setStatus(BUCKET_STATUS_EMPTY)
		bucket.TimeStamp = time.Now().Unix()
		bucket.HeaderSize = uint8(sizeOfBucketHeader)

		if _, err = f.writer.Seek(pointerToBucket, 0); err != nil {
			return err
		}
		if err = binary.Write(f.writer, binary.LittleEndian, bucket); err != nil {
			return err
		}

		f.fh.NumberOfEmptyBuckets++
		f.fh.IndexOfEmptyBucket = index

		if _, err = f.writer.Seek(0, 0); err != nil {
			return err
		}

		if err = binary.Write(f.writer, binary.LittleEndian, f.fh); err != nil {
			return err
		}
	}
	return nil
}

// 关闭文件
func (f *File) Close() error {
	defer func() {
		f.fh = defaultFileHeader
		f.writer, f.reader, f.closer = nil, nil, nil
		f.name = ""
	}()
	if f.closer != nil {
		return f.closer.Close()
	} else {
		return errors.New("not a valid closer.")
	}
}
