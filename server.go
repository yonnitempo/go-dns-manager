package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
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

func (server *Server) manage_update_dns_a_record(writer http.ResponseWriter, request *http.Request, domain string) {
	ip_address := strings.Split(request.RemoteAddr, ":")[0]
	if server.is_dns_record_out_of_date(ip_address, domain) {
		log.Printf("Updating A record for %s with IP %s\n", domain, ip_address)
		fmt.Fprintf(writer, "Updating A record…\n")
		server.update_a_record_for_domain(domain, ip_address)
	} else {
		log.Printf("Already up to date A record for %s with IP %s.\n", domain, ip_address)
		fmt.Fprintf(writer, "A record up to date.\n")
	}
}

func (server *Server) update_a_record_for_domain(domain string, ipaddr string) {
	log.Println("Updating DNS Record on Google Cloud…")
	time.Sleep(3 * time.Second)
	log.Println("Updated DNS Record on Google Cloud.")
}

func (server *Server) is_dns_record_out_of_date(ip_address string, domain string) bool {
	ips, err := net.LookupIP(domain)
	if err != nil {
		log.Printf("Could not get IPs for domain %s: %v\n", err, domain)
	}
	log.Printf("IPs for %s: %s.", domain, ips)
	if len(ips) == 1 {
		if ips[0].String() == ip_address {
			return false
		}
		// All this is assuming there is only one IPv4.
	}
	return true
}

func (server *Server) manage_secret(writer http.ResponseWriter, secret_values []string, domain_values []string) (string, bool) {
	if len(secret_values) == 1 && len(domain_values) == 1 {
		log.Println("Secret is", secret_values[0])
		log.Println("Expected secret for domain is", server.config[domain_values[0]].Secret)
		secret := secret_values[0]
		sha1_string := server.calculate_sha1(secret)
		if sha1_string == server.config[domain_values[0]].Secret {
			log.Printf("Secrets match for domain: %s.\n", domain_values[0])
			fmt.Fprintf(writer, "Authenticated\n")
			return domain_values[0], true
		} else {
			log.Printf("Secrets do not match for domain: %s.\n", domain_values[0])
			http.Error(writer, "Secret does not match!", http.StatusForbidden)
			return "", false
		}
	} else {
		log.Println("Secret not properly set!")
		http.Error(writer, "Secret not properly set!!", http.StatusForbidden)
		return "", false
	}
}

func (server *Server) dns_updater(writer http.ResponseWriter, request *http.Request) {
	query_keys := request.URL.Query()
	if secret_values, ok := query_keys["secret"]; ok {
		if domain_values, ok := query_keys["domain"]; ok {
			log.Printf("There is a secret and domain on query: %s - %s", secret_values, domain_values)
			if domain, ok := server.manage_secret(writer, secret_values, domain_values); ok {
				server.manage_update_dns_a_record(writer, request, domain)
			}
		}
	}
	fmt.Fprintf(writer, "Request completed!\n")
}

func main() {
	server := Server{}
	server.load_configuration("config.json")
	http.HandleFunc("/dns_updater", server.dns_updater)

	log.Println("Listening for incomming connections…")
	http.ListenAndServe(":8090", nil)
}
