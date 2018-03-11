package main

import (
	"io/ioutil"
	"encoding/json"
	"fmt"
	"context"
	"net/http"
	"strings"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"strconv"
	"time"
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

func callSalatAPI(city string, country string, method int, timezone string, today time.Time) Timing {
	ddmmyyyy := fmt.Sprintf("%02d-%02d-%d", today.Day(), today.Month(), today.Year())

	resp, err := http.Get(fmt.Sprintf("http://api.aladhan.com/v1/timingsByCity?city=%s&country=%s"+
		"&method=%stimezonestring=%s&date_or_timestamp=%s",
		city,
		country,
		method,
		timezone,
		ddmmyyyy))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var result Result
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		panic(err)
	}
	return result.Data.Timing
}

func setReminder(timing string, salatTimeUtc time.Time, minBefore int) {
	reminderTimeUtc := salatTimeUtc.Add(time.Duration(-minBefore) * time.Minute)

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create the cloudwatch events client
	svc := cloudwatchevents.New(sess)
	puteventresult, puteventerr := svc.PutRule(&cloudwatchevents.PutRuleInput{
		Name: aws.String("SalatTimeJKT" + timing),
		ScheduleExpression: aws.String(fmt.Sprintf("cron(%d %d %d %d ? %d)",
			reminderTimeUtc.Minute(),
			reminderTimeUtc.Hour(),
			reminderTimeUtc.Day(),
			reminderTimeUtc.Month(),
			reminderTimeUtc.Year())),
	})
	if puteventerr != nil {
		fmt.Println("Error", puteventerr)
	}
	fmt.Println("Rule ARN:", puteventresult.RuleArn)

	puttargetresult, puttargeterr := svc.PutTargets(&cloudwatchevents.PutTargetsInput{
		Rule: aws.String("SalatTimeJKT" + timing),
		Targets: []*cloudwatchevents.Target{
			&cloudwatchevents.Target{
				Arn: aws.String("<your lambda arn here>"),
				Id:  aws.String("<your cloud watch event id here>"),
			},
		},
	})
	if puttargeterr != nil {
		fmt.Println("Error", puttargeterr)
	}
	fmt.Println("Success", puttargetresult)
}

func timingToTimeUTC(timingStr string, today time.Time, timeLoc *time.Location) time.Time {
	s := strings.Split(timingStr, ":")
	hrStr, minStr := s[0], s[1]
	hrInt, err := strconv.Atoi(hrStr)
	if err != nil {
		panic(err)
	}
	minInt, err := strconv.Atoi(minStr)
	if err != nil {
		panic(err)
	}

	salatTimeLocal := time.Date(
		today.Year(), today.Month(), today.Day(), hrInt, minInt, 0, 0, timeLoc)
	salatTimeUtc := salatTimeLocal.In(time.UTC)
	return salatTimeUtc
}

func sendDailyScheduleToSlack(timing Timing){
	resp, err := http.Get("https://hooks.zapier.com/yourwebhookhere/?" +
		"Fajr=" + timing.Fajr + "&" +
		"Dhuhr=" + timing.Dhuhr + "&" +
		"Asr=" + timing.Asr + "&" +
		"Maghrib=" + timing.Maghrib + "&" +
		"Isha=" + timing.Isha)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(bodyBytes))
}

func run() {
	timeZoneStr := "Asia/Jakarta"
	city := "Jakarta"
	country := "Indonesia"
	method := 1 //Muslim World League
	timeJKTLoc, err := time.LoadLocation(timeZoneStr)
	if err != nil {
		panic(err)
	}
	nowGmt7 := time.Now().In(timeJKTLoc)

	timing := callSalatAPI(city, country, method, timeZoneStr, nowGmt7)
	minBefore := 10 //Remind 10 minute before adzan
	setReminder("Fajr", timingToTimeUTC(timing.Fajr, nowGmt7, timeJKTLoc), minBefore)
	setReminder("Dhuhr", timingToTimeUTC(timing.Dhuhr, nowGmt7, timeJKTLoc), minBefore)
	setReminder("Asr", timingToTimeUTC(timing.Asr, nowGmt7, timeJKTLoc), minBefore)
	setReminder("Maghrib", timingToTimeUTC(timing.Maghrib, nowGmt7, timeJKTLoc), minBefore)
	setReminder("Isha", timingToTimeUTC(timing.Isha, nowGmt7, timeJKTLoc), minBefore)
	sendDailyScheduleToSlack(timing)
}

type MyEvent struct {
	Name string `json:"name"`
}

func HandleRequest(ctx context.Context, name MyEvent) (string, error) {
	run()
	return fmt.Sprintf("Hello %s!", name.Name), nil
}

func main() {
	lambda.Start(HandleRequest)
}
