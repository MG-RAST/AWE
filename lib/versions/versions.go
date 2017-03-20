package versions

import (
	"bufio"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/db"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"os"
	"strconv"
)

type Version struct {
	Name    string `bson:"name" json:"name"`
	Version int    `bson:"version" json:"version"`
}

type Versions []Version

var VersionMap = make(map[string]int)

func Initialize() (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Versions")
	c.EnsureIndex(mgo.Index{Key: []string{"name"}, Unique: true})
	var versions = new(Versions)
	err = c.Find(bson.M{}).All(versions)
	if err != nil {
		return
	}
	for _, v := range *versions {
		VersionMap[v.Name] = v.Version
	}
	if err = RunVersionUpdates(); err != nil {
		return
	}
	// After version updates have succeeded without error, we can push the configured version numbers into the mongo db
	// Note: configured version numbers are configured in conf.go but are NOT user configurable by design
	if err = PushVersionsToDatabase(); err != nil {
		return
	}
	return
}

func Print() (err error) {
	fmt.Printf("##### Versions ####\n")
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Versions")
	var versions = new(Versions)
	if err = c.Find(bson.M{}).All(versions); err != nil {
		return err
	}
	for _, v := range *versions {
		fmt.Printf("name: %v\tversion number: %v\n", v.Name, v.Version)
	}
	fmt.Println("")
	return
}

func PushVersionsToDatabase() (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Versions")
	for k, v := range conf.VERSIONS {
		if _, err = c.Upsert(bson.M{"name": k}, bson.M{"$set": bson.M{"name": k, "version": v}}); err != nil {
			return err
		}
	}
	return
}

func RunVersionUpdates() (err error) {
	// get Job struct version
	confVersionJob, ok1 := conf.VERSIONS["Job"]
	dbVersionJob, ok2 := VersionMap["Job"]

	// Upgrading database if the Job version is less than 2 (or no Job version number exists) and
	// the Job version in the config file exists and is greater than or equal to 2.
	if (ok1 && confVersionJob >= 2) && (!ok2 || (ok2 && dbVersionJob < 2)) {
		consoleReader := bufio.NewReader(os.Stdin)
		session := db.Connection.Session.Copy()
		defer session.Close()
		c := session.DB(conf.MONGODB_DATABASE).C("Jobs")
		jCount, err := c.Find(bson.M{}).Count()
		if err != nil {
			return err
		}
		if jCount > 0 {
			fmt.Println("The Job struct version in your database needs updating to version 2 to run this version of the AWE server.")
			fmt.Println("Your AWE server currently has " + strconv.Itoa(jCount) + " jobs in it.")
			fmt.Print("Would you like to run the update to convert these jobs to the version 2 Job struct? (y/n): ")
			text, _ := consoleReader.ReadString('\n')
			if text[0] == 'y' {
				var job = new(core.Job)
				var jobDep = new(core.JobDep)
				cnames, err := session.DB(conf.MONGODB_DATABASE).CollectionNames()
				if err != nil {
					return err
				}
				// Get temporary name for new Jobs collection
				cname_tmp := ""
				num := 1
				var cnameExists = true
				for cnameExists {
					cnameExists = false
					cname_tmp = "Jobs_update_" + strconv.Itoa(num)
					for _, val := range cnames {
						if val == cname_tmp {
							cnameExists = true
						}
					}
					num++
				}
				// Get name for storing old Jobs collection
				cname_old := ""
				num = 1
				cnameExists = true
				for cnameExists {
					cnameExists = false
					cname_old = "Jobs_old_" + strconv.Itoa(num)
					for _, val := range cnames {
						if val == cname_old {
							cnameExists = true
						}
					}
					num++
				}

				// Create handle to new collection
				cNew := session.DB(conf.MONGODB_DATABASE).C(cname_tmp)
				iter := c.Find(bson.M{}).Iter()
				defer iter.Close()
				for iter.Next(jobDep) {
					job, err = core.JobDepToJob(jobDep)
					if err != nil {
						return err
					}
					err = cNew.Insert(&job)
					if err != nil {
						return err
					}
				}
				err = session.Run(bson.D{{"renameCollection", conf.MONGODB_DATABASE + ".Jobs"}, {"to", conf.MONGODB_DATABASE + "." + cname_old}}, bson.M{})
				if err != nil {
					return err
				}
				err = session.Run(bson.D{{"renameCollection", conf.MONGODB_DATABASE + "." + cname_tmp}, {"to", conf.MONGODB_DATABASE + ".Jobs"}}, bson.M{})
				if err != nil {
					return err
				}
			} else {
				fmt.Println("Exiting.")
				os.Exit(0)
			}
		}
	}
	return
}
