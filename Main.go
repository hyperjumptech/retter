package main

import (
	"fmt"
	"log"
	"net/http"
)

func splash() {
	fmt.Println("________________________________________________________________ ")
	fmt.Println(`\______   \_   _____/\__    ___/\__    ___/\_   _____/\______   \`)
	fmt.Println(" |       _/|    __)_   |    |     |    |    |    __)_  |       _/")
	fmt.Println(` |    |   \|        \  |    |     |    |    |        \ |    |   \`)
	fmt.Println(" |____|_  /_______  /  |____|     |____|   /_______  / |____|_  /")
	fmt.Println(`       \/        \/                               \/         \/ `)
}

func main() {
	splash()
	listen := ":80"
	fmt.Printf("Retter is listening on : %s\n", listen)
	log.Fatal(http.ListenAndServe(listen, NewRetterHttpHandler("http://localhost:8088")))
}
