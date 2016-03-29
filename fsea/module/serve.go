package module

import (
	"fmt"
	"fsea/env"
	"fsea/pool"
	"io/ioutil"
	"net/http"
	"time"
)

type Serve struct{}

func (s Serve) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" || r.Method == "POST" {
		s.doPut(w, r)
	} else if r.Method == "GET" {
		s.doGet(w, r)
	} else if r.Method == "DELETE" {
		s.doDelete(w, r)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (s Serve) doGet(w http.ResponseWriter, r *http.Request) {
	p := pool.GetPool()
	if d, t, err := p.Read(r.URL.Path[1:]); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	} else {
		w.Header().Add("Last-Modified", time.Unix(t, 0).Format(time.RFC1123))
		w.Write(d)
	}
}

func (s Serve) doDelete(w http.ResponseWriter, r *http.Request) {
	p := pool.GetPool()
	if err := p.Delete(r.URL.Path[1:]); err != nil {
		writeError(w, http.StatusBadRequest, err)
	}
}

func (s Serve) doPut(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		var data []byte
		data, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			writeError(w, http.StatusBadRequest, env.NewError(UnspecificError, err.Error()))
			return
		}
		p := pool.GetPool()
		id, err := p.Write(data)
		if err != nil {
			writeError(w, http.StatusInternalServerError, env.NewError(UnspecificError, err.Error()))
		} else {
			w.Write([]byte(fmt.Sprintf("{id: \"%s\"}", id)))
		}
	}
}
