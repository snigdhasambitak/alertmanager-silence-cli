package cmd

import (
	"encoding/json"
	"github.com/parnurzeal/gorequest"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Silence : containts Alertmanager silence data
type Silence struct {
	Status struct {
		State string `json:"state"`
	} `json:"status"`
	ID string `json:"id"`
	Comment string `json:"comment"`
	CreatedBy string `json:"createdBy"`
	UpdatedAt string `json:"updatedAt"`
	EndsAt string `json:"endsAt"`
	StartsAt string `json:"startsAt"`
	Matchers []Matcher `json:"matchers"`
}

// Matcher : contains Alertmanager matcher data
type Matcher struct {
	IsRegex bool `json:"isRegex"`
	Name string `json:"name"`
	Value string `json:"value"`
}

// Msg : contains Msg data
type Msg struct {
	data string
	err error
}

// ReceivedSilences : contains Alertmanager request silence data
type ReceivedSilences struct {
	Data []Silence `json:"data"`
	Status string `json:"status"`
}

const (
	silencesURL = "/api/v1/silences"
	silenceURL = "/api/v1/silence"
)

func httpPost(url, data string, result chan<- error) {
	request := gorequest.New()
	resp, _, errs := request.Post(url).Send(data).End()

	if errs != nil {
		var errsStr []string
		for _, e := range errs {
			errsStr = append(errsStr, fmt.Sprintf("%s", e))
		}
		result <- fmt.Errorf("%s", strings.Join(errsStr, "; "))
		return
	}

	if resp.StatusCode != 200 {
		result <- fmt.Errorf("HTTP response code: %s", resp.Status)
		return
	}
	result <- nil
}

func httpDelete(url string, result chan<- error) {
	request := gorequest.New()
	resp, _, errs := request.Delete(url).End()

	if errs != nil {
		var errsStr []string
		for _, e := range errs {
			errsStr = append(errsStr, fmt.Sprintf("%s", e))
		}
		result <- fmt.Errorf("%s", strings.Join(errsStr, "; "))
		return
	}

	if resp.StatusCode != 200 {
		result <- fmt.Errorf("HTTP response code: %s", resp.Status)
		return
	}
	result <- nil
}

func httpGetWithFilter(targetURL, filterData string, result chan<- Msg) {
	var msg Msg

	var URL *url.URL
	URL, err := url.Parse(targetURL)
	if err != nil {
		msg.err = fmt.Errorf("Cannot parse URL %s: %v", targetURL, err)
		result <- msg
		return
	}

	parameters := url.Values{}
	if filterData != "" {
		parameters.Add("filter", filterData)
	}
	URL.RawQuery = parameters.Encode()


	request := gorequest.New()
	resp, body, errs := request.Get(URL.String()).End()

	if errs != nil {
		var errsStr []string
		for _, e := range errs {
			errsStr = append(errsStr, fmt.Sprintf("%s", e))
		}
		msg.err = fmt.Errorf("%s", strings.Join(errsStr, "; "))
		result <- msg
		return
	}

	if resp.StatusCode != 200 {
		msg.err = fmt.Errorf("HTTP response code: %s", resp.Status)
		result <- msg
		return
	}

	msg.data = body
	result <- msg
}

func generateSilence(silencePeriod int, labels, creator, comment string) Silence {
	silence := Silence{
		StartsAt: time.Now().UTC().Format(time.RFC3339),
		EndsAt: time.Now().Add(time.Hour * time.Duration(silencePeriod)).UTC().Format(time.RFC3339),
		Comment: comment,
		CreatedBy: creator,
		UpdatedAt: "0001-01-01T00:00:00Z",
	}

	for k, v := range parseLabels(labels) {
		matcher := Matcher{
			IsRegex: false,
			Name: k,
			Value: v,
		}
		silence.Matchers = append(silence.Matchers, matcher)
	}

	return silence
}

func createSilence(amURL string, timeout, silencePeriod int, labels, creator, comment string) error {
	silence := generateSilence(silencePeriod, labels, creator, comment)

	fmt.Printf("Creating silence [creator: %s, comment: %s, start: %s, end: %s]\n", silence.CreatedBy, silence.Comment, silence.StartsAt, silence.EndsAt)

	json, err := json.Marshal(silence)
	if err != nil {
		return err
	}

	result := make(chan error)
	go httpPost(amURL + silencesURL, string(json), result)

	select {
	case err := <-result:
		return err
	case <-time.After(time.Second * time.Duration(timeout)):
		return errors.New("Alertmanager connections timeout")
	}
}

func genFilter(labels string) string {
	var filter []string
	for k, v := range parseLabels(labels) {
		filter = append(filter, k + "=" + v)
	}

	return strings.Join(filter, ",")
}

func querySilences(amURL string, timeout int, labels string) (string, error) {
	result := make(chan Msg)
	go httpGetWithFilter(amURL + silencesURL, genFilter(labels), result)


	var msg Msg
	select {
	case msg = <-result:
		if msg.err != nil {
			return "", msg.err
		}

	case <-time.After(time.Second * time.Duration(timeout)):
		return "", errors.New("Alertmanager connections timeout")
	}

	return msg.data, nil
}

func parseLabels(labelsStr string) map[string]string {
	results := make(map[string]string)

	labels := strings.Split(labelsStr, ",")
	for _, label := range labels {
		kv := strings.Split(label, "=")
		if len(kv) == 2 {
			results[kv[0]] = kv[1]
		}
	}

	return results
}

func filterActiveSilences(data string) ([]Silence, error) {
	silences, err := parseReceivedData(data)
	if err != nil {
		return []Silence{}, err
	}

	var results []Silence
	for _, silence := range silences.Data {
		if silence.Status.State == "active" {
			results = append(results, silence)
		}
	}

	return results, nil
}

func parseReceivedData(data string) (ReceivedSilences, error) {
	var silences ReceivedSilences

	err := json.Unmarshal([]byte(data), &silences)
	if err != nil {
		return ReceivedSilences{}, err
	}

	return silences, nil
}

func flattenLabels(silences []Silence) map[string]map[string]interface{} {
	flatLabels := make(map[string]map[string]interface{})

	for _, silence := range silences {
		// sort slice of structs by field name
		sort.Slice(silence.Matchers, func(i, j int) bool {
			return silence.Matchers[i].Name < silence.Matchers[j].Name
		})


		flatLabels[silence.ID] = make(map[string]interface{})
		var str string
		for _, matcher := range silence.Matchers {
			str += matcher.Name + matcher.Value
		}
		flatLabels[silence.ID]["flatLabels"] = str
		flatLabels[silence.ID]["silenceDefinition"] = silence
	}

	return flatLabels
}

func flattenInputLabels(labels string) string {
	var flatLabels string
	inputLabels := parseLabels(labels)

	var keys []string
	for k := range inputLabels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		flatLabels += k + inputLabels[k]
	}

	return flatLabels
}

