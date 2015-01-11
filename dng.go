// Package main contains the HTTP server code and driver.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/garfunkel/dng/scraper"
	"github.com/garfunkel/dng/settings"
	"log"
	"net/http"
)

// index returns the index/home page.
func index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}

// getAddresses returns a JSON encoded list of addresses in the DB.
func getAddresses(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(settings.Settings.Addresses)

	if err != nil {
		log.Fatal(err)
	}

	w.Write(data)
}

// getAddressInfo returns information on a particular address.
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

// getStatic serves static files.
func getStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "."+r.URL.Path)
}

// refreshInfo refreshes info for an address.
func refreshInfo(address string) {
	scrape, scraped, err := scraper.New(address)

	if err != nil {
		log.Println(err)
	} else if !scraped {
		if err = scrape.Scrape(); err != nil {
			log.Println(err)
		}
	}
}

// saveNotes saves user submitted notes for an address.
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

// main is the driver function.
func main() {
	refresh := flag.Bool("r", false, "refresh real estate information")
	debug := flag.Bool("d", false, "debug mode")

	flag.Parse()

	if *debug {
		log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
	}

	if err := settings.ReadSettings(); err != nil {
		log.Fatal(err)
	}

	if *refresh {
		log.Printf("refreshing all real estate information")

		c := make(chan bool)

		defer close(c)

		for addressIndex, address := range settings.Settings.Addresses {
			go func (addressIndex int, address string) {
				log.Printf("refreshing %v", address)

				refreshInfo(address)

				log.Printf("refreshed %v", address)

				c <- true
			}(addressIndex, address)
		}

		for range settings.Settings.Addresses {
			<- c
		}

		log.Printf("refreshed all real estate information")
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/addresses", getAddresses)
	http.HandleFunc("/addressinfo", getAddressInfo)
	http.HandleFunc("/savenotes", saveNotes)
	http.HandleFunc("/static/", getStatic)

	hostString := fmt.Sprintf("%v:%v", settings.Settings.Host, settings.Settings.Port)

	log.Printf("listening on http://%v", hostString)

	log.Fatal(http.ListenAndServe(hostString, nil))
}
