package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"smlToHttp/sml"

	"gopkg.in/yaml.v3"
)

func main() {
	configFileFlag := flag.String("config", "", "The config file")
	dumpFlag := flag.String("dump", "", "A file to decode binary SML messages from for debugging. Prints the contents to the terminal and then exits.")

	flag.Parse()

	if len(*configFileFlag) == 0 && len(*dumpFlag) == 0 {
		print(
			"SML to HTTP proxy\n" +
				"  Copyright (C) 2024  Stephan Brunner\n" +
				"  This program comes with ABSOLUTELY NO WARRANTY.\n" +
				"  This is free software, and you are welcome to redistribute it\n" +
				"  under the terms of the GNU GPL v3; see LICENSE.txt and README.md for details.\n\n" +
				"  The source code is available at https://github.com/boomer41/SML-to-HTTP-proxy\n\n",
		)
		flag.Usage()
		return
	}

	if len(*dumpFlag) != 0 {
		dumpFile(*dumpFlag)
		return
	}

	cfg, err := loadConfig(*configFileFlag)

	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	l := newLogger()

	image := newProcessImageManager(cfg)
	exporter := newWebExporter(image, l.newSubLogger("web"))
	meters := newMeterManager(cfg.Meters, image, l.newSubLogger("meterManager"))

	errorChannel := make(chan error)

	go func() {
		err := exporter.serve(&cfg.Web)

		if err == http.ErrServerClosed {
			return
		}

		errorChannel <- err
	}()

	go func() {
		err := meters.run()
		errorChannel <- err
	}()

	for {
		err := <-errorChannel

		log.Fatalf("subsystem returned error: %v", err)
	}
}

func loadConfig(path string) (*config, error) {
	var configContent []byte
	{
		f, err := os.OpenFile(path, os.O_RDONLY, 0)

		if err != nil {
			return nil, err
		}

		defer f.Close()

		configContent, err = io.ReadAll(f)

		if err != nil {
			return nil, err
		}
	}

	var c config
	err := yaml.Unmarshal(configContent, &c)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func dumpFile(filePath string) {
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0)

	if err != nil {
		fmt.Printf("failed to load file to dump: %v\n", err)
		os.Exit(1)
	}

	defer f.Close()

	reader := sml.NewReader(f)

	dumpedAny := false

	for {
		var msg *sml.File
		msg, err = reader.ReadFile()

		if err != nil {
			break
		}

		dumpedAny = true
		fmt.Println("found file:")
		fmt.Printf("%s\n\n", msg)
	}

	if err == nil || err == io.EOF {
		if !dumpedAny {
			fmt.Printf("no valid sml files found in file\n")
			os.Exit(2)
		}

		return
	}

	fmt.Printf("failed to read from file: %v\n", err)
	os.Exit(1)
}