func getSilencesFromAM(amURL string, timeout int, labels string) ([]Silence, error) {
	rawSilences, err := querySilences(amURL, timeout, labels)
	if err != nil {
		return []Silence{}, err
	}

	silences, err := filterActiveSilences(rawSilences)
	if err != nil {
		return []Silence{}, err
	}

	return silences, nil
}

func deleteSilences(amURL string, timeout int, labels string) error {
	silencesIDs, err := getSilenceIDToDelete(amURL, timeout, labels)
	if err != nil {
		return err
	}

	if len(silencesIDs) == 0 {
		fmt.Println("No silences to delete with given labels")
		return nil
	}

	results := make(chan error, len(silencesIDs))

	for silenceID, silence := range silencesIDs {
		fmt.Printf("Deleting silence [creator: %s, comment: %s, start: %s, end: %s]\n", silence.CreatedBy, silence.Comment, silence.StartsAt, silence.EndsAt)
		go httpDelete(amURL + silenceURL + "/" + silenceID, results)
	}

	var errsStr []string
	for i := 0; i < len(silencesIDs); i++ {
		select {
		case err := <-results:
			if err != nil {
				errsStr = append(errsStr, fmt.Sprintf("%s", err))
			}
		case <-time.After(time.Second * time.Duration(timeout)):
			errsStr = append(errsStr, "Alertmanager connections timeout")
		}
	}

	if len(errsStr) > 0 {
		return fmt.Errorf("%s", strings.Join(errsStr, "; "))
	}
	return nil
}

func getSilenceIDToDelete(amURL string, timeout int, labels string) (map[string]Silence, error) {
	silences, err := getSilencesFromAM(amURL, timeout, labels)
	if err != nil {
		return map[string]Silence{}, err
	}

	flatInputLabels := flattenInputLabels(labels)

	results := make(map[string]Silence)
	for k, v := range flattenLabels(silences) {
		if v["flatLabels"] == flatInputLabels {
			results[k] = v["silenceDefinition"].(Silence)
		}
	}

	return results, nil

}

func printSilences(amURL string, timeout int, labels string) error {
	silences, err := getSilencesFromAM(amURL, timeout, labels)
	if err != nil {
		return err
	}

	for _, silence := range silences {
		var matchers []string
		for _, matcher := range silence.Matchers {
			matchers = append(matchers, matcher.Name + "=" + matcher.Value)
		}
		fmt.Printf("ID: %s, creator: %s, comment: %s, start: %s, end: %s, labels: %s\n",
			silence.ID,
			silence.CreatedBy,
			silence.Comment,
			silence.StartsAt,
			silence.EndsAt,
			strings.Join(matchers, ","),
		)
	}

	return nil
}

func Run(amURL string, timeout int, mode, labels string, silencePeriod int, creator, comment string) error {
	switch mode {
	case "create":
		if labels == "" {
			return errors.New("Parameter labels cannot be empty in mode: create")
		}
		if err := createSilence(amURL, timeout, silencePeriod, labels, creator, comment); err != nil {
			return err
		}
	case "delete":
		if labels == "" {
			return errors.New("Parameter labels cannot be empty in mode: delete")
		}
		if err := deleteSilences(amURL, timeout, labels); err != nil {
			return err
		}
	case "show":
		printSilences(amURL, timeout, labels)
	default:
		return errors.New("Unrecognized mode parameter")
	}

	return nil
}