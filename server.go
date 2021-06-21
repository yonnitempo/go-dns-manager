package main

import (
	"crypto/sha1"
	"fmt"
	"log"
	"net/http"
)

func hello(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "hello\n")
}

func headers(writer http.ResponseWriter, request *http.Request) {
	for name, headers := range request.Header {
		for _, h := range headers {
			fmt.Fprintf(writer, "%v: %v\n", name, h)
		}
	}
}

func manage_secret(writer http.ResponseWriter, secret_values []string) {
	if len(secret_values) == 1 {
		log.Println("Secret is", secret_values[0])
		secret := secret_values[0]
		sha1_object := sha1.New()
		sha1_object.Write([]byte(secret))
		sha1_hex := sha1_object.Sum(nil)
		fmt.Fprintf(writer, "SHA1: %x\n", sha1_hex)
	} else {
		log.Println("Secret has no value!")
		http.Error(writer, "Secret empty!!", http.StatusForbidden)
		return
	}
}

func dns_updater(writer http.ResponseWriter, request *http.Request) {
	query_keys := request.URL.Query()
	if value, ok := query_keys["secret"]; ok {
		log.Println("There is a secret on query. Value:", value)
		manage_secret(writer, value)
	}
}

func main() {

	http.HandleFunc("/hello", hello)
	http.HandleFunc("/headers", headers)
	http.HandleFunc("/dns_updater", dns_updater)

	http.ListenAndServe(":8090", nil)
}
