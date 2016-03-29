package module

import (
	"encoding/json"
	"fsea/env"
	"fsea/pool"
	"gwf"
	"net/http"
	"os"
	"strconv"
)

type Mount struct {
}

func writeError(w http.ResponseWriter, status int, err *env.Error) {
	w.WriteHeader(status)
	w.Write([]byte(err.Error()))
}

func (m Mount) Action(ctx *gwf.Context) {
	depth := ctx.Depth()
	w := ctx.Writer()
	if depth < 3 || depth > 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	config := env.GetConfig()
	bucketId, _ := ctx.Path(1)
	if depth == 3 {
		name, _ := ctx.Path(2)

		b, f, err := config.AssignFile(bucketId, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, env.NewError(UnspecificError, err.Error()))
			return
		}
		f.Name = name

		fullName := b.Path + "/" + f.Name
		p := pool.GetPool()
		if err = p.AddFile(bucketId, f.Id, fullName); err != nil {
			writeError(w, http.StatusInternalServerError, env.NewError(UnspecificError, err.Error()))
			return
		}
		config.AddFileAndSave(bucketId, f)

		m.response(w, pool.TransId(bucketId, f.Id), fullName)

	} else if depth == 4 {
		p2, _ := ctx.Path(2)
		p3, _ := ctx.Path(3)
		bucketSize, _ := strconv.Atoi(p2)
		numberOfBuckets, _ := strconv.Atoi(p3)

		// 桶过大
		if bucketSize < 1 || bucketSize > 2048 {
			writeError(w, http.StatusBadRequest, env.NewError(InvalidBucketSize, ""))
			return
		}

		// max is 16GB.
		if bucketSize*numberOfBuckets > 1<<34 {
			writeError(w, http.StatusBadRequest, env.NewError(InvalidFileSize, ""))
			return
		}

		// Now mount
		b, f, err := config.AssignFile(bucketId, strconv.FormatInt(int64(bucketSize), 10))
		if err != nil {
			writeError(w, http.StatusInternalServerError, env.NewError(UnspecificError, err.Error()))
			return
		}
		fullName := b.Path + "/" + f.Name
		bucketSize = bucketSize * 4096 // this is actual size
		p := pool.GetPool()
		err = p.MountFile(bucketId, f.Id, fullName, int32(bucketSize), int32(numberOfBuckets))
		if err != nil {
			writeError(w, http.StatusInternalServerError, env.NewError(UnspecificError, err.Error()))
			return
		}
		config.AddFileAndSave(bucketId, f)
		m.response(w, pool.TransId(bucketId, f.Id), fullName)
	}
}

func (m *Mount) response(w http.ResponseWriter, id string, name string) {
	fi, err := os.Stat(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, env.NewError(UnspecificError, err.Error()))
		return
	}

	data, err := json.Marshal(map[string]string{
		"id":   id,
		"size": strconv.FormatInt(fi.Size(), 10),
		"name": name,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, env.NewError(UnspecificError, err.Error()))
		return
	}

	w.Write(data)
}

type Umount struct {
}

func (u Umount) Action(ctx *gwf.Context) {

}
