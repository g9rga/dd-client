package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"
)

const (
	DefaultTimeout       = time.Second * 10
	TaskTypeHping3       = "hping3"
	TaskTypeSlowHttpTest = "slowhttptest"
)

var BaseUrl = os.Getenv("DD_API_URL")

type DDClient struct {
	baseUrl string
	client  *http.Client
}

type Task struct {
	Id   string   `json:"id"`
	Type string   `json:"type"`
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}

func (cl *DDClient) Register(clientId string) (string, error) {
	jsonData := map[string]interface{}{
		"id":       clientId,
		"cpuCount": runtime.NumCPU(),
	}
	jsonEncoded, err := json.Marshal(jsonData)
	if err != nil {
		return "", err
	}
	resp, err := http.Post(cl.baseUrl+"/api/registrations/"+clientId, "application/ld+json", bytes.NewBuffer(jsonEncoded))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("unexpected status")
	}
	decodedResponse := struct {
		AccessToken string `json:"accessToken"`
	}{}
	err = json.Unmarshal(body, &decodedResponse)
	if err != nil {
		return "", err
	}

	return decodedResponse.AccessToken, nil
}

func (cl *DDClient) GetTasks(accessToken string, activeTasks []string) (map[string]Task, error) {
	u, _ := url.Parse(cl.baseUrl + "/api/tasks")
	b, _ := json.Marshal(map[string]interface{}{
		"activeTasks": activeTasks,
	})
	r := &http.Request{
		Method: http.MethodGet,
		URL:    u,
		Header: map[string][]string{
			"Authorization": {"Bearer " + accessToken},
			"Content-type":  {"application/json"},
		},
		Body: ioutil.NopCloser(bytes.NewBuffer(b)),
	}
	resp, err := cl.client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected status")
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	tasks := struct {
		Tasks []Task `json:"hydra:member"`
	}{}
	err = json.Unmarshal(body, &tasks)
	if err != nil {
		return nil, err
	}
	result := make(map[string]Task)
	for _, val := range tasks.Tasks {
		result[val.Id] = val
	}
	return result, nil
}

func CreateDDClient() DDClient {
	tr := &http.Transport{
		DisableKeepAlives:     true,
		MaxConnsPerHost:       1,
		ResponseHeaderTimeout: DefaultTimeout,
	}
	c := &http.Client{
		Transport:     tr,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       DefaultTimeout,
	}
	return DDClient{
		client:  c,
		baseUrl: BaseUrl,
	}
}
