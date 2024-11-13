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

import (
	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
)

func GenerateBatchProcessor(batchProcessor v1alpha1.Batch) map[string]any {
	batchProcessorResult := make(map[string]any)
	batchProcessorResult["timeout"] = batchProcessor.Timeout.String()
	if batchProcessor.SendBatchSize != 0 {
		batchProcessorResult["send_batch_size"] = batchProcessor.SendBatchSize
	}
	if batchProcessor.SendBatchMaxSize != 0 {
		batchProcessorResult["send_batch_max_size"] = batchProcessor.SendBatchMaxSize
	}
	if len(batchProcessor.MetadataKeys) != 0 {
		batchProcessorResult["metadata_keys"] = batchProcessor.MetadataKeys
	}
	if batchProcessor.MetadataCardinalityLimit != 0 {
		batchProcessorResult["metadata_cardinality_limit"] = batchProcessor.MetadataCardinalityLimit
	}

	return batchProcessorResult
}
