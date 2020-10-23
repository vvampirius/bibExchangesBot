package main

import (
	"github.com/vvampirius/mygolibs/belinvestbankExchange"
	"io"
	"io/ioutil"
	"log"
	"os"
)

func GetExchangeTest() {
	f, err := ioutil.TempFile(os.TempDir(), ``)
	if err != nil { log.Fatalln(err.Error()) }

	response, err := belinvestbankExchange.MakeRequest()
	if err != nil { os.Exit(1) }
	defer response.Body.Close()

	_, err = io.Copy(f, response.Body)
	if err != nil { log.Fatalln(err.Error()) }

	_, err = f.Seek(0, 0)
	if err != nil { log.Fatalln(err.Error()) }

	currencies, err := belinvestbankExchange.Get(f)
	if err != nil {
		log.Println(err.Error())
		log.Printf("You can check HTML in file: %s\n", f.Name())
		os.Exit(1)
	}

	usd, ok := currencies[`USD`]
	if !ok {
		log.Printf("USD currency not found in: %v\n", currencies)
		log.Printf("You can check HTML in file: %s\n", f.Name())
		os.Exit(1)
	}

	log.Println(usd)
	f.Close()
	os.Remove(f.Name())
}