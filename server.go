package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Server struct {
	config map[string]Config
}

type Config struct {
	Domain string `json:"domain"`
	Secret string `json:"secret"`
}

func (server *Server) calculate_sha1(secret string) string {
	sha1_object := sha1.New()
	sha1_object.Write([]byte(secret))
	// echo -n $secret|sha1sum
	sha1_hex := sha1_object.Sum(nil)
	return fmt.Sprintf("%x", sha1_hex)
}

func (server *Server) load_configuration(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Error opening config file!: ", err)
	}
	err = json.Unmarshal(data, &server.config)
	if err != nil {
		log.Fatal("Error unmarshalling config:", err)
	}
}

func (server *Server) manage_secret(writer http.ResponseWriter, secret_values []string, domain_values []string) {
	if len(secret_values) == 1 && len(domain_values) == 1 {
		log.Println("Secret is", secret_values[0])
		log.Println("Expected secret for domain is", server.config[domain_values[0]].Secret)
		secret := secret_values[0]
		sha1_string := server.calculate_sha1(secret)
		if sha1_string == server.config[domain_values[0]].Secret {
			log.Printf("Secrets match for domain: %s.\n", domain_values[0])
			fmt.Fprintf(writer, "Authenticated\n")
		} else {
			log.Printf("Secrets do not match for domain: %s.\n", domain_values[0])
			http.Error(writer, "Secret does not match!", http.StatusForbidden)
		}
	} else {
		log.Println("Secret not properly set!")
		http.Error(writer, "Secret not properly set!!", http.StatusForbidden)
		return
	}
}

func (server *Server) dns_updater(writer http.ResponseWriter, request *http.Request) {
	query_keys := request.URL.Query()
	if secret_values, ok := query_keys["secret"]; ok {
		if domain_values, ok := query_keys["domain"]; ok {
			log.Printf("There is a secret and domain on query: %s - %s", secret_values, domain_values)
			server.manage_secret(writer, secret_values, domain_values)
		}
	}
}

func main() {
	server := Server{}
	server.load_configuration("config.json")
	http.HandleFunc("/dns_updater", server.dns_updater)

	http.ListenAndServe(":8090", nil)
}
