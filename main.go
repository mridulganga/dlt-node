package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
	"github.com/mridulganga/dlt-node/pkg/constants"
	mqttlib "github.com/mridulganga/dlt-node/pkg/mqttlib"
	plug "github.com/mridulganga/dlt-node/pkg/plugin"
	"github.com/mridulganga/dlt-node/pkg/util"
	log "github.com/sirupsen/logrus"
)

type Data map[string]any

const (
	action_node_update      = "node_update"
	action_add_node         = "add_node"
	action_add_node_success = "add_node_success"
	action_remove_node      = "remove_node"
	action_start_loadtest   = "start_loadtest"
	action_stop_loadtest    = "stop_loadtest"

	node_healthy = "healthy"
	pluginName   = "loadTest"
)

var (
	nodegroupId    string
	nodeId         string
	mqttHost       string
	mqttPort       int
	nodegroupTopic string
	nodeTopic      string

	loadTestId         string
	loadTestResults    = []string{}
	loadTestResultLock = sync.Mutex{}

	isTestActive bool
)

func sendNodeStatus(m mqttlib.MqttClient, ngTopic string) {
	loadTestResultLock.Lock()
	defer loadTestResultLock.Unlock()

	payload := Data{
		"action":       action_node_update,
		"timestamp":    fmt.Sprintf("%v", time.Now().Unix()),
		"node_id":      nodeId,
		"node_status":  node_healthy,
		"isTestActive": isTestActive,
	}

	if len(loadTestResults) > 0 {
		resultBatchBytes, _ := json.Marshal(loadTestResults)
		encodedResults := base64.StdEncoding.EncodeToString(resultBatchBytes)
		payload["load_test_id"] = loadTestId
		payload["load_test_results"] = string(encodedResults)
		loadTestResults = []string{}
	}

	// log.Infof("Sending NodeStatus %v", payload)
	m.Publish(ngTopic, payload)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Error("Error loading .env file")
	}

	nodegroupId = os.Getenv(constants.NODEGROUP_ID)
	nodeId = os.Getenv(constants.NODE_ID)
	nodegroupTopic = fmt.Sprintf("ngs/%s", nodegroupId)
	nodeTopic = fmt.Sprintf("nodes/%s", nodeId)
	mqttHost = os.Getenv(constants.MQTT_HOST)
	mqttPort, _ = strconv.Atoi(os.Getenv(constants.MQTT_PORT))

	healthCheckStopper := make(chan bool)
	quitLoadTestChannel := make(chan bool)

	/*
		start mqtt
		publish message to nodegroup to add node (message contains node_id)
		when we get ack in the nodes/node_id topic
				start sending health every 30 secs to nodegroup
				if ongoing load test - then send metrics from kv db
		listen for commands on nodes/topic
		if command = ExecuteLoadTest then load plugin and config(tps, duration etc)
				trigger Run for duration
				keep updating local kv db with success/failure stats
				when duration completes, send load results
		if command = StopLoadTest then terminate all the running threads and send results
	*/

	// start mqtt
	m := mqttlib.NewMqtt(mqttHost, mqttPort)
	go m.Connect()
	m.WaitUntilConnected()

	go util.CallPeriodic(time.Second*5, func() {
		sendNodeStatus(m, nodegroupTopic)
	}, healthCheckStopper)

	// start listening to the messages sent to the node topic
	m.Sub(nodeTopic, func(client mqtt.Client, message mqtt.Message) {
		data := Data{}
		json.Unmarshal(message.Payload(), &data)
		log.Infof("received: %v", data)

		// perform operations based on action
		switch data["action"] {
		case action_start_loadtest:
			if !isTestActive {
				isTestActive = true
				// start load test routine
				go loadTest(data, quitLoadTestChannel)
			}
		case action_stop_loadtest:
			if isTestActive {
				quitLoadTestChannel <- true
				isTestActive = false
			}
		}
	})

	// pause main thread
	select {}
}

func loadTest(data Data, quitChan chan bool) {
	// input keys - pluginData, tps, duration, action

	var pluginExecutors []plug.PluginManager
	var callLocks = []sync.Mutex{}
	var wg sync.WaitGroup

	// read load test inputs
	loadTestId = data["load_test_id"].(string)
	tps := int(data["tps"].(float64))
	duration := time.Second * time.Duration(int64(data["duration"].(float64)))
	until := time.Now().Add(duration)

	// decode pluginData from base64 to string
	pluginDataBytes, err := base64.StdEncoding.DecodeString(data["plugin_data"].(string))
	if err != nil {
		log.Error("Could not decode bas64 pluginData")
		return
	}

	// create n(tps) no of pluginExecutors and mutexes
	for i := 0; i < tps; i++ {
		pm := plug.NewPluginManager()
		pm.SetBulk(constants.EXPOSED_METHODS)
		_, err := pm.LoadFromString(pluginName, string(pluginDataBytes))
		if err != nil {
			log.Error("Could not load plugin", err)
			return
		}
		// initialize plugin executors and their corresponding mutexes
		callLocks = append(callLocks, sync.Mutex{})
		pluginExecutors = append(pluginExecutors, pm)
	}
	log.Infof("Created %d pluginExecutors", len(pluginExecutors))

	// run load test for given duration
	log.Infof("Starting Load Test, tps %d duration %v", tps, duration)
	for time.Now().Before(until) {
		select {
		case <-quitChan:
			log.Info("Load Test Failed")
			return
		default:
			// use data to trigger x tps
			for i := 0; i < tps; i++ {
				// create routine and trigger Run method in plugin to run transaction
				// note latency, response and status code

				go func(quitChan chan bool, index int) {
					defer wg.Done()
					wg.Add(1)

					var output string

					callLocks[index].Lock()
					err := pluginExecutors[index].CallUnmarshal(&output, pluginName+".Run")
					if err != nil {
						// we got issue in plugin execution, stop load test
						log.Errorf("Error while executing Run in plugin %v", err)
						quitChan <- true
					}
					callLocks[index].Unlock()

					log.Debugf("loadtest output %d %v", index, output)

					// save result in loadTestResults
					loadTestResultLock.Lock()
					loadTestResults = append(loadTestResults, output)
					loadTestResultLock.Unlock()

				}(quitChan, i)
			}
		}
		// create transactions on per second basis
		time.Sleep(time.Second)
	}
	// wait until all routines are done
	wg.Wait()
	isTestActive = false
	log.Info("Load Test Completed")
}
