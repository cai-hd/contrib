/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"k8s.io/kubernetes/test/e2e/perftype"
)

func stripCount(data *perftype.DataItem) {
	delete(data.Labels, "Count")
}

func parseResponsivenessData(data []byte, buildNumber int, job string, testName string, result TestToBuildData) {
	build := fmt.Sprintf("%d", buildNumber)
	obj := perftype.PerfData{}
	if err := json.Unmarshal(data, &obj); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing JSON in build %d: %v %s\n", buildNumber, err, string(data))
		return
	}
	if _, found := result[testName]; !found {
		result[testName] = BuildData{Job: job, Version: obj.Version, Builds: map[string][]perftype.DataItem{}}
	}
	if result[testName].Version == obj.Version {
		for i := range obj.DataItems {
			stripCount(&obj.DataItems[i])
			result[testName].Builds[build] = append(result[testName].Builds[build], obj.DataItems[i])
		}
	}
}

type ResourceUsagePercentiles map[string][]ResourceUsages

type ResourceUsages struct {
	Name   string  `json:"Name"`
	Cpu    float64 `json:"Cpu"`
	Memory int     `json:"Mem"`
}

type ResourceUsage struct {
	Cpu    float64
	Memory float64
}
type UsageAtPercentiles map[string]ResourceUsage
type PodNameToUsage map[string]UsageAtPercentiles

func parseResourceUsageData(data []byte, buildNumber int, job string, testName string, result TestToBuildData) {
	build := fmt.Sprintf("%d", buildNumber)
	var obj ResourceUsagePercentiles
	if err := json.Unmarshal(data, &obj); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing JSON in build %d: %v %s\n", buildNumber, err, string(data))
		return
	}
	if _, found := result[testName]; !found {
		result[testName] = BuildData{Job: job, Version: "v1", Builds: map[string][]perftype.DataItem{}}
	}
	usage := make(PodNameToUsage)
	for percentile, items := range obj {
		for _, item := range items {
			name := RemoveDisambiguationInfixes(item.Name)
			if _, ok := usage[name]; !ok {
				usage[name] = make(UsageAtPercentiles)
			} // TODO(porridge): keep max usage?
			usage[name][percentile] = ResourceUsage{float64(item.Cpu), float64(item.Memory)}
		}
	}
	for podName, usageAtPercentiles := range usage {
		cpu := perftype.DataItem{Unit: "cores", Labels: map[string]string{"PodName": podName, "Resource": "CPU"}, Data: make(map[string]float64)}
		memory := perftype.DataItem{Unit: "MiB", Labels: map[string]string{"PodName": podName, "Resource": "memory"}, Data: make(map[string]float64)}
		for percentile, usage := range usageAtPercentiles {
			cpu.Data[percentile] = usage.Cpu
			memory.Data[percentile] = usage.Memory / (1024 * 1024)
		}
		result[testName].Builds[build] = append(result[testName].Builds[build], cpu)
		result[testName].Builds[build] = append(result[testName].Builds[build], memory)
	}
}
