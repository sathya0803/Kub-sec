package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/go-co-op/gocron"
	"github.com/vikiatinvisiblio/opa/pkg/opa"
)

var (
	inCluster = flag.Bool("incluster", false,
		"Does the exporter run within a K8S cluster, when true it will try to look for K8S service account details in the usual location.")
)

func task() {
	constraints, err := opa.GetConstraints(inCluster)
	if err != nil {
		log.Printf("err: %+v\n", err)
	}
	log.Println("listing the constraints violations...")
	jsondata, err := json.MarshalIndent(constraints, "", "")
	if err != nil {
		log.Printf("err: %+v\n", err)
	}
	log.Printf("data: %v", string(jsondata))
	sendRequest(string(jsondata))
	log.Print("request sent")

}
func main() {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(10).Minute().Do(task)
	scheduler.StartBlocking()
}
func sendRequest(data string) error {
	buff := bytes.NewBuffer([]byte(data))
	var apiendpoint string = os.Getenv("API_ENDPOINT")
	log.Println("sending to apiendpoint")
	req, err := http.NewRequest(http.MethodPost, apiendpoint, buff)
	if err != nil {
		log.Print("httprequesterror ", err)
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print("httpresponseerror", err)
		return err
	}
	log.Println(resp.Status)
	return nil
}
