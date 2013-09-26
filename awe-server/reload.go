package main

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/db"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	dirExp, _ = regexp.Compile("[a-f0-9]{2}/[a-f0-9]{2}/[a-f0-9]{32}$")
)

func reload(source string) (err error) {
	fmt.Println("source:", conf.RELOAD)
	if strings.HasPrefix(conf.RELOAD, "http://") {
		fmt.Println("source-type: url")
		if err = reloadDB(); err != nil {
			return err
		}
		return reloadFromUrl(source)
	} else {
		fmt.Println("source-type: dir")
		if source == conf.DATA_PATH {
			fmt.Println("source dir is the same as conf data dir: db reload only.")
		} else {
			return errors.New("source dir is not the same as conf data dir: copy not implemented yet")
		}
		if err = reloadDB(); err != nil {
			return err
		}
		return filepath.Walk(source, reloadFromDir)
	}
	return
}

func reloadDB() (err error) {
	fmt.Printf("dropping & re-initializing database...")
	if err = db.Drop(); err != nil {
		return err
	}
	db.Initialize()
	fmt.Printf("done\n")
	return
}

func reloadFromUrl(url string) error {
	return errors.New("reload from url not implemented yet")
}

func reloadFromDir(path string, info os.FileInfo, err error) error {
	if dirExp.MatchString(path) {
		id := filepath.Base(path)
		fmt.Printf(id + "...")
		if err := core.ReloadFromDisk(path); err != nil {
			return err
		}
		fmt.Printf("done\n")
	}
	return nil
}
