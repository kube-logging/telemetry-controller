// Copyright © 2024 Kube logging authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package processor

import "github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"

func GenerateProcessorMemoryLimiter(memoryLimiter v1alpha1.MemoryLimiter) map[string]any {
	memoryLimiterResult := make(map[string]any)
	memoryLimiterResult["check_interval"] = memoryLimiter.CheckInterval.String()
	if memoryLimiter.MemoryLimitMiB != 0 {
		memoryLimiterResult["limit_mib"] = memoryLimiter.MemoryLimitMiB
	}
	if memoryLimiter.MemorySpikeLimitMiB != 0 {
		memoryLimiterResult["spike_limit_mib"] = memoryLimiter.MemorySpikeLimitMiB
	}
	if memoryLimiter.MemoryLimitPercentage != 0 {
		memoryLimiterResult["limit_percentage"] = memoryLimiter.MemoryLimitPercentage
	}
	if memoryLimiter.MemorySpikePercentage != 0 {
		memoryLimiterResult["spike_limit_percentage"] = memoryLimiter.MemorySpikePercentage
	}

	return memoryLimiterResult
}
