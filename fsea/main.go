package main

import (
	"fmt"
	"fsea/env"
	"fsea/module"
	"fsea/pool"
	"gwf"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

func init() {
}

func main() {

	var dispatcher gwf.Dispatcher
	f, _ := filepath.Abs(os.Getenv("_"))
	p := path.Dir(path.Dir(f))
	name := p + "/conf/fsea.conf"
	c, err := env.CreateConfig(name)
	if err != nil {
		fmt.Println(err)
		return
	}

	pool.GetPool()

	dispatcher.AddModule("mount", module.Mount{})
	dispatcher.AddModule("umount", module.Umount{})
	go http.ListenAndServe(fmt.Sprintf(":%d", c.AdminPort), &dispatcher)

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", c.Port),
		Handler:        module.Serve{},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
