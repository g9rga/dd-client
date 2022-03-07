package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/g9rga/dd-client/internal"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	PollingInterval       = 10
	ExitTimeout           = time.Second * 5
	ObtainTokenTriesCount = 3
	ObtainTokenInterval   = time.Second * 3
	AccessTokenExpGap     = 60
)

var (
	clientId            string
	ddClient            internal.DDClient
	obtainTokenTryCount int8
	supportedTypes      map[string]interface{}
)

func main() {
	supportedTypes = make(map[string]interface{})
	if os.Getenv("SUPPORTED_TYPES") != "" {
		pieces := strings.Split(os.Getenv("SUPPORTED_TYPES"), ",")
		for _, taskType := range pieces {
			supportedTypes[taskType] = nil
		}
	}
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logrus.Info("Starting dd-client")
	ddClient = internal.CreateDDClient()
	ctx, cancel := context.WithCancel(context.TODO())
	cmdPool := internal.CreateCommandPool()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	var accessToken string
	var err error
	var lastCheck int64
	for {
		select {
		case <-quit:
			logrus.Info("Exiting")
			cancel()
			time.Sleep(ExitTimeout)
			return
		case <-ctx.Done():
			logrus.Info("Polling done")
			return
		default:
			if accessToken == "" || isExpiredToken(accessToken) {
				accessToken, err = obtainAccessToken()
				if err != nil {
					logrus.Error("Client registering failed, please check your internet connect and restart client")
					cancel()
					time.Sleep(ExitTimeout)
					return
				}
			}
			if (lastCheck + PollingInterval) < time.Now().Unix() {
				runningCommands := cmdPool.GetCommands()
				var activeTasks []string
				for key := range runningCommands {
					activeTasks = append(activeTasks, key)
				}
				tasks, err := ddClient.GetTasks(accessToken, activeTasks)
				if err != nil {
					logrus.Error(err)
					continue
				}
				lastCheck = time.Now().Unix()
				for cid := range runningCommands {
					if _, ok := tasks[cid]; !ok {
						logrus.Info("stopping command: ", cid)
						err = cmdPool.StopCommand(cid)
						if err != nil {
							logrus.Error("command stop failed: ", err)
						}
					}
				}
				for taskId, task := range tasks {
					if _, ok := runningCommands[taskId]; ok {
						continue
					}
					if _, ok := supportedTypes[task.Type]; !ok && len(supportedTypes) > 0 {
						continue
					}
					logrus.Info("new task received: ", task.Cmd, " ", strings.Join(task.Args, " "))
					go func(task internal.Task) {
						err = cmdPool.RunCommand(ctx, task.Id, task.Cmd, task.Args)
						if err != nil {
							logrus.Error("command start failed: ", err)
						}
					}(task)
				}
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func obtainAccessToken() (string, error) {
	if clientId == "" {
		clientId = uuid.NewString()
	}
	accessToken, err := ddClient.Register(clientId)
	if err != nil {
		if obtainTokenTryCount == ObtainTokenTriesCount {
			return "", errors.New("max errors count reached")
		}
		logrus.Error("Fail to register client, retrying")
		obtainTokenTryCount = obtainTokenTryCount + 1
		time.Sleep(ObtainTokenInterval)
		return obtainAccessToken()
	} else {
		obtainTokenTryCount = 0
		return accessToken, nil
	}
}

func isExpiredToken(accessToken string) bool {
	pieces := strings.Split(accessToken, ".")
	if len(pieces) != 3 {
		panic("Wrong access token format")
	}

	payload, err := base64.StdEncoding.DecodeString(pieces[1])
	if err != nil {
		panic("Access token decoding failed")
	}
	payloadStr := struct {
		Exp int64 `json:"exp"`
	}{}
	err = json.Unmarshal(payload, &payloadStr)
	if err != nil {
		panic("Access token decoding failed")
	}
	return (payloadStr.Exp - AccessTokenExpGap) < time.Now().Unix()
}
