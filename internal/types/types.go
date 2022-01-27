package types

import (
	"encoding/json"
	"log"
	"os"
)

// Root struct for commands
// Field tags control where the values come from
// If opts:"-" yaml:"-" are set in object creation
//    opts:="-" come from config file
//    yaml:="-" come from command line flags
type Root struct {
}

func Config(filename string, dump bool, in interface{}) {
	if filename != "" {
		fd, err := os.Open(filename)
		// config is in its own func
		// this defer fire correctly
		//
		// won't fire if dump is used as os.Exit terminates program
		defer func() {
			fd.Close()
		}()
		if err != nil {
			log.Fatalf("error opening file %s %v", filename, err)
		}
		dec := json.NewDecoder(fd)
		err = dec.Decode(in)
		if err != nil {
			log.Fatalf("json error %v", err)
		}
	}
	if dump {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		err := enc.Encode(in)
		if err != nil {
			log.Fatalf("json encoding error %v", err)
		}
		os.Exit(0)
	}
}
