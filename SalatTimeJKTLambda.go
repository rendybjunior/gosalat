package main

import (
	"fmt"
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"net/http"
	"io/ioutil"
)

type Result struct {
	Data Data `json:"data"`
}

type Data struct {
	Timing Timing `json:"timings"`
}

type Timing struct {
	Fajr    string `json:"Fajr"`
	Dhuhr   string `json:"Dhuhr"`
	Asr     string `json:"Asr"`
	Maghrib string `json:"Maghrib"`
	Isha    string `json:"Isha"`
}

func sendToSlack(timing string, hour int, min int) {
	resp, err := http.Get(fmt.Sprintf("https://hooks.zapier.com/something/something/?" +
		"timing=%s&hhmm=%02d:%02d",
		timing,
		hour,
		min))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(bodyBytes))
}

type SalatTimeParam struct {
	Timing string `json:"timing"`
	Hour   int    `json:"hh"`
	Minute int    `json:"mm"`
}

func HandleRequest(ctx context.Context, param SalatTimeParam) (string, error) {
	sendToSlack(param.Timing, param.Hour, param.Minute)
	return fmt.Sprintf("Hello %s!", param.Timing), nil
}

func main() {
	lambda.Start(HandleRequest)
}
