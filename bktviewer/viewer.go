package main

import (
	"bktfile"
	"flag"
	"fmt"
	"strconv"
	"time"
)

func main() {
	flag.Parse()
	name := flag.Arg(0)

	if name == "" {
		fmt.Println("Please specify the file name.")
		return
	}

	arg2 := flag.Arg(1)
	index, err := strconv.Atoi(arg2)
	if err != nil {
		index = 0
	}

	f, err := bktfile.OpenFile(name, bktfile.OF_RDONLY)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer f.Close()

	data, t, err := f.Read(int32(index))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Print(f.FileHeader())
	fmt.Println(time.Unix(t, 0).Format("2006-01-02 15:04:05"))
	fmt.Println(string(data))
}
