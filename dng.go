package main

import (
	"net/http"
	"log"
	"encoding/json"
	"github.com/garfunkel/dng/settings"
	"github.com/garfunkel/dng/scraper"
	"flag"
	"fmt"
)

const (
	Host = "localhost"
	Port = "8080"
)

func index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}

func getAddresses(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(settings.Settings.Addresses)

	if err != nil {
		log.Fatal(err)
	}

	w.Write(data)
}

func getAddressInfo(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()

	if err != nil {
		log.Fatal(err)
	}

	addressValue, ok := r.Form["address"]

	if !ok {
		log.Fatal("no address in querystring")
	}

	address := addressValue[0]
	scrape, _, err := scraper.New(address)

	if err != nil {
		log.Println(err)
	}

	data, err := json.Marshal(scrape)

	w.Write(data)
}

func getStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "." + r.URL.Path)
}

func refreshInfo(c chan string, address string) {
	scrape, scraped, err := scraper.New(address)

	if err != nil {
		log.Println(err)
	}

	if !scraped {
		err = scrape.Scrape()

		if err != nil {
			log.Println(err)
		}
	}

	c <- address
}

func saveNotes(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	addresses, ok := r.PostForm["address"]

	if !ok || len(addresses) == 0 {
		log.Printf("could not update notes as no address could be parsed")

		return
	}

	address := addresses[0]
	notes, ok := r.PostForm["notes"]

	if !ok || len(notes) == 0 {
		log.Printf("could not update notes for %v - no notes could be parsed", address)
	}

	scrape, _, err := scraper.New(address)

	if err != nil {
		log.Println(err)
	}

	scrape.Notes = notes[0]

	if err = scrape.Save(); err != nil {
		log.Println(err)
	}
}

func main() {
	refresh := flag.Bool("r", false, "refresh real estate information")

	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)

	if err := settings.ReadSettings(); err != nil {
		log.Fatal(err)
	}

	if *refresh {
		log.Printf("refreshing real estate information")

		c := make(chan string)

		for _, address := range settings.Settings.Addresses {
			log.Printf("refreshing %v", address)

			go refreshInfo(c, address)
		}

		num := 0

		for address := range c {
			num += 1

			log.Printf("refreshed %v", address)

			if num == len(settings.Settings.Addresses) {
				break
			}
		}

		close(c)

		log.Printf("refreshed real estate information")
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/addresses", getAddresses)
	http.HandleFunc("/addressinfo", getAddressInfo)
	http.HandleFunc("/savenotes", saveNotes)
	http.HandleFunc("/static/", getStatic)

	hostString := fmt.Sprintf("%v:%v", Host, Port)

	log.Printf("listening on http://%v", hostString)

	log.Fatal(http.ListenAndServe(hostString, nil))
}
