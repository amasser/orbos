package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gopkg.in/raintank/schema.v1"
)

func graphite(orbID, cloudURL, cloudKey, branch string, test func(orbconfig string) error) func(orbconfig string) error {

	send := func(value float64) {
		if err := sendGraphiteStatus(orbID, cloudURL, cloudKey, branch, value); err != nil {
			panic(err)
		}
	}

	return func(orbconfig string) error {
		send(0.5)
		err := test(orbconfig)
		var value float64
		if err == nil {
			value = 1
		}
		send(value)
		return err
	}
}

func sendGraphiteStatus(orbID, cloudURL, cloudKey, branch string, value float64) error {

	name := fmt.Sprintf("e2e.%s.%s", orbID, branch)

	metrics := schema.MetricDataArray{&schema.MetricData{
		Name:     name,
		Interval: 10,
		Value:    value,
		Time:     time.Now().Unix(),
		Mtype:    "gauge",
	}}

	// encode as json
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	client := &http.Client{}

	req, err := http.NewRequest("POST", cloudURL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Bearer "+cloudKey)
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	buf := make([]byte, 4096)
	n, err := resp.Body.Read(buf)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sending metric to graphana cloud graphite api at %s failed with status %s and response %s", cloudURL, resp.Status, string(buf[:n]))
	}
	fmt.Println("Metric", name, "with value", value, "sent to grafana cloud graphite api")
	return nil
}
