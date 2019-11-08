package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

var url = ""
var version = "1.0.0"
var app = cli.NewApp()

func info() {
	app.Name = "json-to-hugo"
	app.Usage = "Convert json content to hugo format"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Jan Kenith Johannessen",
			Email: "jankenith.johannessen@gmail.com",
		},
		cli.Author{
			Name:  "Richard Burk Orofeo",
			Email: "rborofeo@gmail.com",
		},
	}
	app.Version = version
}

func commands() {
	// Default Command
	app.Action = func(c *cli.Context) error {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
		serverURL := os.Getenv("SERVER_URL")
		log.Printf("ServerURL: %v", serverURL)
		if serverURL == "" {
			log.Fatal("SERVER_URL not set")
		}
		url = strings.TrimRight(serverURL, "/")

		t := Setting{}
		yamlFile, err := ioutil.ReadFile("strapi-settings.yaml")
		if err != nil {
			log.Printf("yamlFile.Get err #%v ", err)
			log.Print("\n-------------------------------------\n   YAML settings file error...\n-------------------------------------\n")
		}

		log.Print("\n-------------------------------------\n   Pulling Data from JSON Source...\n-------------------------------------\n")
		err = yaml.Unmarshal(yamlFile, &t)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		var wg sync.WaitGroup
		sliceLength := len(t.Tables)
		wg.Add(sliceLength)
		if sliceLength > 0 {
			for i := 0; i < sliceLength; i++ {
				go func(table Table) {
					defer wg.Done()
					err = getContentType(-1, 0, table)
					if err != nil {
						log.Printf("Error: %v", err)
					}
				}(t.Tables[i])
			}
		}
		wg.Wait()
		log.Print("\n-------------------------------------\n")
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Show version",
			Action: func(c *cli.Context) {
				fmt.Println(version)
			},
		},
	}
}

func main() {
	info()
	commands()

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

/*
	[WIP] Params
	limit - limit of items to pull (-1 for no limit)
	skip - for paginate?
*/
func getContentType(limit int, skip int, table Table) error {
	refURL := fmt.Sprintf("%s/%s", url, table.ID)

	req, err := http.NewRequest("GET", refURL, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Print(fmt.Sprintf("   %v - %v %v", table.ID, "StatusCode:", res.StatusCode))
		return nil
	}
	m := []map[string]interface{}{}
	err = json.NewDecoder(res.Body).Decode(&m)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range m {
		d, err := yaml.Marshal(v)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fileName := fmt.Sprintf("%v.md", v["id"])
		filePath := table.Directory + fileName
		os.MkdirAll(table.Directory, os.ModePerm)
		file, err := os.Create(filePath)
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
		}
		defer file.Close()

		fmt.Fprintln(file, "---")
		fmt.Fprint(file, string(d))
		fmt.Fprintln(file, "---")
	}
	ctr := "items"
	if len(m) == 1 {
		ctr = "item"
	}
	log.Print(fmt.Sprintf("   %v - %v %v", table.ID, len(m), ctr))
	return nil
}

// Setting ...
type Setting struct {
	Tables []Table `yaml:"tables"`
}

// Table ...
type Table struct {
	ID            string `yaml:"id"`
	Directory     string `yaml:"directory"`
	FileExtension string `yaml:"fileExtension"`
	MainContent   string `yaml:"mainContent"`
}
