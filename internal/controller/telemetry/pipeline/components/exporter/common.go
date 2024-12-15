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

package exporter

import (
	"time"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/utils"
)

type queueWrapper struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled *bool `json:"enabled,omitempty"`

	// NumConsumers is the number of consumers from the queue.
	NumConsumers *int `json:"num_consumers,omitempty"`

	// QueueSize is the maximum number of batches allowed in queue at a given time.
	// Default value is 100.
	QueueSize *int `json:"queue_size,omitempty"`

	// If Storage is not empty, enables the persistent storage and uses the component specified
	// as a storage extension for the persistent queue.
	Storage *string `json:"storage,omitempty"`
}

func (q *queueWrapper) setDefaultQueueSettings(apiQueueSettings *v1alpha1.QueueSettings) {
	q.Enabled = utils.ToPtr(true)
	q.QueueSize = utils.ToPtr(100)

	if apiQueueSettings != nil {
		if apiQueueSettings.NumConsumers != nil {
			q.NumConsumers = apiQueueSettings.NumConsumers
		}
		if apiQueueSettings.QueueSize != nil {
			q.QueueSize = apiQueueSettings.QueueSize
		}
	}
}

type backOffWrapper struct {
	// Enabled indicates whether to retry sending the batch to the consumerSender in case of a failure.
	Enabled *bool `json:"enabled,omitempty"`

	// InitialInterval the time to wait after the first failure before retrying.
	InitialInterval *time.Duration `json:"initial_interval,omitempty"`

	// RandomizationFactor is a random factor used to calculate next backoffs
	// Randomized interval = RetryInterval * (1 ± RandomizationFactor)
	RandomizationFactor *string `json:"randomization_factor,omitempty"`

	// Multiplier is the value multiplied by the backoff interval bounds
	Multiplier *string `json:"multiplier,omitempty"`

	// MaxInterval is the upper bound on backoff interval. Once this value is reached the delay between
	// consecutive retries will always be `MaxInterval`.
	MaxInterval *time.Duration `json:"max_interval,omitempty"`

	// MaxElapsedTime is the maximum amount of time (including retries) spent trying to send a request/batch.
	// Once this value is reached, the data is discarded. If set to 0, the retries are never stopped.
	// Default value is 0 to ensure that the data is never discarded.
	MaxElapsedTime *time.Duration `json:"max_elapsed_time,omitempty"`
}

func (b *backOffWrapper) setDefaultBackOffConfig(apiBackOffConfig *v1alpha1.BackOffConfig) {
	b.Enabled = utils.ToPtr(true)
	b.MaxElapsedTime = utils.ToPtr(0 * time.Second)

	if apiBackOffConfig != nil {
		if apiBackOffConfig.InitialInterval != nil {
			b.InitialInterval = apiBackOffConfig.InitialInterval
		}
		if apiBackOffConfig.RandomizationFactor != nil {
			b.RandomizationFactor = apiBackOffConfig.RandomizationFactor
		}
		if apiBackOffConfig.Multiplier != nil {
			b.Multiplier = apiBackOffConfig.Multiplier
		}
		if apiBackOffConfig.MaxInterval != nil {
			b.MaxInterval = apiBackOffConfig.MaxInterval
		}
		if apiBackOffConfig.MaxElapsedTime != nil {
			b.MaxElapsedTime = apiBackOffConfig.MaxElapsedTime
		}
	}
}
