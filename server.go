package main

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
)

const ResourceRecordSetTypeA = "A"

type Server struct {
	config             map[string]Config
	zoneName           string
	projectName        string
	serviceAccountFile string
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

func (server *Server) getDnsRecord(dnsService *dns.Service, domain string) (*dns.ResourceRecordSetsListResponse, error) {
	log.Printf("Project: %s, Zone: %s. Domain: %s", server.projectName, server.zoneName, domain)
	record, err := dnsService.ResourceRecordSets.List(server.projectName, server.zoneName).Name(domain).Type(ResourceRecordSetTypeA).Do()

	if err != nil {
		return nil, fmt.Errorf("Issues find record %s: %s", domain, err)
	}

	return record, nil
}

func (server *Server) updateDnsRecord(dnsService *dns.Service, record *dns.ResourceRecordSet, newIPAddress string) error {
	change := &dns.Change{
		Deletions: []*dns.ResourceRecordSet{record},
		Additions: []*dns.ResourceRecordSet{
			{
				Name:    record.Name,
				Type:    record.Type,
				Ttl:     record.Ttl,
				Rrdatas: []string{newIPAddress},
			},
		},
	}

	log.Printf("Updating A record for %s - from: %v to: %v\n", record.Name, change.Deletions[0].Rrdatas, change.Additions[0].Rrdatas)

	_, err := dnsService.Changes.Create(server.projectName, server.zoneName, change).Do()
	if err != nil {
		return err
	}
	return nil
}
func (server *Server) manageUpdateDNSRecord(domain string, NewIPAddress string) error {
	log.Printf("Updating %s with new IP address %s.\n", domain, NewIPAddress)
	ctx := context.Background()
	dnsService, err := dns.NewService(ctx, option.WithCredentialsFile(server.serviceAccountFile))
	if err != nil {
		fmt.Errorf("Cannot get DNS Service: %s", err)
	}
	record, err := server.getDnsRecord(dnsService, domain)
	if err != nil {
		fmt.Errorf("Cannot get record: %s", err)
	}
	if record == nil {
		fmt.Errorf("record is nil :(")
	}
	// s, _ := record.MarshalJSON()
	log.Printf("Record: %s.", record)

	server.updateDnsRecord(dnsService, record.Rrsets[0], NewIPAddress)
	if err != nil {
		fmt.Errorf("Error updating DNS A record: %s", err)
	}
	log.Println("Updated.")
	return nil
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

func (server *Server) GetRemoteIPAddress(request *http.Request) string {
	fwdAddress := request.Header.Get("X-Forwarded-For")
	if fwdAddress != "" {
		log.Printf("Using forwarded address: %s.", fwdAddress)
		return fwdAddress
	}
	return strings.Split(request.RemoteAddr, ":")[0]
}

func (server *Server) manage_update_dns_a_record(writer http.ResponseWriter, request *http.Request, domain string) {
	ip_address := server.GetRemoteIPAddress(request)
	if server.is_dns_record_out_of_date(ip_address, domain) {
		log.Printf("Updating A record for %s with IP %s\n", domain, ip_address)
		fmt.Fprintf(writer, "Updating A record…\n")
		server.manageUpdateDNSRecord(domain, ip_address)
	} else {
		log.Printf("Already up to date A record for %s with IP %s.\n", domain, ip_address)
		fmt.Fprintf(writer, "A record up to date.\n")
	}
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
	server.zoneName = "[INSERT HERE PROJECT]"
	server.projectName = "[INSERT HERE ZONE]"
	server.serviceAccountFile = "api-key.json"
	http.HandleFunc("/dns_updater", server.dns_updater)

	log.Println("Listening for incomming connections…")
	http.ListenAndServe(":8090", nil)
}
